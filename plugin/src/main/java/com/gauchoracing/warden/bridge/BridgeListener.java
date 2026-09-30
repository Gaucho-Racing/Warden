package com.gauchoracing.warden.bridge;

import com.gauchoracing.warden.Permissions;
import com.gauchoracing.warden.staff.VanishManager;
import com.google.gson.JsonObject;
import io.papermc.paper.advancement.AdvancementDisplay;
import io.papermc.paper.event.player.AsyncChatEvent;
import java.util.Map;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import net.kyori.adventure.text.format.TextColor;
import net.kyori.adventure.text.serializer.plain.PlainTextComponentSerializer;
import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.bukkit.event.player.PlayerAdvancementDoneEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import org.bukkit.plugin.Plugin;

/**
 * Turns game events into bridge events, and bridge messages into chat.
 *
 * <p>Only players with {@link Permissions#PLAY} have their chat relayed, so
 * people stuck at spawn cannot spam the Discord channel. Vanished staff are
 * never announced.
 */
public final class BridgeListener implements Listener {

    /** Discord's blurple, marking a line as coming from Discord instead of a prefix. */
    private static final TextColor DISCORD_BLURPLE = TextColor.color(0x5865F2);
    private static final PlainTextComponentSerializer PLAIN = PlainTextComponentSerializer.plainText();

    private final Plugin plugin;
    private final VanishManager vanish;
    private BridgeClient client;

    public BridgeListener(Plugin plugin, VanishManager vanish) {
        this.plugin = plugin;
        this.vanish = vanish;
    }

    public void attach(BridgeClient client) {
        this.client = client;
    }

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onChat(AsyncChatEvent event) {
        Player player = event.getPlayer();
        if (player.hasPermission(Permissions.PLAY) && !vanish.isVanished(player.getUniqueId())) {
            client.send(Map.of("type", "chat", "uuid", player.getUniqueId().toString(),
                    "text", PLAIN.serialize(event.message())));
        }
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onJoin(PlayerJoinEvent event) {
        announce(event.getPlayer(), "join");
    }

    /** LOW so vanish state is still set; VanishManager reveals the player at NORMAL. */
    @EventHandler(priority = EventPriority.LOW)
    public void onQuit(PlayerQuitEvent event) {
        announce(event.getPlayer(), "quit");
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onDeath(PlayerDeathEvent event) {
        Player player = event.getPlayer();
        Component message = event.deathMessage();
        if (message != null && !vanish.isVanished(player.getUniqueId())) {
            client.send(Map.of("type", "death", "uuid", player.getUniqueId().toString(),
                    "text", PLAIN.serialize(message)));
        }
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onAdvancement(PlayerAdvancementDoneEvent event) {
        AdvancementDisplay display = event.getAdvancement().getDisplay();
        Player player = event.getPlayer();
        if (display != null && display.doesAnnounceToChat() && !vanish.isVanished(player.getUniqueId())) {
            client.send(Map.of("type", "advancement", "uuid", player.getUniqueId().toString(),
                    "text", PLAIN.serialize(display.title())));
        }
    }

    private void announce(Player player, String type) {
        if (!vanish.isVanished(player.getUniqueId())) {
            client.send(Map.of("type", type, "uuid", player.getUniqueId().toString()));
        }
    }

    /**
     * Called on the bridge thread. Text arrives as plain strings and is only
     * ever placed in text components, so a Discord message cannot smuggle in
     * formatting, click actions or commands.
     */
    public void onBridgeMessage(JsonObject message) {
        if (!message.has("type") || !"discord_message".equals(message.get("type").getAsString())) {
            return;
        }
        // Same "Name » text" shape as game chat (see the warden datapack), with
        // the name in blurple instead of a [Discord] prefix.
        Component line = Component.text()
                .append(Component.text(message.get("name").getAsString(), DISCORD_BLURPLE))
                .append(Component.text(" » ", NamedTextColor.WHITE))
                .append(Component.text(message.get("text").getAsString()))
                .build();
        Server server = plugin.getServer();
        server.getGlobalRegionScheduler().run(plugin, task -> server.sendMessage(line));
    }
}
