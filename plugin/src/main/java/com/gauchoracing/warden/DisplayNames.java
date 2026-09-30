package com.gauchoracing.warden;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.entity.Player;

/**
 * Shows linked players as "First (username)". The Minecraft name stays in
 * view because commands, nametags and bans all still key off it.
 */
public final class DisplayNames {

    private DisplayNames() {}

    /** Main thread or the player's own scheduler only. Null resets to the Minecraft name. */
    public static void apply(Player player, String firstName) {
        if (firstName == null || firstName.isBlank()) {
            player.displayName(null);
            player.playerListName(null);
            return;
        }
        Component name = Component.text(firstName)
                .append(Component.text(" (" + player.getName() + ")", NamedTextColor.GRAY));
        player.displayName(name);
        player.playerListName(name);
    }
}
