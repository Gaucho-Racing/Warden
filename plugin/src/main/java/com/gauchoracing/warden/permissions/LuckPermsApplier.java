package com.gauchoracing.warden.permissions;

import com.gauchoracing.warden.api.ManagedGroup;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;
import java.util.logging.Logger;
import net.luckperms.api.LuckPerms;
import net.luckperms.api.model.group.Group;
import net.luckperms.api.model.user.User;
import net.luckperms.api.node.NodeType;
import net.luckperms.api.node.types.InheritanceNode;
import net.luckperms.api.node.types.PermissionNode;
import net.luckperms.api.node.types.WeightNode;

/**
 * Writes Warden's desired state into LuckPerms.
 *
 * <p>Two rules govern everything here:
 *
 * <ol>
 *   <li><b>Only {@code warden-} groups are touched.</b> Memberships of any
 *       other group are never added or removed, so permissions an admin
 *       grants by hand survive every reconcile.
 *   <li><b>A {@code warden-} group is owned outright.</b> Its nodes and
 *       weight are replaced from the binding, so hand-editing one will be
 *       reverted on the next sync. Hand-granted permissions belong on a
 *       group Warden does not manage.
 * </ol>
 *
 * <p>Every method blocks on LuckPerms futures and must run off the main
 * thread.
 */
public final class LuckPermsApplier {

    private final LuckPerms luckPerms;
    private final Logger log;

    public LuckPermsApplier(LuckPerms luckPerms, Logger log) {
        this.luckPerms = luckPerms;
        this.log = log;
    }

    /**
     * Brings the managed groups themselves into line with the bindings, and
     * deletes any that no longer have one.
     *
     * <p>This must run before {@link #applyPlayer}: LuckPerms will not create
     * a group on demand, and assigning a player to one that does not exist
     * stores an inheritance node that resolves to nothing without raising an
     * error anywhere.
     */
    public void applyGroups(List<ManagedGroup> desired, String prefix) {
        Set<String> keep = new HashSet<>();

        for (ManagedGroup definition : desired) {
            keep.add(definition.name());
            Group group = luckPerms.getGroupManager()
                    .loadGroup(definition.name())
                    .join()
                    .orElseGet(() -> luckPerms.getGroupManager()
                            .createAndLoadGroup(definition.name())
                            .join());

            // Replace rather than merge — the binding is the whole truth for
            // a group Warden owns.
            group.data().clear(NodeType.PERMISSION::matches);
            group.data().clear(NodeType.WEIGHT::matches);
            for (String permission : definition.permissions()) {
                group.data().add(PermissionNode.builder(permission).build());
            }
            if (definition.weight() != 0) {
                group.data().add(WeightNode.builder(definition.weight()).build());
            }
            luckPerms.getGroupManager().saveGroup(group).join();
        }

        // Anything still carrying the prefix but absent from the bindings
        // belongs to a binding that was deleted.
        for (Group group : luckPerms.getGroupManager().getLoadedGroups()) {
            String name = group.getName();
            if (name.startsWith(prefix) && !keep.contains(name)) {
                log.info("Warden: removing orphaned group " + name);
                luckPerms.getGroupManager().deleteGroup(group).join();
            }
        }
    }

    /**
     * Sets a player's {@code warden-} memberships to exactly {@code desired}.
     *
     * <p>An empty set is meaningful — it is how a revocation, or an unlinked
     * player, is applied.
     */
    public void applyPlayer(UUID uuid, List<String> desired, String prefix) {
        User user = luckPerms.getUserManager().loadUser(uuid).join();
        if (user == null) {
            log.warning("Warden: no LuckPerms user for " + uuid);
            return;
        }

        user.data()
                .clear(NodeType.INHERITANCE.predicate(
                        node -> node.getGroupName().startsWith(prefix)));
        for (String group : desired) {
            user.data().add(InheritanceNode.builder(group).build());
        }
        luckPerms.getUserManager().saveUser(user).join();
    }
}
