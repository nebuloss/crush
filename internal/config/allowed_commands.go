package config

import "slices"

// AllowedCommands returns permissions.allowed_commands: the commands the user
// has removed from the bash tool's built-in blocklist.
//
// The Permissions section is a pointer and is nil whenever the user has not
// configured one, which is the common case, so this tolerates both a nil
// Config and a nil section rather than making every caller check.
func (c *Config) AllowedCommands() []string {
	if c == nil || c.Permissions == nil {
		return nil
	}
	return c.Permissions.AllowedCommands
}

// CommandAllowed reports whether name is in permissions.allowed_commands.
// It has a value receiver so prompt templates, which receive Config by
// value, can call it.
func (c Config) CommandAllowed(name string) bool {
	return slices.Contains(c.AllowedCommands(), name)
}
