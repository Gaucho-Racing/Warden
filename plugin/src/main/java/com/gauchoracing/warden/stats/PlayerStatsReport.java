package com.gauchoracing.warden.stats;

import java.util.List;

/**
 * One player's statistics, already normalised into the units Warden stores.
 *
 * <p>Conversion happens here rather than server-side because the units are
 * a Bukkit quirk, not a Warden concept: PLAY_ONE_MINUTE is ticks despite
 * its name (1200 per minute), and every distance statistic is centimetres.
 */
public record PlayerStatsReport(
        String username,
        long playtimeMinutes,
        long deaths,
        long mobKills,
        long playerKills,
        long blocksMined,
        long itemsCrafted,
        long distanceMeters,
        long damageDealt,
        long damageTaken,
        long jumps,
        long timesSlept,
        long villagerTrades,
        long raidWins,
        long sessions,
        List<Counted> topBlocks,
        List<Counted> topMobs,
        List<Counted> topCrafted) {

    public record Counted(String key, long count) {}
}
