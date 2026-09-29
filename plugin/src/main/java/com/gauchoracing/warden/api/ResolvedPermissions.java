package com.gauchoracing.warden.api;

import java.util.List;
import java.util.UUID;

/**
 * A player's complete desired permission state — not a delta.
 *
 * <p>Complete state is deliberate: a dropped delta could leave someone
 * holding a revoked permission indefinitely, whereas a dropped full-state
 * update is corrected by the next one.
 *
 * <p>An unlinked player is a normal 200 with {@code linked=false} and empty
 * lists. "This player gets nothing" is a valid answer to apply, not an error.
 */
public record ResolvedPermissions(
        UUID uuid,
        String username,
        boolean linked,
        String entityId,
        List<String> sentinelGroups,
        List<String> luckpermsGroups,
        List<String> permissions) {

    public List<String> luckpermsGroups() {
        return luckpermsGroups == null ? List.of() : luckpermsGroups;
    }

    public static ResolvedPermissions unlinked(UUID uuid, String username) {
        return new ResolvedPermissions(uuid, username, false, null, List.of(), List.of(), List.of());
    }
}
