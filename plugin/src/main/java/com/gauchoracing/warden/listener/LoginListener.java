package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.PlayerState;
import com.gauchoracing.warden.WardenPlugin;
import com.gauchoracing.warden.api.LinkToken;
import com.gauchoracing.warden.api.ResolvedPermissions;
import java.util.List;
import java.util.UUID;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.event.ClickEvent;
import net.kyori.adventure.text.format.NamedTextColor;
import net.kyori.adventure.text.format.TextDecoration;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.AsyncPlayerPreLoginEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;

/** Resolves permissions at login and hands unlinked players their link. */
public final class LoginListener implements Listener {

    private final WardenPlugin plugin;
    private final PlayerState state;

    public LoginListener(WardenPlugin plugin, PlayerState state) {
        this.plugin = plugin;
        this.state = state;
    }

    /**
     * Resolve and apply during prelogin.
     *
     * <p>This event runs off the main thread, which is the only reason
     * blocking HTTP and blocking LuckPerms calls are legal here. The player
     * is already authenticated by Mojang at this point, so the UUID is
     * trustworthy.
     *
     * <p>Warden being unreachable never denies the login. The player falls
     * back to their cached permissions, and to unlinked if there is no
     * usable cache — a permissions service outage should not be an outage
     * of the game server.
     */
    @EventHandler(priority = EventPriority.LOW)
    public void onPreLogin(AsyncPlayerPreLoginEvent event) {
        UUID uuid = event.getUniqueId();
        String username = event.getName();

        List<String> groups;
        boolean linked;
        try {
            ResolvedPermissions resolved = plugin.client().player(uuid, username);
            linked = resolved.linked();
            groups = resolved.luckpermsGroups();
            plugin.cache().put(uuid, groups);
        } catch (Exception e) {
            List<String> cached = plugin.cache().get(uuid);
            if (cached == null) {
                plugin.getLogger()
                        .warning("Warden: unreachable and no usable cache for " + username
                                + "; joining unlinked (" + e.getMessage() + ")");
                groups = List.of();
                linked = false;
            } else {
                plugin.getLogger()
                        .warning("Warden: unreachable, using cached permissions for " + username
                                + " (" + e.getMessage() + ")");
                groups = cached;
                linked = true;
            }
        }

        try {
            plugin.applier().applyPlayer(uuid, groups, plugin.managedGroupPrefix());
        } catch (Exception e) {
            plugin.getLogger()
                    .severe("Warden: failed to apply permissions for " + username + ": "
                            + e.getMessage());
        }

        if (linked) {
            state.markLinked(uuid);
        } else {
            state.markUnlinked(uuid);
        }
    }

    @EventHandler
    public void onJoin(PlayerJoinEvent event) {
        UUID uuid = event.getPlayer().getUniqueId();
        String username = event.getPlayer().getName();

        plugin.runAsync(() -> {
            try {
                plugin.client().markSeen(uuid, username);
            } catch (Exception e) {
                plugin.getLogger().fine("Warden: markSeen failed for " + username);
            }
        });

        if (state.isUnlinked(uuid)) {
            plugin.runAsync(() -> sendLinkPrompt(event.getPlayer().getUniqueId(), username));
        }
    }

    @EventHandler
    public void onQuit(PlayerQuitEvent event) {
        state.forget(event.getPlayer().getUniqueId());
    }

    /** Mints a token and sends the clickable link. Runs async — it makes a request. */
    public void sendLinkPrompt(UUID uuid, String username) {
        try {
            LinkToken token = plugin.client().createLinkToken(uuid, username);
            Component message = Component.text()
                    .append(Component.text("Your Minecraft account isn't linked yet. ",
                            NamedTextColor.GRAY))
                    .append(Component.text("Click here to link it",
                                    NamedTextColor.GREEN,
                                    TextDecoration.UNDERLINED)
                            .clickEvent(ClickEvent.openUrl(token.url())))
                    .build();
            var player = plugin.getServer().getPlayer(uuid);
            if (player != null) {
                player.sendMessage(message);
            }
        } catch (Exception e) {
            plugin.getLogger()
                    .warning("Warden: could not mint a link token for " + username + ": "
                            + e.getMessage());
        }
    }
}
