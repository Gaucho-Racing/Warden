package com.gauchoracing.warden;

import java.time.Duration;
import org.bukkit.configuration.file.FileConfiguration;

/** Typed view of config.yml. */
public record WardenConfig(
        String baseUrl,
        String token,
        Duration timeout,
        Duration syncInterval,
        boolean statsEnabled,
        Duration statsInterval,
        boolean confinementEnabled,
        double confinementRadius,
        Duration confinementReminder,
        String noAccessMessage,
        boolean bridgeEnabled) {

    private static final String DEFAULT_NO_ACCESS_MESSAGE =
            "You must be a member of the MinecraftPlayers group to play on this server!";

    public static WardenConfig from(FileConfiguration c) {
        return new WardenConfig(
                c.getString("warden.base-url", "http://localhost:10310"),
                c.getString("warden.token", ""),
                Duration.ofSeconds(c.getLong("warden.timeout-seconds", 5)),
                Duration.ofSeconds(c.getLong("sync.interval-seconds", 300)),
                c.getBoolean("stats.enabled", true),
                Duration.ofSeconds(c.getLong("stats.interval-seconds", 600)),
                c.getBoolean("confinement.enabled", true),
                c.getDouble("confinement.radius", 32),
                Duration.ofSeconds(c.getLong("confinement.reminder-seconds", 2)),
                c.getString("confinement.no-access-message", DEFAULT_NO_ACCESS_MESSAGE),
                c.getBoolean("bridge.enabled", true));
    }
}
