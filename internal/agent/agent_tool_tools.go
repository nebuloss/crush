package agent

import (
	"regexp"
	"strings"
)

var agentToolListRE = regexp.MustCompile(`has access to the following tools: [^.]*\.`)

// taskAgentToolDescription rewrites the agent tool's description so the tool
// list it advertises is the sub-agent's real one. Upstream hardcodes a
// read-only list, which goes stale once options.task_agent_tools grants more,
// and the calling model then either avoids delegating work that needs those
// tools or, worse, briefs the sub-agent with tools it does not have.
func taskAgentToolDescription(base string, tools []string) string {
	list := "has access to the following tools: " + strings.Join(tools, ", ") + "."
	if agentToolListRE.MatchString(base) {
		return agentToolListRE.ReplaceAllLiteralString(base, list)
	}
	return strings.TrimRight(base, "\n") + "\n\nThe agent " + list + "\n"
}
