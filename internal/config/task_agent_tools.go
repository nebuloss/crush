package config

import "slices"

// taskAgentTools returns the tool set for the task agent, the sub-agent the
// agent tool launches: the upstream read-only set plus options.task_agent_tools.
//
// Extra tools are drawn from allowedTools, so a tool in disabled_tools stays
// disabled. The agent tool itself is never granted, so a sub-agent cannot
// spawn further sub-agents.
func (c *Config) taskAgentTools(allowedTools []string) []string {
	tools := resolveReadOnlyTools(allowedTools)
	if c.Options == nil {
		return tools
	}
	for _, name := range allowedTools {
		if name == "agent" || slices.Contains(tools, name) {
			continue
		}
		if slices.Contains(c.Options.TaskAgentTools, name) {
			tools = append(tools, name)
		}
	}
	return tools
}
