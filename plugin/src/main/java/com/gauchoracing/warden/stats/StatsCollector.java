package com.gauchoracing.warden.stats;

import com.gauchoracing.warden.stats.PlayerStatsReport.Counted;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import org.bukkit.Material;
import org.bukkit.Statistic;
import org.bukkit.entity.EntityType;
import org.bukkit.entity.Player;

/**
 * Reads a player's statistics out of the server.
 *
 * <p>Must run on the main thread — Bukkit entity access is not thread safe.
 * The work is only map lookups, so it is microseconds even though the typed
 * scans make a few thousand of them; it is the HTTP report that has to be
 * asynchronous, not the gathering.
 */
public final class StatsCollector {

    private static final int TOP_N = 6;

    /** Every movement statistic, so "distance travelled" means all of it. */
    private static final Statistic[] DISTANCE = {
        Statistic.WALK_ONE_CM,
        Statistic.SPRINT_ONE_CM,
        Statistic.CROUCH_ONE_CM,
        Statistic.SWIM_ONE_CM,
        Statistic.WALK_ON_WATER_ONE_CM,
        Statistic.WALK_UNDER_WATER_ONE_CM,
        Statistic.FLY_ONE_CM,
        Statistic.AVIATE_ONE_CM,
        Statistic.BOAT_ONE_CM,
        Statistic.MINECART_ONE_CM,
        Statistic.HORSE_ONE_CM,
        Statistic.PIG_ONE_CM,
        Statistic.STRIDER_ONE_CM,
        Statistic.CLIMB_ONE_CM,
    };

    private StatsCollector() {}

    /** A full tally plus its top slice, so the total is not the top-N sum. */
    private record Tally(long total, List<Counted> top) {}

    public static PlayerStatsReport collect(Player player) {
        Tally blocks = tally(player, Statistic.MINE_BLOCK, Material::isBlock);
        Tally crafted = tally(player, Statistic.CRAFT_ITEM, Material::isItem);
        Tally mobs = mobTally(player);

        return new PlayerStatsReport(
                player.getName(),
                // PLAY_ONE_MINUTE is ticks, not minutes. 20/sec, 1200/min.
                stat(player, Statistic.PLAY_ONE_MINUTE) / 1200,
                stat(player, Statistic.DEATHS),
                stat(player, Statistic.MOB_KILLS),
                stat(player, Statistic.PLAYER_KILLS),
                blocks.total(),
                crafted.total(),
                distanceMeters(player),
                // Damage statistics are in tenths of a heart.
                stat(player, Statistic.DAMAGE_DEALT),
                stat(player, Statistic.DAMAGE_TAKEN),
                stat(player, Statistic.JUMP),
                stat(player, Statistic.SLEEP_IN_BED),
                stat(player, Statistic.TRADED_WITH_VILLAGER),
                stat(player, Statistic.RAID_WIN),
                // There is no join counter; LEAVE_GAME is the closest thing
                // and reads one low while the player is still online.
                stat(player, Statistic.LEAVE_GAME),
                blocks.top(),
                mobs.top(),
                crafted.top());
    }

    private static long stat(Player player, Statistic statistic) {
        try {
            return player.getStatistic(statistic);
        } catch (IllegalArgumentException e) {
            // A statistic the running server does not track. Not worth
            // failing a whole report over.
            return 0;
        }
    }

    private static long distanceMeters(Player player) {
        long centimetres = 0;
        for (Statistic statistic : DISTANCE) {
            centimetres += stat(player, statistic);
        }
        return centimetres / 100;
    }

    /**
     * Walks every material for a typed statistic.
     *
     * <p>The total is summed over the whole tally, not the returned slice —
     * "blocks mined" means all of them, while the chart only shows the top
     * few.
     */
    private static Tally tally(
            Player player, Statistic statistic, java.util.function.Predicate<Material> include) {
        List<Counted> counts = new ArrayList<>();
        long total = 0;
        for (Material material : Material.values()) {
            if (material.isLegacy() || !include.test(material)) {
                continue;
            }
            long count = typed(player, statistic, material);
            if (count > 0) {
                counts.add(new Counted(material.getKey().toString(), count));
                total += count;
            }
        }
        return new Tally(total, top(counts));
    }

    private static Tally mobTally(Player player) {
        List<Counted> counts = new ArrayList<>();
        long total = 0;
        for (EntityType type : EntityType.values()) {
            if (!type.isAlive()) {
                continue;
            }
            long killed;
            try {
                killed = player.getStatistic(Statistic.KILL_ENTITY, type);
            } catch (IllegalArgumentException e) {
                continue;
            }
            if (killed > 0) {
                counts.add(new Counted(type.getKey().toString(), killed));
                total += killed;
            }
        }
        return new Tally(total, top(counts));
    }

    private static long typed(Player player, Statistic statistic, Material material) {
        try {
            return player.getStatistic(statistic, material);
        } catch (IllegalArgumentException e) {
            // Materials and statistics do not line up perfectly; a block
            // that is not minable throws rather than returning zero.
            return 0;
        }
    }

    private static List<Counted> top(List<Counted> counts) {
        counts.sort(Comparator.comparingLong(Counted::count).reversed());
        return counts.size() <= TOP_N ? counts : new ArrayList<>(counts.subList(0, TOP_N));
    }
}
