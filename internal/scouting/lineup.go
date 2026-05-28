package scouting

import (
	"fmt"
	"strings"

	"github.com/pdavlin/go-playball/internal/api"
)

// topBatters is the number of lineup slots per side whose season hitting
// stats we fetch and inject into the prompt.
const topBatters = 5

// LineupCtx is one side's batting order plus per-batter context. An empty
// Batters slice signals "lineup not posted" and triggers the team-line
// fallback in RenderPrompt.
type LineupCtx struct {
	Batters []BatterCtx
}

// BatterCtx is one batter in the lineup. SeasonLine is nil when the MLB
// stats fetch failed or the batter has no rostered splits yet.
type BatterCtx struct {
	PlayerID     int
	Name         string
	Position     string
	BattingOrder int
	SeasonLine   *api.HittingLine
}

// gatherLineups extracts the batting orders from the live feed. Returns
// zero-value LineupCtx structs when either side's lineup is missing so the
// caller falls back to the team-line prompt.
func gatherLineups(g *api.Game) (away, home LineupCtx) {
	if g == nil || g.LiveData == nil {
		return LineupCtx{}, LineupCtx{}
	}
	bs := g.LiveData.Boxscore
	awayOrder := bs.Teams.Away.BattingOrder
	homeOrder := bs.Teams.Home.BattingOrder
	if len(awayOrder) == 0 || len(homeOrder) == 0 {
		return LineupCtx{}, LineupCtx{}
	}

	away.Batters = buildBatters(awayOrder, bs.Teams.Away.Players, g.GameData)
	home.Batters = buildBatters(homeOrder, bs.Teams.Home.Players, g.GameData)
	return away, home
}

func buildBatters(order []int, boxPlayers map[string]api.BoxscorePlayer, gameData *api.GameData) []BatterCtx {
	out := make([]BatterCtx, 0, len(order))
	for i, id := range order {
		b := BatterCtx{PlayerID: id, BattingOrder: i + 1}
		key := fmt.Sprintf("ID%d", id)
		if gameData != nil {
			if p, ok := gameData.Players[key]; ok {
				b.Name = p.FullName
			}
		}
		if bp, ok := boxPlayers[key]; ok {
			if b.Name == "" {
				b.Name = bp.Person.FullName
			}
			b.Position = bp.Position.Abbreviation
		}
		out = append(out, b)
	}
	return out
}

// lineupFingerprint is the deterministic cache-key contribution. It covers
// the full batting order on both sides so any swap busts the cache. The
// empty-lineup fingerprint ("away:|home:") is a stable non-zero string so
// Phase 1 caches (which had no fingerprint input at all) are also treated
// as misses.
func lineupFingerprint(away, home LineupCtx) string {
	var b strings.Builder
	b.WriteString("away:")
	writeIDs(&b, away.Batters)
	b.WriteString("|home:")
	writeIDs(&b, home.Batters)
	return b.String()
}

func writeIDs(b *strings.Builder, batters []BatterCtx) {
	for i, x := range batters {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(b, "%d", x.PlayerID)
	}
}
