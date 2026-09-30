package com.gauchoracing.warden;

import com.gauchoracing.warden.api.WardenClient;
import com.gauchoracing.warden.listener.ConfinementListener;
import com.gauchoracing.warden.listener.LoginListener;
import com.gauchoracing.warden.permissions.LuckPermsApplier;
import com.gauchoracing.warden.stats.StatsReporter;
import com.gauchoracing.warden.task.SyncTask;
import java.time.Duration;
import java.util.Objects;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.TimeUnit;
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
     * Owned rather than borrowed from Paper's async scheduler so shutdown can
     * wait for in-flight requests. Paper cancels a plugin's pending tasks on
     * disable but not running ones, and those then die loading classes from a
     * jar that has already been closed.
     */
    private ExecutorService io;

    /**
     * Mirrors service.ManagedGroupPrefix. Seeded with the compiled-in value
     * and replaced by whatever the service reports on each sync, so the two
     * cannot drift into Warden orphaning every group it manages.
     */
    private volatile String managedGroupPrefix = "warden-";

    @Override
    public void onEnable() {
        io = Executors.newThreadPerTaskExecutor(Thread.ofVirtual().name("warden-io-", 0).factory());
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
                scheduled -> runAsync(syncTask),
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

    @Override
    public void onDisable() {
        if (io == null) {
            return;
        }
        // Players are kicked only after plugins are disabled, so they are
        // still online here and this is the last chance to report them.
        if (statsReporter != null) {
            statsReporter.reportOnline();
        }
        io.shutdown();
        Duration grace = config.timeout().plusSeconds(2);
        try {
            if (!io.awaitTermination(grace.toMillis(), TimeUnit.MILLISECONDS)) {
                getLogger().warning("Warden: requests still running after " + grace.toSeconds()
                        + "s, abandoning them");
                io.shutdownNow();
            }
        } catch (InterruptedException e) {
            io.shutdownNow();
            Thread.currentThread().interrupt();
        }
        if (client != null) {
            client.close();
        }
    }

    public void runAsync(Runnable runnable) {
        try {
            io.execute(runnable);
        } catch (RejectedExecutionException e) {
            getLogger().fine("Warden: shutting down, dropped a request");
        }
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
