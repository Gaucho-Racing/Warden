package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.PlayerState;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.Location;
import org.bukkit.event.EventHandler;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerMoveEvent;

/**
 * Keeps unlinked players near spawn.
 *
 * <p>Deliberately does not change game mode. Forcing adventure would strip
 * creative from an unlinked admin and there is no reliable way to know what
 * to restore afterwards; confining movement achieves the same thing without
 * mutating state Warden does not own.
 */
public final class ConfinementListener implements Listener {

    private final PlayerState state;
    private final double radius;
    private final boolean enabled;

    public ConfinementListener(PlayerState state, double radius, boolean enabled) {
        this.state = state;
        this.radius = radius;
        this.enabled = enabled;
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
        Location spawn = event.getPlayer().getWorld().getSpawnLocation();
        Location to = event.getTo();
        if (!to.getWorld().equals(spawn.getWorld())
                || to.distanceSquared(spawn) > radius * radius) {
            event.setCancelled(true);
            event.getPlayer()
                    .sendActionBar(Component.text(
                            "Link your account to leave spawn", NamedTextColor.RED));
        }
    }
}
