# `dblab teleport setup` CLI command

## Overview

Add a `dblab teleport setup` command that automates the Teleport integration
setup described in `engine/cmd/cli/commands/teleport/SETUP.md`. Currently, users must
manually create Teleport roles, generate bot identities, create SSL certificates,
export CA certs, write pg_hba.conf, and configure the Teleport DB agent — a multi-step
process that is error-prone and time-consuming.

The command runs 8 idempotent steps in sequence, skipping any that are already done.
It is flag-driven (no interactive prompts), consistent with the existing CLI pattern.
For server.yml changes, the command prints YAML snippets for the user to apply manually
rather than editing the file directly (to avoid YAML anchor/comment corruption).

**Acceptance criteria:** Running `dblab teleport setup` on a fresh host with Teleport
access produces all required files and Teleport resources. After following the printed
next-steps (DLE restart, data refresh), the Teleport integration is fully operational.

## Context (from discovery)

- files/components involved:
  - `engine/cmd/cli/commands/teleport/serve.go` — existing command pattern to follow
  - `engine/cmd/cli/commands/teleport/tctl.go` — existing tctl execution helpers
  - `engine/cmd/cli/commands/teleport/SETUP.md` — reference for all setup steps
  - `engine/cmd/cli/main.go:43` — where `teleport.CommandList()` is registered
- related patterns found:
  - CLI uses `urfave/cli/v2`, all config via flags/env vars, no interactive prompts
  - `tctl` called via `exec.CommandContext()` with identity + auth-server args
  - existing `runTctl()` helper in `tctl.go` requires identity file (not usable for pre-identity steps)
- dependencies identified:
  - external binaries: `tctl`, `tbot`, `openssl`
  - DBLab API client for edition check

## Development Approach

- **testing approach**: Regular (code first, then tests)
- complete each task fully before moving to the next
- make small, focused changes
- use a `cmdRunner` interface for external command execution to enable mocking in tests
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- run tests after each change
- maintain backward compatibility

## Testing Strategy

- **unit tests**: required for every task
  - use `cmdRunner` interface to mock all external command calls (tctl/tbot/openssl)
  - test idempotency checks (skip-if-done logic)
  - test pg_hba.conf and teleport.yaml content generation
  - test flag validation and defaults
  - test token parsing with real output samples from tctl/tbot
- no e2e tests (CLI command, tested via unit tests + manual verification on demo server)

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix
- update plan if implementation deviates from original scope

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): code changes, tests, documentation
- **Post-Completion** (no checkboxes): manual testing on demo server, SETUP.md updates

## Implementation Steps

### Task 1: Command registration, config, and command runner interface

**Files:**
- Create: `engine/cmd/cli/commands/teleport/setup.go`

- [ ] add `setup` subcommand to the existing `teleport` parent command in `CommandList()`
- [ ] define all flags with defaults: `--teleport-proxy` (required), `--cert-dir` (default: `/var/lib/dblab/cert`), `--pg-hba-path` (default: `/var/lib/dblab/pg_hba.conf`), `--teleport-yaml` (default: `/etc/teleport.yaml`), `--identity-dir` (default: `/etc/teleport/bot-dest`), `--environment-id` (required), `--webhook-listen-addr` (default: `172.17.0.1:9876`), `--dblab-url` (default: `http://localhost:2345`), `--dblab-token` (env: `DBLAB_TOKEN`), `--webhook-secret` (env: `WEBHOOK_SECRET`)
- [ ] define `SetupConfig` struct and `setupAction` that parses flags, validates required fields
- [ ] define `cmdRunner` interface (`Run(ctx, name, args) ([]byte, error)`) and real implementation using `exec.CommandContext`
- [ ] implement `checkTools()` to verify `tctl`, `tbot`, `openssl` are in PATH
- [ ] write tests for flag validation and `checkTools()` using mock `cmdRunner`
- [ ] run tests — must pass before task 2

### Task 2: Teleport role creation (steps 1-2)

