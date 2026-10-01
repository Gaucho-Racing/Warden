package com.gauchoracing.warden.api;

import com.google.gson.FieldNamingPolicy;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.UUID;

/**
 * Talks to the Warden service.
 *
 * <p>Every method blocks. None of them may be called from the main server
 * thread — a slow reply would stall the whole server. Callers use
 * {@code AsyncPlayerPreLoginEvent}, which already runs off-thread, or the
 * async scheduler.
 */
public final class WardenClient implements AutoCloseable {

    /** The service returns snake_case; records here are camelCase. */
    private static final Gson GSON = new GsonBuilder()
            .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
            .create();

    /** See completeBackup: long enough that a slow Warden does not lose a report. */
    private static final Duration REPORT_TIMEOUT = Duration.ofSeconds(30);

    private final HttpClient http;
    private final String baseUrl;
    private final String token;
    private final Duration timeout;

    public WardenClient(String baseUrl, String token, Duration timeout) {
        this.baseUrl = baseUrl.replaceAll("/+$", "");
        this.token = token;
        this.timeout = timeout;
        this.http = HttpClient.newBuilder().connectTimeout(timeout).build();
    }

    @Override
    public void close() {
        http.close();
    }

    /** Full desired state: every managed group and every linked player. */
    public SyncSnapshot sync() throws IOException, InterruptedException {
        return get("/api/plugin/sync", SyncSnapshot.class);
    }

    /**
     * One player's desired state. An unlinked player is a 200, not a 404 —
     * see {@link ResolvedPermissions}.
     */
    public ResolvedPermissions player(UUID uuid, String username)
            throws IOException, InterruptedException {
        return get(
                "/api/plugin/players/" + uuid + "?username=" + enc(username),
                ResolvedPermissions.class);
    }

    /** Mints a pending link for a player Mojang has already authenticated. */
    public LinkToken createLinkToken(UUID uuid, String username)
            throws IOException, InterruptedException {
        String body = GSON.toJson(new LinkRequest(uuid.toString(), username));
        return send(
                HttpRequest.newBuilder(URI.create(baseUrl + "/api/plugin/link-tokens"))
                        .header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(body, StandardCharsets.UTF_8)),
                LinkToken.class);
    }

    /**
     * Records that a UUID was seen and refreshes the cached username.
     * Separate from the permission read so that read stays side-effect free
     * and freely retryable.
     */
    public void markSeen(UUID uuid, String username) throws IOException, InterruptedException {
        send(
                HttpRequest.newBuilder(
                                URI.create(
                                        baseUrl
                                                + "/api/plugin/players/"
                                                + uuid
                                                + "/seen?username="
                                                + enc(username)))
                        .POST(HttpRequest.BodyPublishers.noBody()),
                null);
    }

    /** Reports one server health sample; see ServerStatusReporter. */
    public void reportServerStatus(Object status) throws IOException, InterruptedException {
        String body = GSON.toJson(status);
        send(
                HttpRequest.newBuilder(URI.create(baseUrl + "/api/plugin/server/status"))
                        .header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(body, StandardCharsets.UTF_8)),
                null);
    }

    /**
     * Reports a player's statistics. Whole-state, not increments — Minecraft
     * keeps running totals, so a duplicated or replayed report is harmless.
     */
    public void reportStats(UUID uuid, Object report) throws IOException, InterruptedException {
        String body = GSON.toJson(report);
        send(
                HttpRequest.newBuilder(
                                URI.create(baseUrl + "/api/plugin/players/" + uuid + "/stats"))
                        .header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(body, StandardCharsets.UTF_8)),
                null);
    }

    /**
     * Reports that the archive is built and the upload has started, so the
     * portal shows a phase rather than one opaque "running" for minutes.
     */
    public void reportBackupProgress(String jobId, long archiveMillis, long sizeBytes)
            throws IOException, InterruptedException {
        String body = GSON.toJson(new BackupProgress("uploading", archiveMillis, sizeBytes));
        send(
                HttpRequest.newBuilder(URI.create(baseUrl + "/api/plugin/backups/" + enc(jobId) + "/progress"))
                        .header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(body, StandardCharsets.UTF_8)),
                null);
    }

    /**
     * The final word on a backup. Warden verifies the object with Depot
     * rather than trusting the reported size, so this is a notification
     * rather than an assertion.
     */
    public void completeBackup(String jobId, boolean ok, long sizeBytes, long archiveMillis,
            long uploadMillis, String error) throws IOException, InterruptedException {
        String body = GSON.toJson(new BackupResult(ok, sizeBytes, archiveMillis, uploadMillis, error));
        send(
                HttpRequest.newBuilder(URI.create(baseUrl + "/api/plugin/backups/" + enc(jobId) + "/complete"))
                        .header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(body, StandardCharsets.UTF_8)),
                null,
                // The default timeout is tuned for a player waiting on a join.
                // A backup report is worth waiting longer for: losing it leaves
                // the job hanging until Warden's own timeout.
                REPORT_TIMEOUT);
    }

    private <T> T get(String path, Class<T> type) throws IOException, InterruptedException {
        return send(HttpRequest.newBuilder(URI.create(baseUrl + path)).GET(), type);
    }

    private <T> T send(HttpRequest.Builder builder, Class<T> type)
            throws IOException, InterruptedException {
        return send(builder, type, timeout);
    }

    private <T> T send(HttpRequest.Builder builder, Class<T> type, Duration requestTimeout)
            throws IOException, InterruptedException {
        HttpRequest request = builder.header("Authorization", "Bearer " + token)
                .header("Accept", "application/json")
                .timeout(requestTimeout)
                .build();
        HttpResponse<String> response = http.send(request, HttpResponse.BodyHandlers.ofString());
        int status = response.statusCode();
        if (status < 200 || status >= 300) {
            throw new WardenApiException(status, response.body());
        }
        return type == null ? null : GSON.fromJson(response.body(), type);
    }

    private static String enc(String value) {
        return java.net.URLEncoder.encode(value == null ? "" : value, StandardCharsets.UTF_8);
    }

    private record LinkRequest(String uuid, String username) {}

    private record BackupProgress(String phase, long archiveMillis, long sizeBytes) {}

    private record BackupResult(
            boolean ok, long sizeBytes, long archiveMillis, long uploadMillis, String error) {}

    /** Non-2xx from the service. Carries the status so callers can tell 409 from 502. */
    public static final class WardenApiException extends IOException {
        private final int status;

        WardenApiException(int status, String body) {
            super("warden returned " + status + ": " + body);
            this.status = status;
        }

        public int status() {
            return status;
        }
    }
}
