package permission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionTree(t *testing.T) {
	t.Parallel()

	t.Run("child inherits auto-approve", func(t *testing.T) {
		t.Parallel()
		s := NewPermissionService(t.TempDir(), false, nil).(*permissionService)
		s.AutoApproveSession("main")
		s.InheritSession("sub", "main")
		s.InheritSession("subsub", "sub")

		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		for _, id := range []string{"sub", "subsub"} {
			ok, err := s.Request(ctx, CreatePermissionRequest{SessionID: id, ToolName: "bash", Action: "execute", Path: t.TempDir()})
			require.NoError(t, err)
			require.True(t, ok, id)
		}
	})

	t.Run("unrelated session is not approved", func(t *testing.T) {
		t.Parallel()
		s := NewPermissionService(t.TempDir(), false, nil).(*permissionService)
		s.AutoApproveSession("main")
		s.InheritSession("sub", "other")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err := s.Request(ctx, CreatePermissionRequest{SessionID: "sub", ToolName: "bash", Action: "execute", Path: t.TempDir()})
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("persistent grants are shared across the tree", func(t *testing.T) {
		t.Parallel()
		s := NewPermissionService(t.TempDir(), false, nil).(*permissionService)
		s.InheritSession("sub", "main")
		s.InheritSession("sub2", "main")
		require.Equal(t, "main", s.rootSession("sub"))

		key := PermissionKey{SessionID: s.rootSession("sub"), ToolName: "bash", Action: "execute", Path: "/w"}
		s.sessionPermissions.Set(key, true)
		for _, id := range []string{"main", "sub", "sub2"} {
			key.SessionID = id
			require.True(t, s.sessionGranted(key), id)
		}
		key.SessionID = "unrelated"
		require.False(t, s.sessionGranted(key))
	})

	t.Run("cycles terminate", func(t *testing.T) {
		t.Parallel()
		s := NewPermissionService(t.TempDir(), false, nil).(*permissionService)
		s.InheritSession("a", "b")
		s.InheritSession("b", "a")
		require.Len(t, s.ancestors("a"), maxSessionDepth)
	})
}
