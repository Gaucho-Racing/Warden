package com.gauchoracing.warden.staff;

import com.destroystokyo.paper.event.server.PaperServerListPingEvent;
import com.gauchoracing.warden.Permissions;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.EntityTargetLivingEntityEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import org.bukkit.metadata.FixedMetadataValue;
import org.bukkit.plugin.Plugin;

/**
 * {@code /vanish}: hidden from everyone without {@link Permissions#VANISH_SEE},
 * from the tab list and from the server list, and left alone by mobs.
 *
 * <p>Vanish lasts one session. Invulnerability is saved with the player, so
 * everything set here is undone on quit, and again on join in case the
 * server stopped while someone was vanished.
 */
public final class VanishManager implements CommandExecutor, Listener {

    /** The key TAB and most other plugins read to treat a player as vanished. */
    private static final String METADATA_KEY = "vanished";

    private final Plugin plugin;
    private final Set<UUID> vanished = ConcurrentHashMap.newKeySet();

    public VanishManager(Plugin plugin) {
        this.plugin = plugin;
    }

    public boolean isVanished(UUID uuid) {
        return vanished.contains(uuid);
    }

    @Override
    public boolean onCommand(CommandSender sender, Command command, String label, String[] args) {
        if (!(sender instanceof Player player)) {
            sender.sendMessage(Component.text("Only players can vanish.", NamedTextColor.RED));
            return true;
        }
        if (vanished.contains(player.getUniqueId())) {
            reveal(player);
            player.sendMessage(Component.text("You are visible again", NamedTextColor.GRAY));
        } else {
            vanish(player);
            player.sendMessage(Component.text("You are vanished", NamedTextColor.GRAY));
        }
        return true;
    }

    private void vanish(Player player) {
        vanished.add(player.getUniqueId());
        player.setMetadata(METADATA_KEY, new FixedMetadataValue(plugin, true));
        for (Player other : plugin.getServer().getOnlinePlayers()) {
            if (!other.equals(player) && !other.hasPermission(Permissions.VANISH_SEE)) {
                other.hidePlayer(plugin, player);
            }
        }
        setUntouchable(player, true);
    }

    private void reveal(Player player) {
        vanished.remove(player.getUniqueId());
        player.removeMetadata(METADATA_KEY, plugin);
        for (Player other : plugin.getServer().getOnlinePlayers()) {
            other.showPlayer(plugin, player);
        }
        setUntouchable(player, false);
    }

    private static void setUntouchable(Player player, boolean untouchable) {
        player.setInvulnerable(untouchable);
        player.setCanPickupItems(!untouchable);
        player.setCollidable(!untouchable);
        player.setSleepingIgnored(untouchable);
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onJoin(PlayerJoinEvent event) {
        Player joined = event.getPlayer();
        setUntouchable(joined, false);
        if (joined.hasPermission(Permissions.VANISH_SEE)) {
            return;
        }
        for (UUID id : vanished) {
            Player hidden = plugin.getServer().getPlayer(id);
            if (hidden != null) {
                joined.hidePlayer(plugin, hidden);
            }
        }
    }

    /** A quit message would announce someone nobody knew was online. */
    @EventHandler
    public void onQuit(PlayerQuitEvent event) {
        Player player = event.getPlayer();
        if (vanished.contains(player.getUniqueId())) {
            event.quitMessage(null);
            reveal(player);
        }
    }

    @EventHandler(ignoreCancelled = true)
    public void onTarget(EntityTargetLivingEntityEvent event) {
        if (event.getTarget() instanceof Player player && vanished.contains(player.getUniqueId())) {
            event.setCancelled(true);
        }
    }

    @EventHandler
    public void onServerListPing(PaperServerListPingEvent event) {
        event.getListedPlayers().removeIf(info -> vanished.contains(info.id()));
        long online = vanished.stream().filter(id -> plugin.getServer().getPlayer(id) != null).count();
        event.setNumPlayers(Math.max(0, event.getNumPlayers() - (int) online));
    }
}
