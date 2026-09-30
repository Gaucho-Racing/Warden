package com.gauchoracing.warden.listener;

import com.gauchoracing.warden.Errors;
import com.gauchoracing.warden.Permissions;
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
     * is already authenticated by Mojang, so the UUID is trustworthy.
     *
     * <p>Warden being unreachable never denies the login, but it is treated
     * as unlinked: no managed groups, and confined to spawn. That is a
     * deliberate fail-closed posture — if Warden cannot say who somebody is,
     * nobody roams. An outage therefore pens the whole server at spawn,
     * which is the intended behaviour rather than a side effect.
     */
    @EventHandler(priority = EventPriority.LOW)
    public void onPreLogin(AsyncPlayerPreLoginEvent event) {
        UUID uuid = event.getUniqueId();
        String username = event.getName();

        ResolvedPermissions resolved;
        try {
            resolved = plugin.client().player(uuid, username);
        } catch (Exception e) {
            plugin.getLogger()
                    .warning("Warden: could not resolve " + username
                            + ", confining to spawn with no managed groups ("
                            + Errors.describe(e) + ")");
            state.markUnlinked(uuid);
            try {
                plugin.applier().applyPlayer(uuid, List.of(), plugin.managedGroupPrefix());
            } catch (Exception applyFailure) {
                plugin.getLogger()
                        .severe("Warden: failed to strip permissions for " + username + ": "
                                + Errors.describe(applyFailure));
            }
            return;
        }

        try {
            plugin.applier()
                    .applyPlayer(uuid, resolved.luckpermsGroups(), plugin.managedGroupPrefix());
        } catch (Exception e) {
            plugin.getLogger()
                    .severe("Warden: failed to apply permissions for " + username + ": "
                            + Errors.describe(e));
        }

        if (resolved.linked()) {
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
        } else if (!event.getPlayer().hasPermission(Permissions.PLAY)) {
            event.getPlayer().sendMessage(
                    Component.text(plugin.config().noAccessMessage(), NamedTextColor.RED));
        }
    }

    /**
     * Statistics are gathered here, while the player object is still live
     * and on the main thread. Anything later would be reading a stale or
     * absent player.
     */
    @EventHandler
    public void onQuit(PlayerQuitEvent event) {
        plugin.statsReporter().report(event.getPlayer());
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
                                    NamedTextColor.DARK_PURPLE,
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
                            + Errors.describe(e));
        }
    }
}
