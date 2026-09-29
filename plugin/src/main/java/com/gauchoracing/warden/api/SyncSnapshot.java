package com.gauchoracing.warden.api;

import java.util.List;

/**
 * The whole permission system in one response.
 *
 * <p>Both lists are exhaustive, which is what makes garbage collection
 * possible: a group carrying {@link #managedGroupPrefix()} that is absent
 * from {@link #groups()} belongs to a deleted binding, and a managed group
 * absent from a player's entry has been revoked.
 */
public record SyncSnapshot(
        String managedGroupPrefix, List<ManagedGroup> groups, List<ResolvedPermissions> players) {

    public List<ManagedGroup> groups() {
        return groups == null ? List.of() : groups;
    }

    public List<ResolvedPermissions> players() {
        return players == null ? List.of() : players;
    }
}
