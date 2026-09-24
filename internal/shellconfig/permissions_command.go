package shellconfig

import (
	"io"
	"log/slog"
)

// permissionsAllowCommand removes commands from the bash tool's built-in
// blocklist by adding them to permissions.allowed_commands. Unlike allow,
// this takes shell command names (ssh, curl) rather than tool names.
func permissionsAllowCommand(b *ConfigBuilder, args []string, stderr io.Writer) error {
	if len(args) < 3 {
		return usage(stderr, "usage: permissions allow-command <command> [<command> ...]")
	}
	perms := b.section("permissions")
	allowed, _ := perms["allowed_commands"].([]any)

	for _, cmd := range args[2:] {
		if !containsAny(allowed, cmd) {
			allowed = append(allowed, cmd)
		}
	}
	perms["allowed_commands"] = allowed

	slog.Info("Commands allowed in shell config", "commands", args[2:])
	return nil
}
