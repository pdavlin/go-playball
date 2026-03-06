package ui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

// TeamColors holds primary and secondary colors for a team
type TeamColors struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
}

// teamColorMap maps team names to their official colors
var teamColorMap = map[string]TeamColors{
	// AL East
	"Yankees":   {Primary: lipgloss.Color("#003087"), Secondary: lipgloss.Color("#E4002B")},
	"Red Sox":   {Primary: lipgloss.Color("#BD3039"), Secondary: lipgloss.Color("#0C2340")},
	"Blue Jays": {Primary: lipgloss.Color("#134A8E"), Secondary: lipgloss.Color("#E8291C")},
	"Rays":      {Primary: lipgloss.Color("#092C5C"), Secondary: lipgloss.Color("#8FBCE6")},
	"Orioles":   {Primary: lipgloss.Color("#DF4601"), Secondary: lipgloss.Color("#000000")},

	// AL Central
	"White Sox":  {Primary: lipgloss.Color("#27251F"), Secondary: lipgloss.Color("#C4CED4")},
	"Guardians":  {Primary: lipgloss.Color("#E31937"), Secondary: lipgloss.Color("#0C2340")},
	"Tigers":     {Primary: lipgloss.Color("#0C2340"), Secondary: lipgloss.Color("#FA4616")},
	"Royals":     {Primary: lipgloss.Color("#004687"), Secondary: lipgloss.Color("#BD9B60")},
	"Twins":      {Primary: lipgloss.Color("#002B5C"), Secondary: lipgloss.Color("#D31145")},

	// AL West
	"Astros":    {Primary: lipgloss.Color("#002D62"), Secondary: lipgloss.Color("#EB6E1F")},
	"Angels":    {Primary: lipgloss.Color("#BA0021"), Secondary: lipgloss.Color("#003263")},
	"Athletics": {Primary: lipgloss.Color("#003831"), Secondary: lipgloss.Color("#EFB21E")},
	"Mariners":  {Primary: lipgloss.Color("#0C2C56"), Secondary: lipgloss.Color("#005C5C")},
	"Rangers":   {Primary: lipgloss.Color("#003278"), Secondary: lipgloss.Color("#C0111F")},

	// NL East
	"Braves":     {Primary: lipgloss.Color("#CE1141"), Secondary: lipgloss.Color("#13274F")},
	"Marlins":    {Primary: lipgloss.Color("#00A3E0"), Secondary: lipgloss.Color("#EF3340")},
	"Mets":       {Primary: lipgloss.Color("#002D72"), Secondary: lipgloss.Color("#FF5910")},
	"Phillies":   {Primary: lipgloss.Color("#E81828"), Secondary: lipgloss.Color("#002D72")},
	"Nationals":  {Primary: lipgloss.Color("#AB0003"), Secondary: lipgloss.Color("#14225A")},

	// NL Central
	"Cubs":      {Primary: lipgloss.Color("#0E3386"), Secondary: lipgloss.Color("#CC3433")},
	"Reds":      {Primary: lipgloss.Color("#C6011F"), Secondary: lipgloss.Color("#000000")},
	"Brewers":   {Primary: lipgloss.Color("#12284B"), Secondary: lipgloss.Color("#FFC52F")},
	"Pirates":   {Primary: lipgloss.Color("#27251F"), Secondary: lipgloss.Color("#FDB827")},
	"Cardinals": {Primary: lipgloss.Color("#C41E3A"), Secondary: lipgloss.Color("#0C2340")},

	// NL West
	"Diamondbacks": {Primary: lipgloss.Color("#A71930"), Secondary: lipgloss.Color("#E3D4AD")},
	"Rockies":      {Primary: lipgloss.Color("#33006F"), Secondary: lipgloss.Color("#C4CED4")},
	"Dodgers":      {Primary: lipgloss.Color("#005A9C"), Secondary: lipgloss.Color("#EF3E42")},
	"Padres":       {Primary: lipgloss.Color("#2F241D"), Secondary: lipgloss.Color("#FFC425")},
	"Giants":       {Primary: lipgloss.Color("#FD5A1E"), Secondary: lipgloss.Color("#27251F")},

	// WBC national teams
	"Australia":              {Primary: lipgloss.Color("#00843D"), Secondary: lipgloss.Color("#FFCD00")},
	"Brazil":                 {Primary: lipgloss.Color("#009C3B"), Secondary: lipgloss.Color("#FFDF00")},
	"Canada":                 {Primary: lipgloss.Color("#FF0000"), Secondary: lipgloss.Color("#FFFFFF")},
	"Chinese Taipei":         {Primary: lipgloss.Color("#0036B5"), Secondary: lipgloss.Color("#E80000")},
	"Colombia":               {Primary: lipgloss.Color("#FCD116"), Secondary: lipgloss.Color("#003893")},
	"Cuba":                   {Primary: lipgloss.Color("#002590"), Secondary: lipgloss.Color("#CC0D0D")},
	"Czechia":                {Primary: lipgloss.Color("#11457E"), Secondary: lipgloss.Color("#D7141A")},
	"Dominican Republic":     {Primary: lipgloss.Color("#002D62"), Secondary: lipgloss.Color("#CE1126")},
	"Great Britain":          {Primary: lipgloss.Color("#012169"), Secondary: lipgloss.Color("#C8102E")},
	"Israel":                 {Primary: lipgloss.Color("#0038B8"), Secondary: lipgloss.Color("#FFFFFF")},
	"Italy":                  {Primary: lipgloss.Color("#009246"), Secondary: lipgloss.Color("#CE2B37")},
	"Japan":                  {Primary: lipgloss.Color("#BC002D"), Secondary: lipgloss.Color("#FFFFFF")},
	"Korea":                  {Primary: lipgloss.Color("#003478"), Secondary: lipgloss.Color("#CD2E3A")},
	"Kingdom of the Netherlands": {Primary: lipgloss.Color("#FF6F00"), Secondary: lipgloss.Color("#21468B")},
	"Mexico":                 {Primary: lipgloss.Color("#006847"), Secondary: lipgloss.Color("#CE1126")},
	"Nicaragua":              {Primary: lipgloss.Color("#0067C6"), Secondary: lipgloss.Color("#FFFFFF")},
	"Panama":                 {Primary: lipgloss.Color("#005BA6"), Secondary: lipgloss.Color("#D21034")},
	"Puerto Rico":            {Primary: lipgloss.Color("#003DA5"), Secondary: lipgloss.Color("#E81B23")},
	"United States":          {Primary: lipgloss.Color("#002868"), Secondary: lipgloss.Color("#BF0A30")},
	"Venezuela":              {Primary: lipgloss.Color("#FFCC00"), Secondary: lipgloss.Color("#003DA5")},
}

