package com.gauchoracing.warden.listener;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.TextReplacementConfig;
import net.kyori.adventure.text.serializer.plain.PlainTextComponentSerializer;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.bukkit.event.player.PlayerAdvancementDoneEvent;
import org.bukkit.event.player.PlayerQuitEvent;

/**
 * Vanilla builds death and advancement messages from the scoreboard name, so
 * they would still say "BK1031" after the display name is set. This swaps in
 * the display name before anything else, including the Discord bridge at
 * MONITOR, reads the message.
 */
public final class DisplayNameListener implements Listener {

    private static final PlainTextComponentSerializer PLAIN = PlainTextComponentSerializer.plainText();

    @EventHandler(priority = EventPriority.LOW)
    public void onDeath(PlayerDeathEvent event) {
        Component message = withDisplayName(event.deathMessage(), event.getPlayer());
        Player killer = event.getPlayer().getKiller();
        if (killer != null && !killer.equals(event.getPlayer())) {
            message = withDisplayName(message, killer);
        }
        event.deathMessage(message);
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onAdvancement(PlayerAdvancementDoneEvent event) {
        event.message(withDisplayName(event.message(), event.getPlayer()));
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onQuit(PlayerQuitEvent event) {
        event.quitMessage(withDisplayName(event.quitMessage(), event.getPlayer()));
    }

    /**
     * Skips messages that already use the display name, which also stops the
     * name from being substituted inside itself ("Bharat (Bharat (BK1031))").
     */
    private static Component withDisplayName(Component message, Player player) {
        if (message == null) {
            return null;
        }
        String displayName = PLAIN.serialize(player.displayName());
        if (displayName.equals(player.getName()) || PLAIN.serialize(message).contains(displayName)) {
            return message;
        }
        return message.replaceText(TextReplacementConfig.builder()
                .matchLiteral(player.getName())
                .replacement(player.displayName())
                .build());
    }
}
