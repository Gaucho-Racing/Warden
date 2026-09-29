package service

import (
	"hash/fnv"
	"math/rand"
	"sort"
	"time"

	"github.com/gaucho-racing/warden/warden/model"
)

// minedBlocks is the pool the mock draws from — the blocks a survival player
// actually racks up counts on, so the chart looks plausible.
var minedBlocks = []string{
	"minecraft:stone", "minecraft:deepslate", "minecraft:dirt", "minecraft:cobblestone",
	"minecraft:oak_log", "minecraft:iron_ore", "minecraft:coal_ore", "minecraft:sand",
	"minecraft:gravel", "minecraft:diamond_ore", "minecraft:andesite", "minecraft:granite",
}

// MockPlayerStats fabricates a plausible stat line for a UUID.
//
// PLACEHOLDER. Seeded from the UUID so a given player's numbers are stable
// across reloads — random-per-request values make the UI impossible to judge
// and look broken. Replace wholesale once the plugin reports real counters;
// the return shape is the contract and should not need to change.
func MockPlayerStats(uuid string, username string) model.PlayerStats {
	h := fnv.New64a()
	_, _ = h.Write([]byte(uuid))
	rnd := rand.New(rand.NewSource(int64(h.Sum64())))

	playtime := 240 + rnd.Intn(14_000)
	mined := 1_200 + rnd.Intn(48_000)
	now := time.Now()

	blocks := append([]string(nil), minedBlocks...)
	rnd.Shuffle(len(blocks), func(i, j int) { blocks[i], blocks[j] = blocks[j], blocks[i] })
	top := make([]model.BlockCount, 0, 6)
	remaining := mined
	for i := 0; i < 6; i++ {
		// Each entry takes a shrinking slice of what's left, which produces
		// the long-tailed shape a real mined tally has.
		count := remaining / (3 + i)
		if count < 1 {
			count = 1
		}
		top = append(top, model.BlockCount{Block: blocks[i], Count: count})
		remaining -= count
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Count > top[j].Count })

	return model.PlayerStats{
		UUID:            uuid,
		Username:        username,
		Source:          model.StatSourceMock,
		PlaytimeMinutes: playtime,
		Deaths:          rnd.Intn(80),
		MobKills:        60 + rnd.Intn(3_000),
		BlocksMined:     mined,
		DistanceMeters:  5_000 + rnd.Intn(900_000),
		JoinCount:       5 + rnd.Intn(300),
		FirstSeen:       now.AddDate(0, 0, -(30 + rnd.Intn(300))),
		LastSeen:        now.Add(-time.Duration(rnd.Intn(72)) * time.Hour),
		TopBlocks:       top,
		Last7Days: &model.StatWindow{
			PlaytimeMinutes: rnd.Intn(600),
			Deaths:          rnd.Intn(6),
			MobKills:        rnd.Intn(180),
			BlocksMined:     rnd.Intn(4_000),
		},
	}
}

// PlayerStatsForUUID returns the stat line for a linked account.
func PlayerStatsForUUID(uuid string) (model.PlayerStats, error) {
	account, err := GetAccountByUUID(uuid)
	if err != nil {
		return model.PlayerStats{}, err
	}
	stats := MockPlayerStats(account.UUID, account.Username)
	if !account.LastSeenAt.IsZero() {
		stats.LastSeen = account.LastSeenAt
	}
	return stats, nil
}