// GetTeamColors returns the colors for a team, falling back to default colors.
// Colors are automatically lightened for dark backgrounds to ensure readability.
func GetTeamColors(teamName string) TeamColors {
	var colors TeamColors

	// Try exact match first
	if c, ok := teamColorMap[teamName]; ok {
		colors = c
	} else {
		// Try partial match
		found := false
		for key, c := range teamColorMap {
			if contains(teamName, key) {
				colors = c
				found = true
				break
			}
		}
		if !found {
			return TeamColors{
				Primary:   lipgloss.Color("#00D9FF"),
				Secondary: lipgloss.Color("#FFB86C"),
			}
		}
	}

	if darkMode {
		colors.Primary = ensureMinLuminance(colors.Primary, 0.15)
		colors.Secondary = ensureMinLuminance(colors.Secondary, 0.12)
	}

	return colors
}

// darkMode tracks whether the terminal has a dark background.
// Set during init via DetectDarkMode.
var darkMode = true

// DetectDarkMode checks the terminal background and sets the darkMode flag.
func DetectDarkMode(dark bool) {
	darkMode = dark
}

// ensureMinLuminance lightens a hex color if its relative luminance
// is below the given threshold. Uses WCAG relative luminance formula.
func ensureMinLuminance(c lipgloss.Color, minLum float64) lipgloss.Color {
	hex := string(c)
	r, g, b, ok := parseHex(hex)
	if !ok {
		return c
	}

	lum := relativeLuminance(r, g, b)
	if lum >= minLum {
		return c
	}

	// Blend toward white until we reach the target luminance.
	// Binary search for the blend factor.
	lo, hi := 0.0, 1.0
	for i := 0; i < 16; i++ {
		mid := (lo + hi) / 2
		mr := blend(r, 255, mid)
		mg := blend(g, 255, mid)
		mb := blend(b, 255, mid)
		if relativeLuminance(mr, mg, mb) < minLum {
			lo = mid
		} else {
			hi = mid
		}
	}

	factor := hi
	nr := blend(r, 255, factor)
	ng := blend(g, 255, factor)
	nb := blend(b, 255, factor)

	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", nr, ng, nb))
}

func parseHex(hex string) (r, g, b uint8, ok bool) {
	if len(hex) == 0 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	hex = hex[1:]
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(rv), uint8(gv), uint8(bv), true
}

// relativeLuminance computes WCAG 2.0 relative luminance.
func relativeLuminance(r, g, b uint8) float64 {
	rl := linearize(float64(r) / 255.0)
	gl := linearize(float64(g) / 255.0)
	bl := linearize(float64(b) / 255.0)
	return 0.2126*rl + 0.7152*gl + 0.0722*bl
}

func linearize(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func blend(a, b uint8, t float64) uint8 {
	v := float64(a)*(1-t) + float64(b)*t
	if v > 255 {
		v = 255
	}
	return uint8(v)
}

// nameOverrides maps full names to shorter display names for cases where
// the API name is too long or differs from the common short form.
var nameOverrides = map[string]string{
	"Arizona Diamondbacks":       "D-backs",
	"Kingdom of the Netherlands": "Netherlands",
	"Dominican Republic":         "Dominican Rep.",
	"Czech Republic":             "Czech Rep.",
}

// GetTeamShortName extracts the display name from a full team name.
// e.g. "New York Mets" -> "Mets", "Kingdom of the Netherlands" -> "Netherlands"
func GetTeamShortName(fullName string) string {
	if short, ok := nameOverrides[fullName]; ok {
		return short
	}
	for key := range teamColorMap {
		if contains(fullName, key) {
			return key
		}
	}
	return fullName
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
