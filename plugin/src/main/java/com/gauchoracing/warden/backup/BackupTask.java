package com.gauchoracing.warden.backup;

import com.gauchoracing.warden.WardenPlugin;
import java.io.BufferedOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.file.Files;
import java.nio.file.LinkOption;
import java.nio.file.Path;
import java.nio.file.attribute.BasicFileAttributes;
import java.time.Duration;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Level;
import java.util.stream.Stream;
import org.bukkit.World;

/**
 * Archives the server directory and uploads it straight to object storage.
 *
 * <p>The world lives on a volume mounted only into this pod, so the game
 * server is the only party that can read it — Warden orchestrates but never
 * touches the bytes. It hands over a presigned URL and the archive goes from
 * here to storage directly, through neither Warden nor Depot.
 *
 * <p>The archive is staged on disk rather than streamed, because a presigned
 * PUT needs a Content-Length up front and the compressed size is not known
 * until the compression is done.
 */
public final class BackupTask {

    /** Never archive the staging directory into its own archive. */
    private static final String STAGING_DIR = "warden-backups";
    /** Held open by the running server; restoring it would confuse the next boot. */
    private static final String LOCK_FILE = "session.lock";

    /**
     * Headroom demanded before starting. Region files are already
     * zlib-compressed internally, so gzip typically gets them to 50-70% —
     * half the uncompressed total plus a margin is a conservative floor that
     * still refuses to fill the volume the world itself lives on.
     */
    private static final double SIZE_ESTIMATE_RATIO = 0.5;
    private static final long SIZE_ESTIMATE_FLOOR = 256L * 1024 * 1024;

    private final WardenPlugin plugin;
    private final Duration uploadTimeout;
    private final List<String> excludes;

    /** One at a time. A second run would archive the first one's staging file. */
    private final AtomicBoolean running = new AtomicBoolean();

    /**
     * A platform thread, not a virtual one: this is minutes of continuous
     * deflate, which would pin a carrier thread for the duration.
     */
    private final ExecutorService worker = Executors.newSingleThreadExecutor(
            Thread.ofPlatform().name("warden-backup").daemon().factory());

    /** No Authorization header ever goes to the presigned URL; it carries its own. */
    private final HttpClient http;

    public BackupTask(WardenPlugin plugin, Duration connectTimeout, Duration uploadTimeout,
            List<String> excludes) {
        this.plugin = plugin;
        this.uploadTimeout = uploadTimeout;
        this.excludes = List.copyOf(excludes);
        this.http = HttpClient.newBuilder().connectTimeout(connectTimeout).build();
    }

    public void shutdown() {
        worker.shutdownNow();
        http.close();
    }

    /**
     * Called on the bridge thread when Warden issues a backup command.
     * Returns immediately; every path that reports back does so off-thread,
     * because reporting retries and must never stall the bridge or the tick
     * loop.
     */
    public void start(String jobId, String uploadUrl, String method, String contentType,
            String fileName) {
        if (!running.compareAndSet(false, true)) {
            plugin.runAsync(() -> reportFailure(jobId, "a backup is already running on the game server"));
            return;
        }
        // Quiescing touches worlds, which is main-thread only. The heavy work
        // hops to the worker from there.
        plugin.getServer().getGlobalRegionScheduler().run(plugin, task -> {
            try {
                quiesce();
            } catch (RuntimeException e) {
                resumeAutoSave();
                running.set(false);
                plugin.runAsync(() -> reportFailure(jobId,
                        "could not flush the world to disk: " + describe(e)));
                return;
            }
            worker.execute(() -> run(jobId, uploadUrl, method, contentType, fileName));
        });
    }

    /**
     * Flushes everything to disk and stops the server writing to it.
     * Without this the archive catches region files mid-write, and a restore
     * loses whatever the autosave was part way through.
     */
    private void quiesce() {
        plugin.getServer().savePlayers();
        for (World world : plugin.getServer().getWorlds()) {
            world.setAutoSave(false);
            world.save();
        }
    }

