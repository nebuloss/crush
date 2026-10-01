package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskAgentToolDescription(t *testing.T) {
	t.Parallel()

	got := taskAgentToolDescription(agentToolDescription, []string{"glob", "view", "bash"})
	require.Contains(t, got, "has access to the following tools: glob, view, bash.")
	require.NotContains(t, got, "glob, grep, ls, view")

	got = taskAgentToolDescription("Launch an agent.", []string{"bash"})
	require.Equal(t, "Launch an agent.\n\nThe agent has access to the following tools: bash.\n", got)
}
