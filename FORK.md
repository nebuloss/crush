# nebuloss/crush

A fork of [charmbracelet/crush](https://github.com/charmbracelet/crush) that
carries a few changes on top of upstream releases, each kept as its own commit
so it can be offered upstream as a PR.

## Branches

| branch | what it is |
| --- | --- |
| `main` | Untouched mirror of upstream `main`. Sync it with GitHub's "Sync fork". |
| `patched` | Latest upstream **release** tag + the commits below. Build from here. |

Releases are tagged `v<upstream>-nebuloss.<n>` (e.g. `v0.97.1-nebuloss.1`);
pushing such a tag publishes standalone linux/amd64 and windows/amd64
binaries (`crush_<version>_linux_x86_64`, `..._windows_x86_64.exe`).

## Changes

### `permissions.allowed_commands`

Removes named commands from the bash tool's hardcoded blocklist (`ssh`,
`curl`, `sudo`, package managers, …), which `--yolo` and `allowed_tools` do not
affect. Upstream: [#515](https://github.com/charmbracelet/crush/issues/515)
(closed), [#2831](https://github.com/charmbracelet/crush/pull/2831) (same
config shape, unmerged).

```json
{ "permissions": { "allowed_commands": ["ssh", "scp", "curl"] } }
```

crushrc: `permissions allow-command ssh scp curl`.

- Matches `argv[0]` exactly; allowing a command also lifts its subcommand
  rules (`apt` re-enables `apt install`).
- Un-banning is not permission to run: the normal prompt still applies unless
  `bash` is in `allowed_tools`.
- The bash tool description advertises the effective blocklist, and the coder
  prompt's "never use `curl` through bash" line is dropped when `curl` is
  allowed — otherwise the model refuses before calling the tool.

### `options.task_agent_tools`

Sub-agents (the `agent` tool) are hardcoded upstream to a read-only tool set
with no bash, and `Config.Agents` is not configurable.

```json
{ "options": { "task_agent_tools": ["bash", "job_output", "job_kill"] } }
```

crushrc: `option task-agent-tool bash` (appends).

- Tools come from the coder's resolved set, so `disabled_tools` still wins;
  `agent` is never granted.
- The `agent` tool description lists the sub-agent's real tools.
- Sub-agent sessions inherit their parent's approvals. Without this a
  sub-agent's first bash call blocks forever under `crush run`, and in the TUI
  "allow for session" covers only the one sub-agent that asked.

### `SideQuestion` (engine for `/btw`)

Answers one question from a session's context with no tools, one turn, and
nothing written back, while the main agent keeps working. Engine only; no UI
yet.

### CI

Upstream's workflows rely on Charm's secrets, runners and goreleaser-pro, so
they are removed here. `fork-ci.yml` cross-compiles linux/amd64 and
windows/amd64 on one Ubuntu runner (`scripts/fork-dist.sh`) and keeps the
binaries as artifacts; `fork-release.yml` does the same on a `v*-*` tag and
publishes a GitHub release. Tests are not run in CI; run `go test ./...` on
the build machine after a rebase.

## Updating to a new upstream release

```sh
scripts/fork-rebase.sh            # onto the latest upstream release
go test ./...                     # (on the build machine)
git push --force-with-lease origin patched
git tag v0.98.0-nebuloss.1 && git push origin v0.98.0-nebuloss.1
```

The script keeps the fork's deletion of upstream workflow files when upstream
has edited them, and stops on any other conflict.
