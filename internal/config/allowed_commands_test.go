package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandAllowed(t *testing.T) {
	t.Parallel()

	require.False(t, Config{}.CommandAllowed("curl"))

	c := Config{Permissions: &Permissions{AllowedCommands: []string{"ssh", "curl"}}}
	require.True(t, c.CommandAllowed("curl"))
	require.False(t, c.CommandAllowed("wget"))
}