**Files:**
- Create: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `ensureRole(ctx, runner, name, yamlContent)` — runs `tctl get role/<name>` (without --identity, uses user's tctl credentials from tsh login), creates via `tctl create -f -` if not found
- [ ] define `dblabBotRoleYAML` constant (db/db_server/app/app_server permissions)
- [ ] define `dblabUserRoleYAML` constant (db_labels: dblab=true)
- [ ] wire steps 1-2 into `setupAction` with `[1/8]` / `[2/8]` log output
- [ ] write tests for `ensureRole` — role exists (skip), role missing (create), tctl error
- [ ] run tests — must pass before task 3

### Task 3: Bot identity generation (step 3)

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `ensureBotIdentity(ctx, runner, cfg)` — check if identity file exists in `--identity-dir`, skip if present
- [ ] check if bot already exists with `tctl bots ls`, handle "bot exists but identity missing" case
- [ ] run `tctl bots add dblab-sidecar --roles=dblab-bot --format=json` (use --format=json for stable token parsing; fall back to regex if --format not supported)
- [ ] run `tbot start --oneshot` with the parsed token
- [ ] wire step 3 into `setupAction` with `[3/8]` log output
- [ ] write tests: identity exists (skip), bot missing (create + tbot), bot exists but identity missing (recreate identity), token parsing from JSON and fallback regex
- [ ] run tests — must pass before task 4

### Task 4: SSL certificate and Teleport CA generation (steps 4-5)

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `ensureSSLCerts(ctx, runner, certDir)` — check if `server.crt` + `server.key` exist, skip if present; mkdir certDir if needed
- [ ] run `openssl req -new -x509 ...` then `chown 999:999`; if chown fails, log warning instead of failing (user may not be root)
- [ ] implement `ensureTeleportCA(ctx, runner, certDir)` — check if `teleport-ca.crt` exists, skip if present
- [ ] run `tctl auth export --type=db-client` then `chown 999:999` (same warning approach)
- [ ] wire steps 4-5 into `setupAction` with `[4/8]` / `[5/8]` log output
- [ ] write tests for cert existence checks, command construction, chown warning on failure
- [ ] run tests — must pass before task 5

### Task 5: pg_hba.conf generation (step 6)

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `ensurePgHba(path)` — check if file exists, skip if present
- [ ] write pg_hba.conf with comment header and 3 rules: `local trust`, `hostssl cert`, `host md5`
- [ ] wire step 6 into `setupAction` with `[6/8]` log output
- [ ] write tests for content generation and skip-if-exists
- [ ] run tests — must pass before task 6

### Task 6: Print server.yml configuration snippets (step 7)

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `printServerYmlSnippets(cfg)` — generates and prints YAML blocks for the user to add to server.yml
- [ ] print `databaseConfigs` snippet with ssl, ssl_cert_file, ssl_key_file, ssl_ca_file (using paths from cfg.CertDir)
- [ ] print `databaseContainer.containerConfig.volume` snippet with cert mount path
- [ ] print `webhooks` snippet with URL (using cfg.WebhookAddr), secret, and triggers
- [ ] wire step 7 into `setupAction` with `[7/8]` log output
- [ ] write tests for snippet content (correct paths, correct webhook URL)
- [ ] run tests — must pass before task 7

### Task 7: teleport.yaml generation (step 8)

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup_steps.go`

- [ ] implement `ensureTeleportYaml(ctx, runner, path, cfg)` — check if file exists, skip if present
- [ ] create join token via `tctl tokens add --type=db --ttl=8760h --format=json` (fall back to regex parsing)
- [ ] write teleport.yaml with proxy_server, join_params, db_service with `dblab: "true"` label matcher
- [ ] log a note about token expiration (1 year TTL) in the output
- [ ] wire step 8 into `setupAction` with `[8/8]` log output
- [ ] write tests for teleport.yaml content generation and token parsing
- [ ] run tests — must pass before task 8

### Task 8: Post-setup summary and next-steps output

**Files:**
- Modify: `engine/cmd/cli/commands/teleport/setup.go`

- [ ] implement `printNextSteps(cfg)` — print summary of what was created/skipped
- [ ] print concrete next-steps commands: (1) add server.yml snippets from step 7, (2) restart DLE with `-v pg_hba.conf:/home/dblab/standard/...`, (3) trigger data refresh, (4) start teleport agent, (5) start sidecar with exact flags
- [ ] wire summary into `setupAction` after all steps complete
- [ ] write test for next-steps output containing correct paths from config
- [ ] run tests — must pass before task 9

### Task 9: Verify acceptance criteria

- [ ] verify all 8 steps are implemented and idempotent
- [ ] verify external tool checks fail fast with clear messages
- [ ] verify flag defaults are sensible
- [ ] run full test suite: `make test`
- [ ] run linter: `make run-lint`
- [ ] run formatter: `make fmt`

### Task 10: [Final] Update documentation

- [ ] update SETUP.md to add "Quick Start" section at the top referencing `dblab teleport setup`
- [ ] keep existing manual steps as reference for users who prefer step-by-step
- [ ] move this plan to `docs/plans/completed/`

## Technical Details

### SetupConfig struct

```go
type SetupConfig struct {
    TeleportProxy string
    CertDir       string
    PgHbaPath     string
    TeleportYaml  string
    IdentityDir   string
    EnvironmentID string
    WebhookAddr   string
    DblabURL      string
    DblabToken    string
    WebhookSecret string
}
```

### cmdRunner interface (for testability)

```go
type cmdRunner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
    RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error)
}
```

Real implementation wraps `exec.CommandContext`. Tests provide a mock that records
calls and returns configured output.

### tctl authentication model

Steps 1-3 use `tctl` with the **user's own credentials** (from `tsh login`).
This means `tctl` is called WITHOUT `--identity` or `--auth-server` flags —
it uses the default Teleport profile from `~/.tsh/`.

Steps 5 (Teleport CA export) and 7 (token creation) also use user credentials.

The `--identity` flag from serve.go's `runTctl()` is NOT reused for setup,
because the bot identity does not exist yet during setup.

### Step execution flow

```
setupAction(c *cli.Context)
├── parse flags → SetupConfig
├── checkTools(tctl, tbot, openssl)
├── [1/8] ensureRole("dblab-bot", dblabBotRoleYAML)
├── [2/8] ensureRole("dblab-user", dblabUserRoleYAML)
├── [3/8] ensureBotIdentity(cfg)
├── [4/8] ensureSSLCerts(cfg.CertDir)
├── [5/8] ensureTeleportCA(cfg.CertDir)
├── [6/8] ensurePgHba(cfg.PgHbaPath)
├── [7/8] printServerYmlSnippets(cfg)    // prints, does not edit
├── [8/8] ensureTeleportYaml(cfg)
└── printNextSteps(cfg)
```

### File layout

- `setup.go` — command registration, flags, SetupConfig, setupAction, cmdRunner interface, checkTools
- `setup_steps.go` — step implementations (ensureRole, ensureBotIdentity, ensureSSLCerts, etc.)
- `setup_test.go` — all tests with mock cmdRunner

### Token parsing strategy

Use `--format=json` flag for `tctl bots add` and `tctl tokens add` when available.
If the command fails with `--format=json` (older Teleport versions), fall back to
regex parsing of human-readable output:
- Bot token: `The bot token: ([a-f0-9]+)`
- Join token: `The invite token: ([a-f0-9]+)`

### Role YAML templates

Embedded as Go string constants, matching the YAML from SETUP.md §1-§2.

## Post-Completion

**Manual verification:**
- test full setup flow on a test server with a Teleport cluster
- test idempotency by running the command twice
- test with missing external tools
- test with partially completed setup (some steps done, some not)
- test as non-root user (verify chown warnings)

**SETUP.md update:**
- add "Quick Start" section at the top referencing `dblab teleport setup`
- keep manual steps as reference for users who prefer step-by-step
