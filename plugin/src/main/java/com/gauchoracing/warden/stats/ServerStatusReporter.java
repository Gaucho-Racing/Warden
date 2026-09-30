package com.gauchoracing.warden.stats;

import com.gauchoracing.warden.Errors;
import com.gauchoracing.warden.WardenPlugin;
import com.gauchoracing.warden.staff.VanishManager;
import java.lang.management.ManagementFactory;
import java.util.concurrent.atomic.AtomicBoolean;
import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;

/**
 * Reports server health to Warden every minute and shortly after joins and
 * quits, so the Discord topic and presence follow the player count closely.
 * Warden keeps every sample for analytics whether or not the bridge is on.
 */
public final class ServerStatusReporter implements Listener {

    /** Five seconds, so a burst of joins or a mass disconnect is one report. */
    private static final long SETTLE_TICKS = 100;

    private final WardenPlugin plugin;
    private final VanishManager vanish;
    private final long startedAt = ManagementFactory.getRuntimeMXBean().getStartTime();
    private final AtomicBoolean pending = new AtomicBoolean();

    record ServerStatus(int online, int maxPlayers, int uniquePlayers, double tps, double mspt,
            long startedAt) {}

    public ServerStatusReporter(WardenPlugin plugin, VanishManager vanish) {
        this.plugin = plugin;
        this.vanish = vanish;
    }

    /** Call from the main thread. Sampling is main-thread; the POST is not. */
    public void report() {
        Server server = plugin.getServer();
        int online = 0;
        for (Player player : server.getOnlinePlayers()) {
            if (!vanish.isVanished(player.getUniqueId())) {
                online++;
            }
        }
        ServerStatus status = new ServerStatus(online, server.getMaxPlayers(),
                server.getOfflinePlayers().length, server.getTPS()[0], server.getAverageTickTime(),
                startedAt);
        plugin.runAsync(() -> {
            try {
                plugin.client().reportServerStatus(status);
            } catch (Exception e) {
                plugin.getLogger().fine("Warden: server status report failed: " + Errors.describe(e));
            }
        });
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onJoin(PlayerJoinEvent event) {
        reportSoon();
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onQuit(PlayerQuitEvent event) {
        reportSoon();
    }

    private void reportSoon() {
        if (pending.compareAndSet(false, true)) {
            plugin.getServer().getGlobalRegionScheduler().runDelayed(plugin, task -> {
                pending.set(false);
                report();
            }, SETTLE_TICKS);
        }
    }
}
