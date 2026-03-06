package api

import "time"

// Game represents a baseball game
type Game struct {
	ID          int       `json:"gamePk"`
	GameDate    time.Time `json:"gameDate"`
	Status      GameStatus
	Teams       Teams         `json:"teams"`
	Linescore   *Linescore    `json:"linescore,omitempty"` // hydrated by schedule endpoint
	LiveData    *LiveData     `json:"liveData,omitempty"`
	GameData    *GameData     `json:"gameData,omitempty"`
	MetaData    *MetaData     `json:"metaData,omitempty"`
	GameType    string        `json:"gameType,omitempty"`
}

// GameStatus represents the current state of a game
type GameStatus struct {
	AbstractGameState string `json:"abstractGameState"` // Preview, Live, Final
	DetailedState     string `json:"detailedState"`
	StatusCode        string `json:"statusCode"`
	StartTimeTBD      bool   `json:"startTimeTBD"`
}

// Teams contains both home and away team info
type Teams struct {
	Away TeamInfo `json:"away"`
	Home TeamInfo `json:"home"`
}

// TeamInfo contains team details and score
type TeamInfo struct {
	Team       Team        `json:"team"`
	Score      int         `json:"score,omitempty"`
	IsWinner   bool        `json:"isWinner,omitempty"`
	LeagueRecord LeagueRecord `json:"leagueRecord"`
}

// Team represents a baseball team
type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation,omitempty"`
}

// LeagueRecord contains win-loss record
type LeagueRecord struct {
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Pct    string `json:"pct"`
}

// LiveData contains in-game data
type LiveData struct {
	Plays     Plays     `json:"plays"`
	Linescore Linescore `json:"linescore"`
	Boxscore  Boxscore  `json:"boxscore"`
	Decisions Decisions `json:"decisions,omitempty"`
}

// Plays contains play-by-play data
type Plays struct {
	CurrentPlay *Play   `json:"currentPlay,omitempty"`
	AllPlays    []Play  `json:"allPlays"`
}

// Play represents a single play in the game
type Play struct {
	Result      PlayResult `json:"result"`
	About       About      `json:"about"`
	Count       Count      `json:"count,omitempty"`
	Matchup     Matchup    `json:"matchup"`
	PlayEvents  []PlayEvent `json:"playEvents,omitempty"`
}

// PlayResult contains the outcome of a play
type PlayResult struct {
	Type        string `json:"type"`
	Event       string `json:"event"`
	EventType   string `json:"eventType"`
	Description string `json:"description"`
	RBI         int    `json:"rbi,omitempty"`
	AwayScore   int    `json:"awayScore,omitempty"`
	HomeScore   int    `json:"homeScore,omitempty"`
}

