package com.gauchoracing.warden;

import com.gauchoracing.warden.api.WardenClient;
import com.gauchoracing.warden.listener.ConfinementListener;
import com.gauchoracing.warden.listener.LoginListener;
import com.gauchoracing.warden.permissions.LuckPermsApplier;
import com.gauchoracing.warden.stats.StatsReporter;
import com.gauchoracing.warden.task.SyncTask;
import java.util.Objects;
import net.luckperms.api.LuckPerms;
import net.luckperms.api.LuckPermsProvider;
import org.bukkit.plugin.java.JavaPlugin;

public final class WardenPlugin extends JavaPlugin {

    private WardenConfig config;
    private WardenClient client;
    private LuckPermsApplier applier;
    private SyncTask syncTask;
    private StatsReporter statsReporter;

    /**
     * Mirrors service.ManagedGroupPrefix. Seeded with the compiled-in value
     * and replaced by whatever the service reports on each sync, so the two
     * cannot drift into Warden orphaning every group it manages.
     */
    private volatile String managedGroupPrefix = "warden-";

    @Override
    public void onEnable() {
        saveDefaultConfig();
        config = WardenConfig.from(getConfig());

        if (config.token().isBlank()) {
            getLogger().severe("warden.token is not set — every /plugin request would be "
                    + "rejected. Disabling.");
            getServer().getPluginManager().disablePlugin(this);
            return;
        }

        LuckPerms luckPerms;
        try {
            luckPerms = LuckPermsProvider.get();
        } catch (IllegalStateException e) {
            getLogger().severe("LuckPerms API unavailable. Disabling.");
            getServer().getPluginManager().disablePlugin(this);
            return;
        }

        client = new WardenClient(config.baseUrl(), config.token(), config.timeout());
        applier = new LuckPermsApplier(luckPerms, getLogger());
        PlayerState state = new PlayerState();
        LoginListener loginListener = new LoginListener(this, state);
        getServer().getPluginManager().registerEvents(loginListener, this);
        getServer()
                .getPluginManager()
                .registerEvents(
                        new ConfinementListener(
                                getServer(),
                                state,
                                config.confinementRadius(),
                                config.confinementEnabled(),
                                config.confinementReminder()),
                        this);

        WardenCommand command = new WardenCommand(this, loginListener);
        Objects.requireNonNull(getCommand("warden")).setExecutor(command);
        Objects.requireNonNull(getCommand("link")).setExecutor(command);

        statsReporter = new StatsReporter(this);
        syncTask = new SyncTask(this, state);
        long ticks = config.syncInterval().toSeconds() * 20L;
        // Run once shortly after boot so the managed groups exist before the
        // first player arrives, then on the configured interval.
        getServer().getAsyncScheduler().runAtFixedRate(
                this,
                scheduled -> syncTask.run(),
                5,
                config.syncInterval().toSeconds(),
                java.util.concurrent.TimeUnit.SECONDS);

        if (config.statsEnabled()) {
            // Gathering touches player objects, so it runs on the main
            // thread; StatsReporter hops async for the HTTP itself.
            getServer()
                    .getGlobalRegionScheduler()
                    .runAtFixedRate(
                            this,
                            scheduled -> statsReporter.reportOnline(),
                            config.statsInterval().toSeconds() * 20L,
                            config.statsInterval().toSeconds() * 20L);
        }

        getLogger().info("Warden enabled against " + config.baseUrl()
                + " (sync every " + ticks / 20 + "s)");
    }

    public void runAsync(Runnable runnable) {
        getServer().getAsyncScheduler().runNow(this, scheduled -> runnable.run());
    }

    public WardenConfig config() {
        return config;
    }

    public WardenClient client() {
        return client;
    }

    public LuckPermsApplier applier() {
        return applier;
    }

    public StatsReporter statsReporter() {
        return statsReporter;
    }

    public SyncTask syncTask() {
        return syncTask;
    }

    public String managedGroupPrefix() {
        return managedGroupPrefix;
    }

    public void setManagedGroupPrefix(String prefix) {
        if (prefix != null && !prefix.isBlank()) {
            this.managedGroupPrefix = prefix;
        }
    }
}
