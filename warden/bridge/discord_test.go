package bridge

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	cases := map[string]string{
		"Stone Age":                           "Stone Age",
		"was slain by **Bob** using [Sword]":  `was slain by \*\*Bob\*\* using \[Sword\]`,
		"[click](https://evil.example)":       `\[click\](https://evil.example)`,
		"||spoiler|| _x_ ~~y~~ `z` a\\b":      `\|\|spoiler\|\| \_x\_ \~\~y\~\~ \` + "`z\\`" + ` a\\b`,
		"tried to swim in lava (Bob's fault)": "tried to swim in lava (Bob's fault)",
	}
	for in, want := range cases {
		if got := escapeMarkdown(in); got != want {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", in, got, want)
		}
	}
}
