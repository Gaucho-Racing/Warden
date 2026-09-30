package com.gauchoracing.warden.staff;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.GameMode;
import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.PlayerJoinEvent;

/** {@code /fly}: survival flight for staff, so they never need creative. */
public final class FlyCommand implements CommandExecutor, Listener {

    @Override
    public boolean onCommand(CommandSender sender, Command command, String label, String[] args) {
        if (!(sender instanceof Player player)) {
            sender.sendMessage(Component.text("Only players can fly.", NamedTextColor.RED));
            return true;
        }
        boolean enable = !player.getAllowFlight();
        player.setAllowFlight(enable);
        if (!enable) {
            player.setFlying(false);
        }
        player.sendMessage(Component.text(enable ? "Flight enabled" : "Flight disabled", NamedTextColor.GRAY));
        return true;
    }

    /** Flight is saved with the player, so clear it rather than carry it into a new session. */
    @EventHandler(priority = EventPriority.MONITOR)
    public void onJoin(PlayerJoinEvent event) {
        Player player = event.getPlayer();
        GameMode mode = player.getGameMode();
        if (player.getAllowFlight() && (mode == GameMode.SURVIVAL || mode == GameMode.ADVENTURE)) {
            player.setFlying(false);
            player.setAllowFlight(false);
        }
    }
}
