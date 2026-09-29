package com.gauchoracing.warden;

import com.gauchoracing.warden.listener.LoginListener;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.entity.Player;
import org.jetbrains.annotations.NotNull;

/** {@code /warden sync|status} and {@code /link}. */
public final class WardenCommand implements CommandExecutor {

    private final WardenPlugin plugin;
    private final LoginListener loginListener;

    public WardenCommand(WardenPlugin plugin, LoginListener loginListener) {
        this.plugin = plugin;
        this.loginListener = loginListener;
    }

    @Override
    public boolean onCommand(
            @NotNull CommandSender sender,
            @NotNull Command command,
            @NotNull String label,
            @NotNull String[] args) {

        if (command.getName().equalsIgnoreCase("link")) {
            if (!(sender instanceof Player player)) {
                sender.sendMessage(Component.text("Only a player can link an account."));
                return true;
            }
            sender.sendMessage(Component.text("Fetching your link...", NamedTextColor.GRAY));
            plugin.runAsync(() -> loginListener.sendLinkPrompt(player.getUniqueId(), player.getName()));
            return true;
        }

        if (args.length == 0) {
            return false;
        }
        switch (args[0].toLowerCase()) {
            case "sync" -> {
                sender.sendMessage(Component.text("Warden: sync started", NamedTextColor.GRAY));
                plugin.runAsync(plugin.syncTask());
                return true;
            }
            case "status" -> {
                sender.sendMessage(Component.text(
                        "Warden: " + plugin.config().baseUrl()
                                + " · prefix " + plugin.managedGroupPrefix(),
                        NamedTextColor.GRAY));
                return true;
            }
            default -> {
                return false;
            }
        }
    }
}
