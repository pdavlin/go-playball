package buildinfo

import "fmt"

// These are set via -ldflags at build time.
// Example: go build -ldflags "-X github.com/pdavlin/go-playball/internal/buildinfo.version=v1.2.0"
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Version returns the build version string.
func Version() string {
	return version
}

// String returns a formatted multi-field build info string.
func String() string {
	return fmt.Sprintf("go-playball %s (commit: %s, built: %s)", version, commit, date)
}
