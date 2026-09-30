package com.gauchoracing.warden.stats;

import com.gauchoracing.warden.Errors;
import com.gauchoracing.warden.WardenPlugin;
import org.bukkit.entity.Player;

/**
 * Gathers statistics on the main thread and reports them off it.
 *
 * <p>The split matters: Bukkit entity access is not thread safe, so the
 * gather cannot be async, and an HTTP round trip must never sit on the main
 * thread. Every caller here is already on the main thread when it calls
 * {@link #report}.
 */
public final class StatsReporter {

    private final WardenPlugin plugin;

    public StatsReporter(WardenPlugin plugin) {
        this.plugin = plugin;
    }

    /** Call from the main thread. Returns immediately; the POST is async. */
    public void report(Player player) {
        if (!plugin.config().statsEnabled()) {
            return;
        }
        PlayerStatsReport snapshot = StatsCollector.collect(player);
        java.util.UUID uuid = player.getUniqueId();
        plugin.runAsync(() -> {
            try {
                plugin.client().reportStats(uuid, snapshot);
            } catch (Exception e) {
                // Stats are not load-bearing; losing a report costs nothing
                // beyond a slightly stale portal.
                plugin.getLogger()
                        .fine("Warden: stats report failed for " + snapshot.username() + ": "
                                + Errors.describe(e));
            }
        });
    }

    /** Call from the main thread. Reports everyone currently online. */
    public void reportOnline() {
        if (!plugin.config().statsEnabled()) {
            return;
        }
        for (Player player : plugin.getServer().getOnlinePlayers()) {
            report(player);
        }
    }
}
