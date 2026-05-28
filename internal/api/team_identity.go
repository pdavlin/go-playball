package api

// TeamIdentity captures the canonical team metadata derivable from a
// fetched Game. Live-feed payloads populate gameData.teams.*; schedule
// payloads populate the top-level teams.{away,home}.team.*. Callers
// that hydrate a schedule-view Game via FetchGame lose the top-level
// fields, so they must resolve identity from both sources.
type TeamIdentity struct {
	ID           int
	Name         string // e.g. "Cincinnati Reds"
	TeamName     string // e.g. "Reds" — MLB API teamName field; may be ""
	Abbreviation string // e.g. "CIN"
	LeagueRecord LeagueRecord
}

// ResolveTeam merges gameData.teams.{side} with a caller-supplied
// fallback (typically the schedule-side snapshot captured before
// hydration replaced g). The live feed wins for Name, TeamName,
// LeagueRecord, ID. Abbreviation comes from the fallback because the
// live feed's FullTeamInfo doesn't carry it.
//
// side must be "away" or "home".
func ResolveTeam(g *Game, side string, fallback TeamIdentity) TeamIdentity {
	out := fallback
	if g == nil || g.GameData == nil {
		return out
	}
	var ft FullTeamInfo
	switch side {
	case "away":
		ft = g.GameData.Teams.Away
	case "home":
		ft = g.GameData.Teams.Home
	default:
		return out
	}
	if ft.Name != "" {
		out.Name = ft.Name
	}
	if ft.TeamName != "" {
		out.TeamName = ft.TeamName
	}
	if ft.ID != 0 {
		out.ID = ft.ID
	}
	if ft.LeagueRecord.Wins != 0 || ft.LeagueRecord.Losses != 0 {
		out.LeagueRecord = ft.LeagueRecord
	}
	return out
}

// SnapshotAwayIdentity captures the schedule-side identity for the away
// team before any hydration. Use this to feed the fallback to
// ResolveTeam after FetchGame has run.
func SnapshotAwayIdentity(g *Game) TeamIdentity {
	if g == nil {
		return TeamIdentity{}
	}
	return TeamIdentity{
		ID:           g.Teams.Away.Team.ID,
		Name:         g.Teams.Away.Team.Name,
		Abbreviation: g.Teams.Away.Team.Abbreviation,
		LeagueRecord: g.Teams.Away.LeagueRecord,
	}
}

// SnapshotHomeIdentity is the home-team counterpart of SnapshotAwayIdentity.
func SnapshotHomeIdentity(g *Game) TeamIdentity {
	if g == nil {
		return TeamIdentity{}
	}
	return TeamIdentity{
		ID:           g.Teams.Home.Team.ID,
		Name:         g.Teams.Home.Team.Name,
		Abbreviation: g.Teams.Home.Team.Abbreviation,
		LeagueRecord: g.Teams.Home.LeagueRecord,
	}
}
