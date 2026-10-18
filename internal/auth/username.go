package auth

import (
	"regexp"
	"strings"
)

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,39}$`)

// NormalizeUsername usa a mesma identidade para busca e limite de tentativas.
func NormalizeUsername(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, usernamePattern.MatchString(value)
}