    private void resumeAutoSave() {
        plugin.getServer().getGlobalRegionScheduler().run(plugin, task -> {
            for (World world : plugin.getServer().getWorlds()) {
                world.setAutoSave(true);
            }
        });
    }

    private void run(String jobId, String uploadUrl, String method, String contentType,
            String fileName) {
        Path root = plugin.getServer().getWorldContainer().toPath().toAbsolutePath().normalize();
        Path staging = root.resolve(STAGING_DIR);
        Path archive = staging.resolve(fileName);
        try {
            long archiveMillis = archive(root, staging, archive);
            long size = Files.size(archive);
            plugin.getLogger().info("Warden: archived " + size + " bytes in " + archiveMillis
                    + "ms, uploading");
            reportProgress(jobId, archiveMillis, size);

            long started = System.nanoTime();
            upload(archive, uploadUrl, method, contentType);
            reportSuccess(jobId, size, archiveMillis, millisSince(started));
        } catch (Exception e) {
            plugin.getLogger().log(Level.WARNING, "Warden: backup failed", e);
            reportFailure(jobId, describe(e));
        } finally {
            deleteQuietly(archive);
            running.set(false);
        }
    }

    /** Returns how long the archive took. Always unfreezes the world. */
    private long archive(Path root, Path staging, Path archive) throws IOException {
        try {
            Files.createDirectories(staging);
            // A run killed mid-upload leaves its staging file behind. Clearing
            // here, rather than only after a success, is what stops the volume
            // filling up one abandoned backup at a time.
            clearStaging(staging);

            long started = System.nanoTime();
            List<Path> files = collect(root, staging);
            requireSpace(staging, files);
            writeArchive(root, files, archive);
            return millisSince(started);
        } finally {
            // The world only has to stay frozen for the archive, not for the
            // upload, so it resumes as early as it possibly can.
            resumeAutoSave();
        }
    }

    /**
     * Lists every regular file to archive, in a stable order.
     *
     * <p>Walked once up front rather than streamed into the archive, because
     * the free-space check needs the total before a single byte is written —
     * discovering there is no room half way through would leave the volume
     * the world lives on full.
     */
    private List<Path> collect(Path root, Path staging) throws IOException {
        List<Path> files = new ArrayList<>();
        try (Stream<Path> walk = Files.walk(root)) {
            walk.filter(path -> !path.equals(root))
                    .filter(path -> !path.startsWith(staging))
                    .filter(path -> !isExcluded(root.relativize(path)))
                    // Symlinks, sockets and fifos have no meaning restored into
                    // a fresh container, and a followed symlink can loop.
                    .filter(path -> Files.isRegularFile(path, LinkOption.NOFOLLOW_LINKS))
                    .sorted(Comparator.comparing(Path::toString))
                    .forEach(files::add);
        }
        return files;
    }

    private boolean isExcluded(Path relative) {
        String path = relative.toString().replace('\\', '/');
        if (relative.getFileName().toString().equals(LOCK_FILE)) {
            return true;
        }
        for (String exclude : excludes) {
            if (path.equals(exclude) || path.startsWith(exclude + "/")) {
                return true;
            }
        }
        return false;
    }

    private void requireSpace(Path staging, List<Path> files) throws IOException {
        long total = 0;
        for (Path file : files) {
            try {
                total += Files.size(file);
            } catch (IOException ignored) {
                // Vanished between the walk and here; it will be skipped below too.
            }
        }
        long needed = Math.max((long) (total * SIZE_ESTIMATE_RATIO), SIZE_ESTIMATE_FLOOR);
        long usable = Files.getFileStore(staging).getUsableSpace();
        if (usable < needed) {
            throw new IOException(String.format(Locale.ROOT,
                    "not enough free space to stage the archive: %s of server data needs about %s "
                            + "free, but only %s is left on the data volume",
                    humanBytes(total), humanBytes(needed), humanBytes(usable)));
        }
    }

