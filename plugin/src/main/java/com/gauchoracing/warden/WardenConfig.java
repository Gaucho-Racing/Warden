package com.gauchoracing.warden;

import java.time.Duration;
import org.bukkit.configuration.file.FileConfiguration;

/** Typed view of config.yml. */
public record WardenConfig(
        String baseUrl,
        String token,
        Duration timeout,
        Duration syncInterval,
        boolean confinementEnabled,
        double confinementRadius,
        Duration confinementReminder) {

    public static WardenConfig from(FileConfiguration c) {
        return new WardenConfig(
                c.getString("warden.base-url", "http://localhost:10310"),
                c.getString("warden.token", ""),
                Duration.ofSeconds(c.getLong("warden.timeout-seconds", 5)),
                Duration.ofSeconds(c.getLong("sync.interval-seconds", 300)),
                c.getBoolean("confinement.enabled", true),
                c.getDouble("confinement.radius", 32),
                Duration.ofSeconds(c.getLong("confinement.reminder-seconds", 2)));
    }
}
