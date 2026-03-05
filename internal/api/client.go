package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

const (
	baseURL = "https://statsapi.mlb.com"
	timeout = 10 * time.Second
)

// Client handles MLB Stats API requests
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchSchedule retrieves the game schedule for a specific date
func (c *Client) FetchSchedule(date time.Time) ([]Game, error) {
	dateStr := date.Format("01/02/2006")
	url := fmt.Sprintf("%s/api/v1/schedule?sportId=1&hydrate=team,linescore&date=%s", baseURL, dateStr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching schedule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var scheduleResp ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResp); err != nil {
		return nil, fmt.Errorf("decoding schedule response: %w", err)
	}

	if len(scheduleResp.Dates) == 0 {
		return []Game{}, nil
	}

	return scheduleResp.Dates[0].Games, nil
}

// FetchGame retrieves live game data
func (c *Client) FetchGame(gameID int) (*Game, error) {
	url := fmt.Sprintf("%s/api/v1.1/game/%d/feed/live", baseURL, gameID)

	// Debug logging
	logFile, _ := os.OpenFile("/tmp/go-playball-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if logFile != nil {
		defer logFile.Close()
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Fetching game %d from URL: %s", gameID, url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Error fetching game %d: %v", gameID, err)
		}
		return nil, fmt.Errorf("fetching game: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("API returned status %d for game %d: %s", resp.StatusCode, gameID, string(body))
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read body to byte array so we can log it
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Game %d response length: %d bytes", gameID, len(bodyBytes))
	}

	var game Game
	if err := json.Unmarshal(bodyBytes, &game); err != nil {
		if logFile != nil {
			logger := log.New(logFile, "", log.LstdFlags)
			logger.Printf("Error decoding game %d: %v", gameID, err)
			previewLen := 500
			if len(bodyBytes) < previewLen {
				previewLen = len(bodyBytes)
			}
			logger.Printf("First %d chars of response: %s", previewLen, string(bodyBytes[:previewLen]))
		}
		return nil, fmt.Errorf("decoding game response: %w", err)
	}

	if logFile != nil {
		logger := log.New(logFile, "", log.LstdFlags)
		logger.Printf("Successfully decoded game %d. Status: '%s', DetailedState: '%s', LiveData present: %v",
			gameID, game.Status.AbstractGameState, game.Status.DetailedState, game.LiveData != nil)
		logger.Printf("  Team names: Away='%s', Home='%s'", game.Teams.Away.Team.Name, game.Teams.Home.Team.Name)
		if game.GameData != nil {
			logger.Printf("  GameData.Status: '%s', DetailedState: '%s'",
				game.GameData.Status.AbstractGameState, game.GameData.Status.DetailedState)
			logger.Printf("  GameData team names: Away='%s', Home='%s'",
				game.GameData.Teams.Away.Name, game.GameData.Teams.Home.Name)
		}
		if game.LiveData != nil {
			logger.Printf("  Decisions present: %v", game.LiveData.Decisions.Winner != nil)
			logger.Printf("  Linescore present: %v", len(game.LiveData.Linescore.Innings) > 0)
		}
	}

	return &game, nil
}

// logDebug writes a formatted message to the debug log file
func (c *Client) logDebug(format string, args ...interface{}) {
	logFile, err := os.OpenFile("/tmp/go-playball-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf(format, args...)
}

// FetchGameIncremental fetches a game, using diffPatch if a timestamp is available.
// Returns the parsed game, raw JSON bytes, and any error.
// If incremental fetch fails, falls back to full fetch automatically.
func (c *Client) FetchGameIncremental(gameID int, currentJSON []byte, timestamp string) (*Game, []byte, error) {
	if timestamp != "" && len(currentJSON) > 0 {
		game, rawJSON, err := c.fetchDiffPatch(gameID, currentJSON, timestamp)
		if err == nil {
			return game, rawJSON, nil
		}
		c.logDebug("diffPatch failed for game %d: %v, falling back to full fetch", gameID, err)
	}

	return c.fetchGameFull(gameID)
}

// fetchDiffPatch attempts an incremental update via JSON Patch
func (c *Client) fetchDiffPatch(gameID int, currentJSON []byte, timestamp string) (*Game, []byte, error) {
	url := fmt.Sprintf("%s/api/v1.1/game/%d/feed/live/diffPatch?startTimecode=%s",
		baseURL, gameID, timestamp)

	c.logDebug("Fetching diffPatch for game %d: %s", gameID, url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching diffPatch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("diffPatch returned status %d", resp.StatusCode)
	}

	patchBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading diffPatch body: %w", err)
	}

	c.logDebug("diffPatch response for game %d: %d bytes (original: %d bytes)",
		gameID, len(patchBytes), len(currentJSON))

	patch, err := jsonpatch.DecodePatch(patchBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding patch: %w", err)
	}

	updatedJSON, err := patch.Apply(currentJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("applying patch: %w", err)
	}

	var game Game
	if err := json.Unmarshal(updatedJSON, &game); err != nil {
		return nil, nil, fmt.Errorf("deserializing patched game: %w", err)
	}

	c.logDebug("diffPatch applied for game %d, updated JSON: %d bytes", gameID, len(updatedJSON))

	return &game, updatedJSON, nil
}

// fetchGameFull does a complete fetch and returns both parsed and raw JSON
func (c *Client) fetchGameFull(gameID int) (*Game, []byte, error) {
	url := fmt.Sprintf("%s/api/v1.1/game/%d/feed/live", baseURL, gameID)

	c.logDebug("Full fetch for game %d: %s", gameID, url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching game: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading body: %w", err)
	}

	var game Game
	if err := json.Unmarshal(bodyBytes, &game); err != nil {
		return nil, nil, fmt.Errorf("decoding game: %w", err)
	}

	c.logDebug("Full fetch for game %d: %d bytes", gameID, len(bodyBytes))

	return &game, bodyBytes, nil
}

// FetchStandings retrieves current MLB standings
func (c *Client) FetchStandings() ([]DivisionStandings, error) {
	year := time.Now().Year()
	url := fmt.Sprintf("%s/api/v1/standings?leagueId=103,104&season=%d&standingsTypes=regularSeason&hydrate=division,team",
		baseURL, year)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching standings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var standingsResp StandingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&standingsResp); err != nil {
		return nil, fmt.Errorf("decoding standings response: %w", err)
	}

	return standingsResp.Records, nil
}