    private void writeArchive(Path root, List<Path> files, Path archive) throws IOException {
        try (OutputStream sink = Files.newOutputStream(archive);
                TarGzWriter tar = new TarGzWriter(new BufferedOutputStream(sink, 64 * 1024))) {
            for (Path file : files) {
                BasicFileAttributes attributes;
                try {
                    attributes = Files.readAttributes(file, BasicFileAttributes.class);
                } catch (IOException vanished) {
                    // A log rotated out from under the walk. Skipping one file
                    // is better than failing a whole backup over it.
                    continue;
                }
                String name = root.relativize(file).toString().replace('\\', '/');
                tar.addFile(name, file, attributes.size(),
                        attributes.lastModifiedTime().to(TimeUnit.SECONDS));
            }
        }
    }

    private void upload(Path archive, String uploadUrl, String method, String contentType)
            throws IOException, InterruptedException {
        HttpRequest request = HttpRequest.newBuilder(URI.create(uploadUrl))
                // Echoed exactly: the content type is part of the presignature,
                // and storage rejects a PUT whose header does not match it.
                .header("Content-Type", contentType)
                .timeout(uploadTimeout)
                .method(method == null || method.isBlank() ? "PUT" : method,
                        HttpRequest.BodyPublishers.ofFile(archive))
                .build();
        HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());
        int status = response.statusCode();
        if (status < 200 || status >= 300) {
            throw new IOException("storage returned " + status + ": " + truncate(response.body()));
        }
    }

    private void reportProgress(String jobId, long archiveMillis, long size) {
        try {
            plugin.client().reportBackupProgress(jobId, archiveMillis, size);
        } catch (Exception e) {
            // Cosmetic. The job still completes on the final report.
            plugin.getLogger().fine("Warden: could not report backup progress: " + e.getMessage());
        }
    }

    private void reportSuccess(String jobId, long size, long archiveMillis, long uploadMillis) {
        complete(jobId, true, size, archiveMillis, uploadMillis, "");
    }

    private void reportFailure(String jobId, String error) {
        complete(jobId, false, 0, 0, 0, error);
    }

    /**
     * The final report is the only thing that moves the job out of running,
     * so it is retried: losing it would leave the job hanging until Warden's
     * timeout, blocking every backup until then.
     */
    private void complete(String jobId, boolean ok, long size, long archiveMillis,
            long uploadMillis, String error) {
        for (int attempt = 1; attempt <= 3; attempt++) {
            try {
                plugin.client().completeBackup(jobId, ok, size, archiveMillis, uploadMillis, error);
                return;
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return;
            } catch (Exception e) {
                plugin.getLogger().warning("Warden: could not report backup result (attempt "
                        + attempt + "): " + e.getMessage());
                try {
                    Thread.sleep(Duration.ofSeconds(2L * attempt));
                } catch (InterruptedException interrupted) {
                    Thread.currentThread().interrupt();
                    return;
                }
            }
        }
    }

    private void clearStaging(Path staging) throws IOException {
        try (Stream<Path> entries = Files.list(staging)) {
            entries.forEach(BackupTask::deleteQuietly);
        }
    }

    private static void deleteQuietly(Path path) {
        try {
            Files.deleteIfExists(path);
        } catch (IOException ignored) {
            // Nothing useful to do; the next run clears the staging directory.
        }
    }

    private static long millisSince(long startNanos) {
        return (System.nanoTime() - startNanos) / 1_000_000;
    }

    private static String describe(Throwable e) {
        String message = e.getMessage();
        return message == null || message.isBlank() ? e.getClass().getSimpleName() : message;
    }

    private static String truncate(String body) {
        if (body == null) {
            return "";
        }
        return body.length() <= 200 ? body : body.substring(0, 200) + "…";
    }

    private static String humanBytes(long bytes) {
        if (bytes < 1024) {
            return bytes + " B";
        }
        String units = "KMGT";
        double value = bytes;
        int unit = -1;
        while (value >= 1024 && unit < units.length() - 1) {
            value /= 1024;
            unit++;
        }
        return String.format(Locale.ROOT, "%.1f %cB", value, units.charAt(unit));
    }
}
