package com.gauchoracing.warden.task;

import com.gauchoracing.warden.Errors;
import com.gauchoracing.warden.PlayerState;
import com.gauchoracing.warden.WardenPlugin;
import com.gauchoracing.warden.api.ResolvedPermissions;
import com.gauchoracing.warden.api.SyncSnapshot;

/**
 * The periodic reconcile.
 *
 * <p>This is what makes a revocation land without waiting for the player to
 * rejoin: Sentinel group changes are invisible to the game server until
 * something asks, and nothing pushes.
 *
 * <p>Groups are applied before players, always — assigning somebody to a
 * group LuckPerms has never heard of fails silently.
 */
public final class SyncTask implements Runnable {

    private final WardenPlugin plugin;
    private final PlayerState state;

    public SyncTask(WardenPlugin plugin, PlayerState state) {
        this.plugin = plugin;
        this.state = state;
    }

    @Override
    public void run() {
        SyncSnapshot snapshot;
        try {
            snapshot = plugin.client().sync();
        } catch (Exception e) {
            plugin.getLogger().warning("Warden: sync failed, keeping current state: " + Errors.describe(e));
            return;
        }

        plugin.setManagedGroupPrefix(snapshot.managedGroupPrefix());

        try {
            plugin.applier().applyGroups(snapshot.groups(), snapshot.managedGroupPrefix());
        } catch (Exception e) {
            // Without the groups, assigning memberships would silently grant
            // nothing, so stop rather than press on.
            plugin.getLogger().severe("Warden: failed to reconcile groups, skipping players: "
                    + Errors.describe(e));
            return;
        }

        int applied = 0;
        for (ResolvedPermissions player : snapshot.players()) {
            try {
                plugin.applier()
                        .applyPlayer(player.uuid(), player.luckpermsGroups(),
                                snapshot.managedGroupPrefix());
                state.markLinked(player.uuid());
                applied++;
            } catch (Exception e) {
                plugin.getLogger().warning("Warden: failed to apply " + player.username() + ": "
                        + Errors.describe(e));
            }
        }
        plugin.getLogger().info("Warden: sync applied " + snapshot.groups().size()
                + " groups and " + applied + " players");
    }
}
