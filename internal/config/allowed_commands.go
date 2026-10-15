package config

import "slices"

// AllowAllCommands is the permissions.allowed_commands entry that lifts the
// bash tool's whole blocklist.
const AllowAllCommands = "*"

// AllowedCommands returns permissions.allowed_commands: the commands the user
// has removed from the bash tool's built-in blocklist.
//
// Unset, it defaults to AllowAllCommands: this fork does not block commands
// unless asked to. Setting an explicit list restores upstream's blocklist
// minus the listed commands. Either way the permission prompt still applies.
//
// The Permissions section is a pointer and is nil whenever the user has not
// configured one, which is the common case, so this tolerates both a nil
// Config and a nil section rather than making every caller check.
func (c *Config) AllowedCommands() []string {
	if c == nil || c.Permissions == nil || c.Permissions.AllowedCommands == nil {
		return []string{AllowAllCommands}
	}
	return c.Permissions.AllowedCommands
}

// CommandAllowed reports whether name is allowed by
// permissions.allowed_commands, by name or by AllowAllCommands. It has a
// value receiver so prompt templates, which receive Config by value, can
// call it.
func (c Config) CommandAllowed(name string) bool {
	cmds := c.AllowedCommands()
	return slices.Contains(cmds, AllowAllCommands) || slices.Contains(cmds, name)
}
