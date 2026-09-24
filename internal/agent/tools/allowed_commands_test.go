package tools

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/shell"
)

// blocked reports whether the bash tool's deny rules reject argv, given the
// supplied permissions.allowed_commands.
func blocked(allowed []string, argv ...string) bool {
	return slices.ContainsFunc(blockFuncs(allowed), func(f shell.BlockFunc) bool {
		return f(argv)
	})
}

func TestBlockFuncsWithoutAllowlistMatchesUpstream(t *testing.T) {
	for _, argv := range [][]string{
		{"ssh", "host", "uptime"},
		{"scp", "a", "host:b"},
		{"curl", "https://example.com"},
		{"sudo", "reboot"},
		{"apt", "install", "cowsay"},
		{"go", "install", "example.com/x@latest"},
		{"go", "test", "-exec", "sudo", "./..."},
		{"npm", "install", "-g", "left-pad"},
	} {
		if !blocked(nil, argv...) {
			t.Errorf("expected %v to be blocked with no allowlist", argv)
		}
	}

	for _, argv := range [][]string{
		{"echo", "hi"},
		{"git", "status"},
		{"go", "test", "./..."},
		{"npm", "install"},
	} {
		if blocked(nil, argv...) {
			t.Errorf("expected %v to be allowed with no allowlist", argv)
		}
	}
}

func TestAllowedCommandsUnblocksNamedCommand(t *testing.T) {
	allowed := []string{"ssh", "scp"}

	for _, argv := range [][]string{
		{"ssh", "dev-build", "make"},
		{"scp", "file", "dev-build:/tmp/"},
	} {
		if blocked(allowed, argv...) {
			t.Errorf("expected %v to be allowed", argv)
		}
	}

	// Everything else stays blocked.
	for _, argv := range [][]string{
		{"curl", "https://example.com"},
		{"sudo", "reboot"},
	} {
		if !blocked(allowed, argv...) {
			t.Errorf("expected %v to remain blocked", argv)
		}
	}
}

func TestAllowedCommandsLiftsSubcommandRules(t *testing.T) {
	// Allowing a command lifts its argument rules too, so the command is
	// not left in a half-allowed state.
	if blocked([]string{"apt"}, "apt", "install", "cowsay") {
		t.Error("expected `apt install` to be allowed when apt is allowed")
	}
	if blocked([]string{"go"}, "go", "install", "example.com/x@latest") {
		t.Error("expected `go install` to be allowed when go is allowed")
	}
	// A sibling rule on a different command is untouched.
	if !blocked([]string{"apt"}, "yum", "install", "cowsay") {
		t.Error("expected `yum install` to stay blocked")
	}
}

func TestAllowedCommandsIgnoresBlankEntries(t *testing.T) {
	// A stray empty entry must not match an empty argv or disable the rules.
	allowed := []string{"", "   "}
	if !blocked(allowed, "ssh", "host") {
		t.Error("blank allowlist entries must not unblock anything")
	}
	if blocked(allowed) {
		t.Error("empty argv must not be reported as blocked")
	}
}

func TestAllowedCommandsMatchesNameNotPath(t *testing.T) {
	// The allowlist keys on argv[0] exactly, mirroring shell.CommandsBlocker.
	if blocked([]string{"ssh"}, "/usr/bin/ssh", "host") {
		t.Error("absolute paths were never blocked to begin with")
	}
	if !blocked([]string{"/usr/bin/ssh"}, "ssh", "host") {
		t.Error("allowing a path must not unblock the bare command name")
	}
}

func TestEffectiveBannedCommands(t *testing.T) {
	banned := []string{"curl", "ssh", "sudo"}

	got := effectiveBannedCommands(banned, []string{"ssh"})
	if slices.Contains(got, "ssh") {
		t.Errorf("ssh should be filtered out, got %v", got)
	}
	if !slices.Contains(got, "curl") || !slices.Contains(got, "sudo") {
		t.Errorf("unrelated commands should survive, got %v", got)
	}

	// The input slice must not be mutated: bannedCommands is package state
	// shared by every agent in the process.
	if !slices.Equal(banned, []string{"curl", "ssh", "sudo"}) {
		t.Errorf("input slice was mutated: %v", banned)
	}

	if got := effectiveBannedCommands(banned, nil); !slices.Equal(got, banned) {
		t.Errorf("empty allowlist should return the input unchanged, got %v", got)
	}
}

func TestBashDescriptionReflectsAllowlist(t *testing.T) {
	attribution := &config.Attribution{TrailerStyle: config.TrailerStyleNone}

	if desc := bashDescription(attribution, "m", nil); !strings.Contains(desc, ", ssh,") {
		t.Error("expected ssh to be listed as banned by default")
	}
	if desc := bashDescription(attribution, "m", []string{"ssh"}); strings.Contains(desc, ", ssh,") {
		t.Error("expected ssh to be absent from the banned list when allowed")
	}
}
