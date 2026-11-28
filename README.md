# reta

> Cross-platform developer toolkit for project scaffolding, secret hygiene, shared Make workflows, and local environment utilities.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)
[![Test Suite](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml/badge.svg)](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/eraygdev/reta-toolkit)](https://github.com/eraygdev/reta-toolkit/releases)

[![Total Downloads](https://img.shields.io/github/downloads/eraygdev/reta-toolkit/total?label=total%20downloads)](https://github.com/eraygdev/reta-toolkit/releases)
[![Linux Downloads](https://img.shields.io/github/downloads/eraygdev/reta-toolkit/latest/reta-linux-amd64?label=linux%20downloads)](https://github.com/eraygdev/reta-toolkit/releases)
[![macOS Downloads](https://img.shields.io/github/downloads/eraygdev/reta-toolkit/latest/reta-darwin-amd64?label=macos%20downloads)](https://github.com/eraygdev/reta-toolkit/releases)
[![Windows Downloads](https://img.shields.io/github/downloads/eraygdev/reta-toolkit/latest/reta-windows-amd64.exe?label=windows%20downloads)](https://github.com/eraygdev/reta-toolkit/releases)

---

## Overview

`reta` is a single-binary developer toolkit that consolidates the repetitive tasks every project needs: scaffolding consistent repo templates, generating `.env` files with auto-filled secrets, sharing common GNU `make` targets across projects via `base.mk`, scanning for secret leaks, managing local ports, cleaning system caches, and packaging source code for LLM context.

Compiled with `CGO_ENABLED=0` and embedded templates/Makefiles, `reta` runs natively across **Linux, macOS, and Windows** without external Python or shell-script runtime dependencies.

## Architecture

`cmd/reta` parses subcommands and routes execution to focused packages under `internal/`, while `assets.go` embeds `VERSION`, `templates/`, and `lib/base.mk` directly into the compiled binary so `init` and `doctor --fix` work anywhere.

```text
[reta CLI / rcd shell hook] ──▶ [cmd/reta dispatcher] ──▶ [internal/* packages]
                                        │
                                        ▼
                       [Embedded assets (VERSION, templates/, lib/base.mk)]
```

## Features

- **Zero-dependency cross-platform binary** — embeds templates and `lib/base.mk` directly into the Go binary.
- **Project scaffolding & Makefile adoption** — `reta init` scaffolds project templates and writes a `.retasign` project signature; `reta adopt-makefile` (or `reta init --convert`) migrates legacy `Makefile`s to `base.mk` with automatic backups.
- **Secret generation & env management** — `reta init-env` resolves `GENERATE_SECRET` (32-byte hex) and `AUTO_HASH` (bcrypt) markers; `reta env check|list|diff|sync` inspects, masks, diffs, and syncs keys against `.env.example`.
- **Leak & pre-commit protection** — `reta check-secrets` and `reta hook install` detect tracked `.env` files, private keys, and hardcoded API tokens in working trees or staged git changes.
- **AI context exporter** — `reta export` packages source code into markdown with `.exportignore` filtering and ChatGPT, Gemini, and Claude token estimates.
- **Project search & jump** — `reta search` and the `rcd` shell function discover `.retasign` projects recursively and jump between directories.
- **Port & Docker manager** — `reta port list|kill|docker` inspects listening sockets across Linux (`ss`/`lsof`), macOS (`lsof`), and Windows (`netstat`/`tasklist`) and stops processes or containers safely.
- **System cleanup & diagnostics** — `reta clean` reclaims OS/developer caches with before/after summaries; `reta doctor --fix` verifies paths, shell hooks, and repairs `~/.reta/lib/base.mk`.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Language | Go 1.26+ |
| Dependencies | `golang.org/x/crypto` (bcrypt), `golang.org/x/term`, `golang.org/x/sys` |
| Shared Task Runner | GNU Make (`lib/base.mk`) |
| Shell Integration | Bash, Zsh, PowerShell |
| Testing | Go `testing`, Bash test suite (`tests/run.sh`), PowerShell (`tests/test_windows.ps1`) |
| CI / Releases | GitHub Actions (`test.yml`, `release.yml`) |

## Quick Start

### Prerequisites

- **Prebuilt binary:** No runtime dependencies required for core CLI commands (`make` and `bash` are optional and only needed for `base.mk` targets; `git` is needed for `hook` and git-aware `check-secrets`).
- **Build from source:** Go 1.26+ and Git.

### Installation

**Linux & macOS (Prebuilt Binary):**

```bash
curl -fsSL https://raw.githubusercontent.com/eraygdev/reta-toolkit/main/scripts/install.sh | sh
```

**Windows PowerShell (Prebuilt Binary):**

```powershell
irm https://raw.githubusercontent.com/eraygdev/reta-toolkit/main/scripts/install.ps1 | iex
```

**Build from Source:**

```bash
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta
cd ~/.reta
./install.sh
```

Verify the installation:

```bash
reta doctor
```

## Configuration

Persistent settings are stored in `~/.config/reta/config` and managed via `reta config`.

| Setting / Variable | Description | Default |
| --- | --- | --- |
| `agents-file` | Agent prompt file created by `reta init` (`""` disables) | `AGENTS.md` |
| `agents-template` | Local file copied into `agents-file` during `reta init` | `""` (empty) |
| `hints` | Show or hide `Next steps` output (`on`/`off`) | `on` |
| `learn` | Print underlying commands inside `base.mk` targets (`on`/`off`) | `off` |
| `search-depth` | Max directory depth for `reta search` / `rcd` (`>= 1`) | `4` |
| `RETA_HINTS` | Environment override for `hints` (`on`/`off`/`1`/`0`) | unset |
| `RETA_LEARN` | Environment override for `learn` (`on`/`off`/`1`/`0`) | unset |
| `NO_COLOR` | Disable ANSI terminal colors when set | unset |

```bash
reta config                              # Show all settings
reta config agents-file CLAUDE.md        # Rename agent file created by `reta init`
reta config agents-template ~/prompt.md  # Auto-fill agent file from a local prompt
reta config hints off                    # Hide "Next steps" hints
reta config learn on                     # Print underlying commands in make targets
reta config search-depth 6               # Increase search depth for `reta search` / `rcd`
reta config --reset                      # Restore defaults
```

## Usage

### CLI

```bash
reta help          # List all subcommands, grouped
reta --version     # Show current version
```

#### Scaffold or Adopt a Project

```bash
mkdir my-new-project && cd my-new-project
reta init          # Copies templates and writes .retasign
reta init-env      # Generates .env from .env.example
reta hook install  # Optional: install git pre-commit secret scanner
make help          # List shared base.mk targets and project targets
```

Convert an existing `Makefile` to use `~/.reta/lib/base.mk`:

```bash
reta adopt-makefile             # Converts Makefile in-place (creates Makefile.bak)
reta init --convert             # Scaffolds missing templates and converts Makefile
```

#### Inspect and Sync Environment Files

```bash
reta env check           # List keys missing or empty in .env
reta env list            # List keys with masked values
reta env list --show     # Reveal real values
reta env diff            # Compare keys between .env and .env.example
reta env sync            # Append missing keys from .env.example into .env
```

#### Search and Jump Across Projects

```bash
rcd                # Interactive project selector (requires shell-init)
rcd myapp          # Jump directly to matching project
reta search        # List/select projects directly
```

#### Manage Running Ports & Containers

```bash
reta port list           # List all listening ports (SYSTEM / DOCKER / USER)
reta port list 8000      # Show only port 8000
reta port kill           # Interactive selector
reta port kill 8000      # Stop process or Docker container on port 8000
reta port docker         # List running Docker containers with uptime and ports
```

#### Export AI Context, Scan Secrets & System Cleanup

```bash
reta export                    # Export source files to exports/project_export.md
reta export -r ./src --dir .   # Export ./src to ./project_export.md
reta export --clean-exports    # Clean all files inside exports/ with confirmation
reta check-secrets             # Scan repo for tracked .env files and leaked keys
reta check-secrets --staged    # Scan only git staged changes
reta clean                     # Safe OS/developer cache cleanup with disk/RAM summary
```

#### Update, Repair & Uninstall

```bash
reta doctor --fix  # Verify installation and repair ~/.reta/lib/base.mk
reta update        # Pull latest changes (git install) or update release binary
reta uninstall     # Remove binary/symlinks and optionally ~/.reta
```

## Project Structure

```text
reta-toolkit/
├── assets.go               # Embedded VERSION, templates/, and lib/base.mk
├── cmd/reta/               # CLI entrypoint and command registry
├── internal/
│   ├── adopt/              # Makefile parser and converter (dynamic target matrix)
│   ├── clean/              # Cross-platform system cleanup
│   ├── cli/                # Flag parser and exit-code error helpers
│   ├── config/             # Persistent settings manager
│   ├── doctor/             # Installation health check and auto-fix
│   ├── envfile/            # init-env and env check/list/diff/sync
│   ├── exporter/           # AI context markdown exporter and token estimator
│   ├── hook/               # Git pre-commit hook manager
│   ├── lifecycle/          # Self-update and uninstall workflows
│   ├── port/               # Cross-platform port & Docker manager
│   ├── projectinit/        # Project scaffolding (`reta init`)
│   ├── retasign/           # `.retasign` project signature helpers
│   ├── search/             # Project discovery and jump cache
│   ├── secrets/            # Hardcoded secret and `.env` scanner
│   ├── shell/              # Bash, Zsh, and PowerShell completion & `rcd`
│   ├── testutil/           # Shared stdio capture helpers for tests
│   └── ui/                 # Terminal palette, prompts, and hints
├── lib/base.mk             # Shared GNU Make targets
├── scripts/                # Prebuilt release installers (install.sh, install.ps1)
├── templates/              # Embedded project templates
├── tests/                  # Automated integration tests (Bash & PowerShell)
├── go.mod                  # Go module definition
├── install.sh              # Source build installer
├── Makefile                # Project Makefile using base.mk
└── VERSION                 # Semantic version string
```

## Development

Build and reinstall locally after modifying source files or embedded `templates/`:

```bash
./install.sh
```

Or build the binary directly:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o reta ./cmd/reta
```

## Testing

Run Go unit tests and Unix CLI integration suites:

```bash
go test -v ./...
./tests/run.sh
```

Run the Windows PowerShell runtime suite:

```powershell
go build -o reta.exe ./cmd/reta
./tests/test_windows.ps1
```

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contact

Open an issue on [GitHub](https://github.com/eraygdev/reta-toolkit/issues).