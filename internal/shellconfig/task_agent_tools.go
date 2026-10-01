package shellconfig

// Registers "option task-agent-tool <name>" for options.task_agent_tools.
// Kept out of the optionSpecs literal so the patch does not touch it.
func init() {
	optionSpecs["task-agent-tool"] = optionSpec{jsonKey: "task_agent_tools", kind: optList}
}
