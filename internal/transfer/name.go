package transfer

import (
	"strings"
	"unicode/utf8"
)

const (
	maxNameBytes            = 255
	forbiddenNameCharacters = `/\:*?"<>|`
)

// windowsDeviceNames cannot be used as file names on Windows, with or without
// an extension.
var windowsDeviceNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// ValidName reports whether name can be used for a received file.
func ValidName(name string) bool { return isPlainFileName(name) }

// isPlainFileName reports whether name is a single file name that is safe to
// create on Linux, macOS and Windows.
func isPlainFileName(name string) bool {
	if name == "" || len(name) > maxNameBytes || !utf8.ValidString(name) {
		return false
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return false
	}
	for _, character := range name {
		if character < ' ' || character == 0x7f || strings.ContainsRune(forbiddenNameCharacters, character) {
			return false
		}
	}

	stem, _, _ := strings.Cut(name, ".")
	return !windowsDeviceNames[strings.ToUpper(strings.TrimRight(stem, " "))]
}
