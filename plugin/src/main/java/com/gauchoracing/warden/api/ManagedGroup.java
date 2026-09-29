package com.gauchoracing.warden.api;

import java.util.List;

/**
 * A LuckPerms group Warden owns, as described by the service.
 *
 * <p>LuckPerms will not create a group on demand — {@code getGroup} returns
 * null and {@code loadGroup} an empty Optional — and adding a player to a
 * parent group that does not exist stores an inheritance node resolving to
 * nothing, with no error raised anywhere. So these must be provisioned
 * before any membership is assigned.
 */
public record ManagedGroup(String name, String sourceGroup, List<String> permissions, int weight) {

    public List<String> permissions() {
        return permissions == null ? List.of() : permissions;
    }
}
