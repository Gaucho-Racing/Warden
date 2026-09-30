package com.gauchoracing.warden;

import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Per-player facts learned from Warden at login and on each sync: who is
 * unlinked, so the confinement listener can stay cheap, and linked players'
 * Sentinel first names for their display names.
 */
public final class PlayerState {

    private final Set<UUID> unlinked = ConcurrentHashMap.newKeySet();
    private final Map<UUID, String> firstNames = new ConcurrentHashMap<>();

    public void markUnlinked(UUID uuid) {
        unlinked.add(uuid);
        firstNames.remove(uuid);
    }

    public void markLinked(UUID uuid, String firstName) {
        unlinked.remove(uuid);
        if (firstName == null || firstName.isBlank()) {
            firstNames.remove(uuid);
        } else {
            firstNames.put(uuid, firstName);
        }
    }

    public boolean isUnlinked(UUID uuid) {
        return unlinked.contains(uuid);
    }

    /** Null when unlinked, or when Sentinel had no name to give. */
    public String firstName(UUID uuid) {
        return firstNames.get(uuid);
    }

    public void forget(UUID uuid) {
        unlinked.remove(uuid);
        firstNames.remove(uuid);
    }
}
