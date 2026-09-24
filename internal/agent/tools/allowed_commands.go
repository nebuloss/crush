package tools

import (
	"strings"

	"github.com/charmbracelet/crush/internal/shell"
)

// This file implements permissions.allowed_commands: an opt-in escape hatch
// from the bash tool's built-in blocklist.
//
// It is deliberately self-contained so that the edits it requires in
// bash.go are limited to call signatures — the blocklist and the deny-rule
// table stay byte-identical to upstream, which keeps this patch applying
// cleanly across releases.

// allowedCommandSet indexes the commands a user has explicitly un-banned.
// Blank entries are dropped so a stray empty string in the config cannot
// match an empty argv.
func allowedCommandSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(allowed))
	for _, cmd := range allowed {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			set[cmd] = struct{}{}
		}
	}
	return set
}

// effectiveBannedCommands returns banned minus anything the user allowed. It
// exists so the bash tool's description advertises the blocklist the agent
// will actually hit, rather than claiming a command is banned when it isn't.
// The input slice is never mutated.
func effectiveBannedCommands(banned, allowed []string) []string {
	set := allowedCommandSet(allowed)
	if len(set) == 0 {
		return banned
	}
	out := make([]string, 0, len(banned))
	for _, cmd := range banned {
		if _, ok := set[cmd]; !ok {
			out = append(out, cmd)
		}
	}
	return out
}

// allowCommands wraps a set of deny rules so that any command named in
// allowed short-circuits to "not blocked" before the rules are consulted.
//
// Matching is on argv[0] only, which means allowing a command also lifts the
// subcommand rules for it: `permissions allow-command apt` re-enables
// `apt install`, not just bare `apt`. That is the least surprising reading of
// "allow this command", and it avoids a half-lifted state where a command is
// un-banned but its most useful invocation still fails.
//
// This does not grant the command permission to run — it only removes the
// hard block. The call still goes through the normal permission prompt unless
// the bash tool itself is in permissions.allowed_tools.
func allowCommands(funcs []shell.BlockFunc, allowed []string) []shell.BlockFunc {
	set := allowedCommandSet(allowed)
	if len(set) == 0 {
		return funcs
	}
	return []shell.BlockFunc{
		func(args []string) bool {
			if len(args) > 0 {
				if _, ok := set[args[0]]; ok {
					return false
				}
			}
			for _, blocked := range funcs {
				if blocked(args) {
					return true
				}
			}
			return false
		},
	}
}
