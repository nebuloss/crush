package agent

import "github.com/charmbracelet/crush/internal/permission"

// inheritPermissions makes a sub-agent session share its parent's approvals.
// It is feature-detected rather than added to permission.Service so test
// doubles of that interface need no change.
func inheritPermissions(p permission.Service, child, parent string) {
	if t, ok := p.(interface{ InheritSession(child, parent string) }); ok {
		t.InheritSession(child, parent)
	}
}
