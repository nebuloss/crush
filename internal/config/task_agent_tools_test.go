package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskAgentTools(t *testing.T) {
	t.Parallel()

	readOnly := resolveReadOnlyTools(allToolNames())

	t.Run("defaults to the read-only set", func(t *testing.T) {
		t.Parallel()
		c := &Config{Options: &Options{}}
		c.SetupAgents()
		require.Equal(t, readOnly, c.Agents[AgentTask].AllowedTools)
	})

	t.Run("adds configured tools", func(t *testing.T) {
		t.Parallel()
		c := &Config{Options: &Options{TaskAgentTools: []string{"bash", "job_output", "job_kill"}}}
		c.SetupAgents()
		got := c.Agents[AgentTask].AllowedTools
		require.Subset(t, got, readOnly)
		require.Subset(t, got, []string{"bash", "job_output", "job_kill"})
		require.NotContains(t, got, "edit")
	})

	t.Run("never grants agent or disabled tools", func(t *testing.T) {
		t.Parallel()
		c := &Config{Options: &Options{
			TaskAgentTools: []string{"agent", "bash", "write", "nonexistent"},
			DisabledTools:  []string{"write"},
		}}
		c.SetupAgents()
		got := c.Agents[AgentTask].AllowedTools
		require.Contains(t, got, "bash")
		require.NotContains(t, got, "agent")
		require.NotContains(t, got, "write")
		require.NotContains(t, got, "nonexistent")
	})
}
