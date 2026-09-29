package com.gauchoracing.warden.permissions;

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.logging.Logger;

/**
 * Last known good permissions, persisted across restarts.
 *
 * <p>This exists for the Warden-is-unreachable case. Stripping everyone's
 * access the moment the service blips would be worse than briefly honouring
 * a stale grant, so within the staleness window a player keeps what they
 * last had. Past it the entry is discarded and they are treated as unlinked
 * — an indefinitely stale grant is how a revoked admin keeps their
 * permissions forever.
 */
public final class PermissionCache {

    private record Entry(List<String> groups, long resolvedAtEpochSecond) {}

    private static final Gson GSON = new Gson();

    private final Path file;
    private final Duration maxStale;
    private final Logger log;
    private final Map<UUID, Entry> entries = new ConcurrentHashMap<>();

    public PermissionCache(Path file, Duration maxStale, Logger log) {
        this.file = file;
        this.maxStale = maxStale;
        this.log = log;
    }

    public void load() {
        if (!Files.exists(file)) {
            return;
        }
        try {
            String json = Files.readString(file);
            Map<String, Entry> raw =
                    GSON.fromJson(json, new TypeToken<Map<String, Entry>>() {}.getType());
            if (raw == null) {
                return;
            }
            raw.forEach((uuid, entry) -> entries.put(UUID.fromString(uuid), entry));
            log.info("Warden: loaded " + entries.size() + " cached permission sets");
        } catch (IOException | RuntimeException e) {
            // A corrupt cache is not worth failing startup over; the next
            // successful sync rebuilds it.
            log.warning("Warden: could not read permission cache: " + e.getMessage());
        }
    }

    public void save() {
        try {
            Files.createDirectories(file.getParent());
            Map<String, Entry> raw = new ConcurrentHashMap<>();
            entries.forEach((uuid, entry) -> raw.put(uuid.toString(), entry));
            Files.writeString(file, GSON.toJson(raw));
        } catch (IOException e) {
            log.warning("Warden: could not write permission cache: " + e.getMessage());
        }
    }

    public void put(UUID uuid, List<String> groups) {
        entries.put(uuid, new Entry(List.copyOf(groups), Instant.now().getEpochSecond()));
    }

    /** Returns the cached groups, or null when absent or too stale to trust. */
    public List<String> get(UUID uuid) {
        Entry entry = entries.get(uuid);
        if (entry == null) {
            return null;
        }
        Instant resolvedAt = Instant.ofEpochSecond(entry.resolvedAtEpochSecond());
        if (Duration.between(resolvedAt, Instant.now()).compareTo(maxStale) > 0) {
            entries.remove(uuid);
            return null;
        }
        return entry.groups();
    }
}