// About contains contextual info about when the play occurred
type About struct {
	AtBatIndex    int    `json:"atBatIndex"`
	HalfInning    string `json:"halfInning"` // top, bottom
	Inning        int    `json:"inning"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	IsComplete    bool   `json:"isComplete"`
	IsScoringPlay bool   `json:"isScoringPlay"`
	HasOut        bool   `json:"hasOut"`
}

// Count represents balls, strikes, outs
type Count struct {
	Balls   int `json:"balls"`
	Strikes int `json:"strikes"`
	Outs    int `json:"outs"`
}

// Matchup contains pitcher vs batter info
type Matchup struct {
	Batter  Player `json:"batter"`
	Pitcher Player `json:"pitcher"`
	Splits  Splits `json:"splits"`
}

// Player represents a player
type Player struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
}

// Position represents a player's fielding position
type Position struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Abbreviation string `json:"abbreviation"`
}

// Splits contains batting splits
type Splits struct {
	Batter string `json:"batter"` // L/R
	Pitcher string `json:"pitcher"` // L/R
}

// PlayEvent represents a pitch or play event
type PlayEvent struct {
	Details   EventDetails `json:"details"`
	Count     Count        `json:"count"`
	PitchData *PitchData   `json:"pitchData,omitempty"`
	HitData   *HitData     `json:"hitData,omitempty"`
	IsPitch   bool         `json:"isPitch"`
	Type      string       `json:"type"`
}

// PitchCoordinates contains plate-crossing location data
type PitchCoordinates struct {
	PX float64 `json:"pX"` // horizontal plate location in feet (0 = center)
	PZ float64 `json:"pZ"` // vertical plate location in feet from ground
	X  float64 `json:"x"`  // pixel X on 250px strike zone graphic
	Y  float64 `json:"y"`  // pixel Y on 250px strike zone graphic
}

// PitchBreaks contains pitch movement data
type PitchBreaks struct {
	SpinRate             int     `json:"spinRate"`
	SpinDirection        int     `json:"spinDirection"`
	BreakVertical        float64 `json:"breakVertical"`
	BreakVerticalInduced float64 `json:"breakVerticalInduced"`
	BreakHorizontal      float64 `json:"breakHorizontal"`
	BreakAngle           float64 `json:"breakAngle"`
	BreakLength          float64 `json:"breakLength"`
}

// PitchData contains pitch tracking data
type PitchData struct {
	StartSpeed       float64           `json:"startSpeed"`
	EndSpeed         float64           `json:"endSpeed"`
	StrikeZoneTop    float64           `json:"strikeZoneTop"`
	StrikeZoneBottom float64           `json:"strikeZoneBottom"`
	Zone             int               `json:"zone"`
	Coordinates      *PitchCoordinates `json:"coordinates,omitempty"`
	Breaks           *PitchBreaks      `json:"breaks,omitempty"`
	PlateTime        float64           `json:"plateTime"`
	Extension        float64           `json:"extension"`
}

// HitCoordinates contains batted ball field location
type HitCoordinates struct {
	CoordX float64 `json:"coordX"`
	CoordY float64 `json:"coordY"`
}

// HitData contains batted ball tracking data
type HitData struct {
	LaunchSpeed  float64         `json:"launchSpeed"`
	LaunchAngle  float64         `json:"launchAngle"`
	TotalDistance float64        `json:"totalDistance"`
	Trajectory   string          `json:"trajectory"`
	Hardness     string          `json:"hardness"`
	Location     string          `json:"location"`
	Coordinates  *HitCoordinates `json:"coordinates,omitempty"`
}

// PitchType represents a pitch classification
type PitchType struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// EventDetails contains pitch details
type EventDetails struct {
	Description string     `json:"description"`
	Event       string     `json:"event"`
	EventType   string     `json:"eventType"`
	Code        string     `json:"code"`
	BallColor   string     `json:"ballColor,omitempty"`
	IsStrike    bool       `json:"isStrike,omitempty"`
	IsBall      bool       `json:"isBall,omitempty"`
	StartSpeed  float64    `json:"startSpeed,omitempty"`
	EndSpeed    float64    `json:"endSpeed,omitempty"`
	PitchType   *PitchType `json:"type,omitempty"`
	IsInPlay    bool       `json:"isInPlay,omitempty"`
}

// Linescore contains inning-by-inning scoring
type Linescore struct {
	CurrentInning     int               `json:"currentInning"`
	CurrentInningOrdinal string         `json:"currentInningOrdinal"`
	InningState       string            `json:"inningState"` // Top, Middle, Bottom, End
	Innings           []Inning          `json:"innings"`
	Teams             LinescoreTeams    `json:"teams"`
	Defense           Defense           `json:"defense,omitempty"`
	Offense           Offense           `json:"offense,omitempty"`
	Balls             int               `json:"balls"`
	Strikes           int               `json:"strikes"`
	Outs              int               `json:"outs"`
}

// Inning represents one inning of play
type Inning struct {
	Num        int         `json:"num"`
	OrdinalNum string      `json:"ordinalNum"`
	Home       InningScore `json:"home"`
	Away       InningScore `json:"away"`
}

// InningScore contains runs/hits/errors for half inning
type InningScore struct {
	Runs       *int `json:"runs,omitempty"`
	Hits       int  `json:"hits,omitempty"`
	Errors     int  `json:"errors,omitempty"`
	LeftOnBase int  `json:"leftOnBase,omitempty"`
}

// RunsVal returns the runs value, defaulting to 0 if nil (unplayed).
func (s InningScore) RunsVal() int {
	if s.Runs == nil {
		return 0
	}
	return *s.Runs
}

// WasPlayed returns true if this half-inning was actually played.
func (s InningScore) WasPlayed() bool {
	return s.Runs != nil
}

// LinescoreTeams contains team totals
type LinescoreTeams struct {
	Home LinescoreTeam `json:"home"`
	Away LinescoreTeam `json:"away"`
}

// LinescoreTeam contains runs/hits/errors totals
type LinescoreTeam struct {
	Runs       int `json:"runs"`
	Hits       int `json:"hits"`
	Errors     int `json:"errors"`
	LeftOnBase int `json:"leftOnBase"`
}

// Defense contains current defensive players
type Defense struct {
	Pitcher  Player `json:"pitcher"`
	Catcher  Player `json:"catcher"`
	First    Player `json:"first"`
	Second   Player `json:"second"`
	Third    Player `json:"third"`
	Shortstop Player `json:"shortstop"`
	Left     Player `json:"left"`
	Center   Player `json:"center"`
	Right    Player `json:"right"`
}

// Offense contains current baserunners and batter
type Offense struct {
	Batter Player  `json:"batter"`
	OnDeck Player  `json:"onDeck"`
	First  *Player `json:"first,omitempty"`
	Second *Player `json:"second,omitempty"`
	Third  *Player `json:"third,omitempty"`
}

// BoxscoreInfo contains label/value pairs from the boxscore info array
type BoxscoreInfo struct {
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
}

// Boxscore contains game statistics
type Boxscore struct {
	Teams BoxscoreTeams `json:"teams"`
	Info  []BoxscoreInfo `json:"info,omitempty"`
}

// BoxscoreTeams contains both teams' boxscores
type BoxscoreTeams struct {
	Away BoxscoreTeam `json:"away"`
	Home BoxscoreTeam `json:"home"`
}

// BoxscoreTeam contains team statistics
type BoxscoreTeam struct {
	Team         Team                      `json:"team"`
	TeamStats    TeamStats                 `json:"teamStats"`
	Players      map[string]BoxscorePlayer `json:"players"`
	Pitchers     []int                     `json:"pitchers"`
	BattingOrder []int                     `json:"battingOrder"`
	Batters      []int                     `json:"batters"`
}

// TeamStats contains team-level stats
type TeamStats struct {
	Batting BattingStats `json:"batting"`
	Pitching PitchingStats `json:"pitching"`
}

// BattingStats contains batting statistics
type BattingStats struct {
	Runs        int `json:"runs"`
	Hits        int `json:"hits"`
	Errors      int `json:"errors"`
	LeftOnBase  int `json:"leftOnBase"`
	AtBats      int `json:"atBats"`
	HomeRuns    int `json:"homeRuns"`
	RBI         int `json:"rbi"`
	BaseOnBalls int `json:"baseOnBalls"`
	StrikeOuts  int `json:"strikeOuts"`
	Doubles     int `json:"doubles"`
	Triples     int `json:"triples"`
}

// PitchingStats contains pitching statistics
type PitchingStats struct {
	Runs           int    `json:"runs"`
	Hits           int    `json:"hits"`
	Errors         int    `json:"errors"`
	InningsPitched string `json:"inningsPitched"`
	PitchesThrown  int    `json:"pitchesThrown"`
	EarnedRuns     int    `json:"earnedRuns"`
	BaseOnBalls    int    `json:"baseOnBalls"`
	StrikeOuts     int    `json:"strikeOuts"`
	HomeRuns       int    `json:"homeRuns"`
	Note           string `json:"note,omitempty"`
}

// BoxscorePlayer contains individual player stats
type BoxscorePlayer struct {
	Person       Player             `json:"person"`
	JerseyNumber string             `json:"jerseyNumber,omitempty"`
	Stats        PlayerStats        `json:"stats"`
	SeasonStats  *PlayerSeasonStats `json:"seasonStats,omitempty"`
	Status       PlayerStatus       `json:"status"`
	GameStatus   PlayerGameStatus   `json:"gameStatus"`
	BattingOrder string             `json:"battingOrder,omitempty"`
	Position     Position           `json:"position"`
	AllPositions []Position         `json:"allPositions,omitempty"`
}

// PlayerStats contains batting or pitching stats
type PlayerStats struct {
	Batting  *BattingStats  `json:"batting,omitempty"`
	Pitching *PitchingStats `json:"pitching,omitempty"`
}

// PlayerStatus contains player position info
type PlayerStatus struct {
	Code string `json:"code"`
}

// PlayerGameStatus contains game-specific status
type PlayerGameStatus struct {
	IsCurrentBatter  bool `json:"isCurrentBatter"`
	IsCurrentPitcher bool `json:"isCurrentPitcher"`
	IsOnBench        bool `json:"isOnBench"`
	IsSubstitute     bool `json:"isSubstitute"`
}

// PlayerSeasonStats contains season-level stats
type PlayerSeasonStats struct {
	Pitching *SeasonPitchingStats `json:"pitching,omitempty"`
	Batting  *SeasonBattingStats  `json:"batting,omitempty"`
}

// SeasonPitchingStats contains season pitching stats
type SeasonPitchingStats struct {
	Wins       int    `json:"wins"`
	Losses     int    `json:"losses"`
	Saves      int    `json:"saves"`
	Era        string `json:"era"`
	StrikeOuts int    `json:"strikeOuts"`
}

// SeasonBattingStats contains season batting stats
type SeasonBattingStats struct {
	Avg      string `json:"avg"`
	HomeRuns int    `json:"homeRuns"`
	Rbi      int    `json:"rbi"`
	OPS      string `json:"ops"`
}

// Decisions contains win/loss/save decisions for a game
type Decisions struct {
	Winner *DecisionPitcher `json:"winner,omitempty"`
	Loser  *DecisionPitcher `json:"loser,omitempty"`
	Save   *DecisionPitcher `json:"save,omitempty"`
}

// DecisionPitcher represents a pitcher in a game decision
type DecisionPitcher struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
}

// GameDataPlayer contains player metadata from gameData.players
type GameDataPlayer struct {
	ID           int    `json:"id"`
	FullName     string `json:"fullName"`
	BoxscoreName string `json:"boxscoreName"`
}

// GameData contains pre-game and venue data
type GameData struct {
	Game             GameInfo                    `json:"game"`
	Datetime         DatetimeInfo                `json:"datetime"`
	Status           GameStatus                  `json:"status"`
	Teams            GameTeams                   `json:"teams"`
	Venue            Venue                       `json:"venue"`
	ProbablePitchers ProbablePitchers            `json:"probablePitchers,omitempty"`
	Players          map[string]GameDataPlayer   `json:"players,omitempty"`
}

// GameInfo contains basic game info
type GameInfo struct {
	Pk     int    `json:"pk"`
	Type   string `json:"type"`
	Season string `json:"season"`
}

// DatetimeInfo contains game timing
type DatetimeInfo struct {
	DateTime     time.Time `json:"dateTime"`
	OriginalDate string    `json:"originalDate"`
	Time         string    `json:"time"`
	Ampm         string    `json:"ampm"`
}

// GameTeams contains full team info
type GameTeams struct {
	Away FullTeamInfo `json:"away"`
	Home FullTeamInfo `json:"home"`
}

// FullTeamInfo contains complete team details
type FullTeamInfo struct {
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	TeamName     string       `json:"teamName"`
	Link         string       `json:"link"`
	LeagueRecord LeagueRecord `json:"leagueRecord"`
	Record       *SeriesRecord  `json:"record,omitempty"` // For playoff series records
}

// SeriesRecord contains playoff series record
type SeriesRecord struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

// Venue contains stadium information
type Venue struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Location Location `json:"location,omitempty"`
}

// Location contains venue location details
type Location struct {
	City         string `json:"city"`
	State        string `json:"state"`
	StateAbbrev  string `json:"stateAbbrev"`
}

// ProbablePitchers contains starting pitcher info
type ProbablePitchers struct {
	Away ProbablePitcher `json:"away,omitempty"`
	Home ProbablePitcher `json:"home,omitempty"`
}

// ProbablePitcher contains pitcher details
type ProbablePitcher struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
	Note     string `json:"note,omitempty"`
}

// MetaData contains polling information
type MetaData struct {
	Wait      int    `json:"wait"`      // seconds to wait before next poll
	TimeStamp string `json:"timeStamp"` // for differential updates
}

// ScheduleResponse represents the API response for schedule
type ScheduleResponse struct {
	Dates []ScheduleDate `json:"dates"`
}

// ScheduleDate contains games for a specific date
type ScheduleDate struct {
	Date  string `json:"date"`
	Games []Game `json:"games"`
}

// StandingsResponse represents the API response for standings
type StandingsResponse struct {
	Records []DivisionStandings `json:"records"`
}

// DivisionStandings contains standings for one division
type DivisionStandings struct {
	Division     Division      `json:"division"`
	TeamRecords  []TeamRecord  `json:"teamRecords"`
}

// Division represents a division
type Division struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	NameShort string `json:"nameShort"`
}

// TeamRecord contains a team's standing
type TeamRecord struct {
	Team            Team   `json:"team"`
	LeagueRecord    LeagueRecord `json:"leagueRecord"`
	GamesBack       string `json:"gamesBack"`
	WildCardGamesBack string `json:"wildCardGamesBack"`
	DivisionRank    string `json:"divisionRank"`
	WildCardRank    string `json:"wildCardRank"`
	Wins            int    `json:"wins"`
	Losses          int    `json:"losses"`
	WinningPercentage string `json:"winningPercentage"`
	Streak          Streak `json:"streak"`
	LastTenGames    LastTenRecord `json:"records"`
}

// Streak contains win/loss streak info
type Streak struct {
	StreakType   string `json:"streakType"` // W or L
	StreakNumber int    `json:"streakNumber"`
	StreakCode   string `json:"streakCode"` // W3, L2, etc
}

// LastTenRecord contains recent record
type LastTenRecord struct {
	SplitRecords []SplitRecord `json:"splitRecords"`
}

// SplitRecord contains a specific record split
type SplitRecord struct {
	Type         string       `json:"type"`
	Wins         int          `json:"wins"`
	Losses       int          `json:"losses"`
	Pct          string       `json:"pct"`
}

// WBCPool represents a pool in the World Baseball Classic
type WBCPool struct {
	Name  string          // "Pool A", "Pool B", etc.
	Teams []WBCTeamRecord
}

// WBCTeamRecord represents a team's record in WBC pool play
type WBCTeamRecord struct {
	Team   Team
	Wins   int
	Losses int
}
