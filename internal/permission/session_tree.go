package permission

// Sub-agent sessions are children of the session that launched them. Without
// this, each one starts with no approvals: `crush run` auto-approves only its
// own session, so a sub-agent's first permission request blocks forever, and
// in the TUI "allow for session" covers only the one sub-agent that asked.
//
// A child therefore inherits its ancestors' approvals, and a persistent grant
// made from any session in the tree is recorded on the root, so it covers the
// main agent and every sub-agent it launches.

// maxSessionDepth bounds ancestor walks, guarding against a cycle.
const maxSessionDepth = 32

// InheritSession records parent as the parent of child, so child shares the
// approvals of parent and its ancestors.
func (s *permissionService) InheritSession(child, parent string) {
	if child == "" || parent == "" || child == parent {
		return
	}
	s.sessionParents.Store(child, parent)
}

// ancestors returns the parent chain of id, nearest first, excluding id.
func (s *permissionService) ancestors(id string) []string {
	var chain []string
	for range maxSessionDepth {
		parent, ok := s.sessionParents.Load(id)
		if !ok {
			break
		}
		id = parent.(string)
		chain = append(chain, id)
	}
	return chain
}

// rootSession returns the top-most ancestor of id, or id itself.
func (s *permissionService) rootSession(id string) string {
	if chain := s.ancestors(id); len(chain) > 0 {
		return chain[len(chain)-1]
	}
	return id
}

// ancestorAutoApproved reports whether any ancestor of id is auto-approved.
// The caller must hold autoApproveSessionsMu for reading.
func (s *permissionService) ancestorAutoApproved(id string) bool {
	for _, a := range s.ancestors(id) {
		if s.autoApproveSessions[a] {
			return true
		}
	}
	return false
}

// sessionGranted reports whether key was persistently granted to its session
// or to any ancestor of it.
func (s *permissionService) sessionGranted(key PermissionKey) bool {
	for _, id := range append([]string{key.SessionID}, s.ancestors(key.SessionID)...) {
		key.SessionID = id
		if _, ok := s.sessionPermissions.Get(key); ok {
			return true
		}
	}
	return false
}
