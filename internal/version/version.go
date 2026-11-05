// Package version reports which DeckDrop release is running.
package version

import (
	_ "embed"
	"strings"
)

// The release workflow writes the tag's version into version.txt before
// building (see build/set-version.sh). Local builds keep "dev".
//
//go:embed version.txt
var raw string

// String returns the release, such as "0.1.0" or "0.1.0-rc.5", or "dev" for a
// local build.
func String() string {
	if version := strings.TrimSpace(raw); version != "" {
		return version
	}
	return "dev"
}
