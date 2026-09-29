package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.PlayerState;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.Location;
import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerMoveEvent;

/**
 * Keeps unlinked players near spawn until they link.
 *
 * <p>Deliberately does not change game mode. Forcing adventure would strip
 * creative from an unlinked admin and there is nothing reliable to restore
 * afterwards; confining movement achieves the same thing without mutating
 * state Warden does not own.
 */
public final class ConfinementListener implements Listener {

    private final Server server;
    private final PlayerState state;
    private final double radius;
    private final boolean enabled;
    private final long reminderMillis;
    private final Map<UUID, Long> lastReminder = new ConcurrentHashMap<>();

    public ConfinementListener(
            Server server,
            PlayerState state,
            double radius,
            boolean enabled,
            java.time.Duration reminderInterval) {
        this.server = server;
        this.state = state;
        this.radius = radius;
        this.enabled = enabled;
        this.reminderMillis = reminderInterval.toMillis();
    }

    /**
     * Spawn is always the primary world's, never the player's current world.
     * Using the current world would confine somebody in the Nether to Nether
     * spawn, making a portal an escape hatch.
     */
    private Location spawn() {
        return server.getWorlds().get(0).getSpawnLocation();
    }

    /**
     * Somebody can arrive already outside the boundary — they logged out far
     * away and were unlinked in the meantime, or the radius was shrunk. Move
     * them in rather than leaving them stranded against an invisible wall.
     */
    @EventHandler
    public void onJoin(PlayerJoinEvent event) {
        if (!enabled) {
            return;
        }
        Player player = event.getPlayer();
        if (!state.isUnlinked(player.getUniqueId()) || withinBounds(player.getLocation())) {
            return;
        }
        player.teleportAsync(spawn());
    }

    @EventHandler(ignoreCancelled = true)
    public void onMove(PlayerMoveEvent event) {
        if (!enabled || !state.isUnlinked(event.getPlayer().getUniqueId())) {
            return;
        }
        // Fires on look-only movement too; skip those cheaply.
        if (!event.hasChangedPosition()) {
            return;
        }

        Location to = event.getTo();
        if (withinBounds(to)) {
            return;
        }

        // Only block movement that takes them further out. Cancelling every
        // out-of-bounds move would freeze anyone already outside, including
        // when they are walking back toward spawn.
        Location from = event.getFrom();
        if (movingInward(from, to)) {
            return;
        }

        event.setCancelled(true);
        remind(event.getPlayer());
    }

    private boolean withinBounds(Location location) {
        Location spawn = spawn();
        return location.getWorld().equals(spawn.getWorld())
                && location.distanceSquared(spawn) <= radius * radius;
    }

    private boolean movingInward(Location from, Location to) {
        Location spawn = spawn();
        if (!from.getWorld().equals(spawn.getWorld()) || !to.getWorld().equals(spawn.getWorld())) {
            return false;
        }
        return to.distanceSquared(spawn) < from.distanceSquared(spawn);
    }

    /** Throttled — onMove fires every tick and an action bar per tick is noise. */
    private void remind(Player player) {
        long now = System.currentTimeMillis();
        Long previous = lastReminder.get(player.getUniqueId());
        if (previous != null && now - previous < reminderMillis) {
            return;
        }
        lastReminder.put(player.getUniqueId(), now);
        player.sendActionBar(
                Component.text("Link your account to leave spawn", NamedTextColor.RED));
    }
}
