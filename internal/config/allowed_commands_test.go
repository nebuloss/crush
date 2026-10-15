package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandAllowed(t *testing.T) {
	t.Parallel()

	// Unset allows everything.
	require.True(t, Config{}.CommandAllowed("curl"))
	require.True(t, Config{Permissions: &Permissions{}}.CommandAllowed("sudo"))
	require.Equal(t, []string{AllowAllCommands}, (&Config{}).AllowedCommands())

	// An explicit list allows only what it names.
	c := Config{Permissions: &Permissions{AllowedCommands: []string{"ssh", "curl"}}}
	require.True(t, c.CommandAllowed("curl"))
	require.False(t, c.CommandAllowed("wget"))

	// Unless it includes the wildcard.
	c = Config{Permissions: &Permissions{AllowedCommands: []string{"ssh", "*"}}}
	require.True(t, c.CommandAllowed("wget"))
}
