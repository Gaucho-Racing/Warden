package com.gauchoracing.warden;

import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/** Who is currently unlinked, so the confinement listener can stay cheap. */
public final class PlayerState {

    private final Set<UUID> unlinked = ConcurrentHashMap.newKeySet();

    public void markUnlinked(UUID uuid) {
        unlinked.add(uuid);
    }

    public void markLinked(UUID uuid) {
        unlinked.remove(uuid);
    }

    public boolean isUnlinked(UUID uuid) {
        return unlinked.contains(uuid);
    }

    public void forget(UUID uuid) {
        unlinked.remove(uuid);
    }
}
