package service

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	dashedUUID   = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	undashedUUID = regexp.MustCompile(`^[0-9a-f]{32}$`)
	usernameRe   = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
)

// NormalizeUUID canonicalizes a Minecraft UUID to lowercase dashed form.
// Plugins and Mojang endpoints disagree on dashes, so everything crossing
// Warden's boundary gets normalized once here rather than at each call site.
func NormalizeUUID(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if dashedUUID.MatchString(value) {
		return value, nil
	}
	if undashedUUID.MatchString(value) {
		return fmt.Sprintf("%s-%s-%s-%s-%s", value[0:8], value[8:12], value[12:16], value[16:20], value[20:32]), nil
	}
	return "", fmt.Errorf("%q is not a valid minecraft uuid", raw)
}

// ValidateUsername rejects anything Mojang would never issue. Usernames are
// display-only here — the UUID is the identity — but they end up in chat
// relays and audit rows, so they get checked before storage.
func ValidateUsername(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !usernameRe.MatchString(value) {
		return "", fmt.Errorf("%q is not a valid minecraft username", raw)
	}
	return value, nil
}
