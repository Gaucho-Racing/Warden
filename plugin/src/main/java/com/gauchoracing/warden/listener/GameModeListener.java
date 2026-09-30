package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.Permissions;
import net.luckperms.api.LuckPerms;
import net.luckperms.api.event.user.UserDataRecalculateEvent;
import org.bukkit.GameMode;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.plugin.Plugin;

/**
 * Survival is earned through a binding; everyone else plays adventure.
 *
 * <p>server.properties defaults every join to adventure. This promotes anyone
 * holding {@link Permissions#PLAY} to survival, and demotes them again if a
 * sync takes it away. Creative and spectator are left alone so an admin's
 * {@code /gamemode} is not undone mid-session; force-gamemode resets them on
 * their next join anyway.
 */
public final class GameModeListener implements Listener {

    private final Plugin plugin;

    /** LuckPerms drops subscriptions owned by a plugin when it disables. */
    public GameModeListener(Plugin plugin, LuckPerms luckPerms) {
        this.plugin = plugin;
        luckPerms.getEventBus().subscribe(plugin, UserDataRecalculateEvent.class, this::onRecalculate);
    }

    /** MONITOR so force-gamemode, and any other plugin, has already run. */
    @EventHandler(priority = EventPriority.MONITOR)
    public void onJoin(PlayerJoinEvent event) {
        apply(event.getPlayer());
    }

    /**
     * Fires off the main thread whenever a player's permissions change, which
     * is how a Warden sync granting or revoking a group reaches us.
     */
    private void onRecalculate(UserDataRecalculateEvent event) {
        Player player = plugin.getServer().getPlayer(event.getUser().getUniqueId());
        if (player != null) {
            player.getScheduler().run(plugin, task -> apply(player), null);
        }
    }

    private void apply(Player player) {
        GameMode current = player.getGameMode();
        if (current == GameMode.CREATIVE || current == GameMode.SPECTATOR) {
            return;
        }
        GameMode target = player.hasPermission(Permissions.PLAY)
                ? GameMode.SURVIVAL
                : GameMode.ADVENTURE;
        if (current != target) {
            player.setGameMode(target);
        }
    }
}
