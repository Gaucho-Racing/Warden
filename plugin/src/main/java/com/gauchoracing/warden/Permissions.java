package com.gauchoracing.warden;

/** Permission nodes Warden checks. Declared in plugin.yml. */
public final class Permissions {

    /**
     * Allowed to play: survival mode and free of spawn confinement. Granted
     * only through a binding, never by op, so it follows Sentinel group
     * membership. Unlinked players never hold it.
     */
    public static final String PLAY = "warden.play";

    public static final String FLY = "warden.fly";
    public static final String VANISH = "warden.vanish";

    /** Sees vanished players as normal, so staff do not lose track of each other. */
    public static final String VANISH_SEE = "warden.vanish.see";

    private Permissions() {}
}
