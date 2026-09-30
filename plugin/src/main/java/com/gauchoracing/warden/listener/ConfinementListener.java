package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.Permissions;
import com.gauchoracing.warden.PlayerState;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.Location;
import org.bukkit.Server;
import org.bukkit.entity.Entity;
import org.bukkit.entity.Player;
import org.bukkit.entity.Projectile;
import org.bukkit.event.Cancellable;
import org.bukkit.event.Event;
import org.bukkit.event.EventHandler;
import org.bukkit.event.Listener;
import org.bukkit.event.block.BlockBreakEvent;
import org.bukkit.event.block.BlockPlaceEvent;
import org.bukkit.event.entity.EntityDamageByEntityEvent;
import org.bukkit.event.hanging.HangingBreakByEntityEvent;
import org.bukkit.event.player.PlayerArmorStandManipulateEvent;
import org.bukkit.event.player.PlayerBucketEmptyEvent;
import org.bukkit.event.player.PlayerBucketFillEvent;
import org.bukkit.event.player.PlayerInteractEntityEvent;
import org.bukkit.event.player.PlayerInteractEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerMoveEvent;
import org.bukkit.event.player.PlayerPortalEvent;
import org.bukkit.event.player.PlayerTeleportEvent;
import org.bukkit.event.player.PlayerTeleportEvent.TeleportCause;
import org.bukkit.inventory.ItemStack;

/**
 * Keeps anyone without {@link Permissions#PLAY} near spawn, and unable to
 * change the world or hurt anything there. That covers unlinked players and
 * linked players outside every group that grants it; the two only differ in
 * what they are told to do about it. Game mode is GameModeListener's job.
 */
public final class ConfinementListener implements Listener {

    private static final String LEAVE_REMINDER = "Link your account to leave spawn";
    private static final String BUILD_REMINDER = "Link your account to interact with the world";

    private final Server server;
    private final PlayerState state;
    private final double radius;
    private final boolean enabled;
    private final long reminderMillis;
    private final String noAccessMessage;
    private final Map<UUID, Long> lastReminder = new ConcurrentHashMap<>();

    public ConfinementListener(
            Server server,
            PlayerState state,
            double radius,
            boolean enabled,
            java.time.Duration reminderInterval,
            String noAccessMessage) {
        this.server = server;
        this.state = state;
        this.radius = radius;
        this.enabled = enabled;
        this.reminderMillis = reminderInterval.toMillis();
        this.noAccessMessage = noAccessMessage;
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
        if (!isRestricted(player) || withinBounds(player.getLocation())) {
            return;
        }
        player.teleportAsync(spawn());
    }

    @EventHandler(ignoreCancelled = true)
    public void onMove(PlayerMoveEvent event) {
        if (!isRestricted(event.getPlayer())) {
            return;
        }
        // Fires on look-only movement too; skip those cheaply.
        if (!event.hasChangedPosition()) {
            return;
        }
        if (!escapes(event.getFrom(), event.getTo())) {
            return;
        }
        event.setCancelled(true);
        remind(event.getPlayer(), leaveReminder(event.getPlayer()));
    }

    /**
     * Teleports and portals have their own handler lists, so onMove never sees
     * them and an ender pearl or chorus fruit would hop the boundary. Command
     * and plugin teleports are exempt so admins, and our own pull back to
     * spawn, still work.
     */
    @EventHandler(ignoreCancelled = true)
    public void onTeleport(PlayerTeleportEvent event) {
        if (event.getCause() == TeleportCause.COMMAND || event.getCause() == TeleportCause.PLUGIN) {
            return;
        }
        denyEscape(event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onPortal(PlayerPortalEvent event) {
        denyEscape(event);
    }

    private void denyEscape(PlayerTeleportEvent event) {
        if (!isRestricted(event.getPlayer()) || !escapes(event.getFrom(), event.getTo())) {
            return;
        }
        event.setCancelled(true);
        remind(event.getPlayer(), leaveReminder(event.getPlayer()));
    }

    @EventHandler(ignoreCancelled = true)
    public void onBlockBreak(BlockBreakEvent event) {
        deny(event.getPlayer(), event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onBlockPlace(BlockPlaceEvent event) {
        deny(event.getPlayer(), event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onBucketEmpty(PlayerBucketEmptyEvent event) {
        deny(event.getPlayer(), event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onBucketFill(PlayerBucketFillEvent event) {
        deny(event.getPlayer(), event);
    }

    /**
     * Only block clicks, including pressure plates and trampling. Clicking air
     * is left alone, and so is eating while looking at a block, because hunger
     * drains on hard and a confined player still has to eat.
     */
    @EventHandler
    public void onInteract(PlayerInteractEvent event) {
        if (event.getClickedBlock() == null || !isRestricted(event.getPlayer())) {
            return;
        }
        event.setUseInteractedBlock(Event.Result.DENY);
        ItemStack item = event.getItem();
        if (item == null || !item.getType().isEdible()) {
            event.setUseItemInHand(Event.Result.DENY);
        }
        remind(event.getPlayer(), buildReminder(event.getPlayer()));
    }

    @EventHandler(ignoreCancelled = true)
    public void onInteractEntity(PlayerInteractEntityEvent event) {
        deny(event.getPlayer(), event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onArmorStandManipulate(PlayerArmorStandManipulateEvent event) {
        deny(event.getPlayer(), event);
    }

    @EventHandler(ignoreCancelled = true)
    public void onDamage(EntityDamageByEntityEvent event) {
        Player attacker = responsiblePlayer(event.getDamager());
        if (attacker != null) {
            deny(attacker, event);
        }
    }

    @EventHandler(ignoreCancelled = true)
    public void onHangingBreak(HangingBreakByEntityEvent event) {
        Player remover = responsiblePlayer(event.getRemover());
        if (remover != null) {
            deny(remover, event);
        }
    }

    private static Player responsiblePlayer(Entity entity) {
        if (entity instanceof Player player) {
            return player;
        }
        if (entity instanceof Projectile projectile && projectile.getShooter() instanceof Player shooter) {
            return shooter;
        }
        return null;
    }

    private boolean isRestricted(Player player) {
        return enabled && !player.hasPermission(Permissions.PLAY);
    }

    /** Linking is the fix for an unlinked player; a group is the fix for everyone else. */
    private String leaveReminder(Player player) {
        return state.isUnlinked(player.getUniqueId()) ? LEAVE_REMINDER : noAccessMessage;
    }

    private String buildReminder(Player player) {
        return state.isUnlinked(player.getUniqueId()) ? BUILD_REMINDER : noAccessMessage;
    }

    private void deny(Player player, Cancellable event) {
        if (!isRestricted(player)) {
            return;
        }
        event.setCancelled(true);
        remind(player, buildReminder(player));
    }

    /**
     * Only movement that takes them further out counts. Blocking every
     * out-of-bounds move would freeze anyone already outside, including when
     * they are walking back toward spawn.
     */
    private boolean escapes(Location from, Location to) {
        return !withinBounds(to) && !movingInward(from, to);
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
    private void remind(Player player, String message) {
        long now = System.currentTimeMillis();
        Long previous = lastReminder.get(player.getUniqueId());
        if (previous != null && now - previous < reminderMillis) {
            return;
        }
        lastReminder.put(player.getUniqueId(), now);
        player.sendActionBar(Component.text(message, NamedTextColor.RED));
    }
}
