package filecheck

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxNameBytes = 255

// DisplayName turns a client-supplied filename into a string that is safe
// to store and show. It is metadata only: never use it as a path.
func DisplayName(raw string) string {
	// Windows clients may send "C:\Users\a\b.png"; on Linux "\" is not a
	// separator, so filepath.Base would keep the whole thing.
	raw = strings.ReplaceAll(raw, `\`, "/")
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		raw = raw[i+1:]
	}

	var b strings.Builder
	for _, r := range strings.ToValidUTF8(raw, "") {
		// Drop control characters and bidi overrides such as U+202E,
		// which can make "gnp.exe" display as "exe.png".
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			continue
		}
		b.WriteRune(r)
	}
	name := strings.Trim(b.String(), " .")

	for len(name) > maxNameBytes { // cut on a rune boundary
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if name == "" {
		return "file"
	}
	return name
}
