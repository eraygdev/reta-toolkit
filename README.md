# reta

> Personal developer toolkit for scaffolding projects, managing secrets, automating shared workflows, and preparing AI context.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go&logoColor=white)]()
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)]()
[![Test Suite](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml/badge.svg)](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml)

---

## Overview

`reta` is a single-binary developer toolkit that bundles the small, repetitive tasks every project needs: scaffolding a new repo with consistent structure, generating `.env` files with auto-filled secrets, sharing a common set of `make` targets across projects via `base.mk`, scanning for secret leaks, managing local ports, cleaning system caches, and packaging source code for LLM context.

Compiled as a zero-dependency Go binary, `reta` runs natively across **Linux, macOS, and Windows** (fully tested and verified on all platforms).

## Features

- **Single cross-platform binary** — built in Go with embedded templates and `base.mk`; no Python or `bcrypt` runtime dependencies.
- **Project scaffolding** — `reta init` copies templates into the current directory and writes a `.retasign` project signature. Existing legacy `Makefile`s can be automatically converted (`reta adopt-makefile` or `reta init --convert`) with automated backups.
- **Secret generation** — `reta init-env` reads `.env.example`, resolves `GENERATE_SECRET` (32-byte hex) and `AUTO_HASH` (bcrypt) markers, and prompts interactively for missing keys.
- **Env inspection & syncing** — `reta env check|list|diff|sync` validates `.env` against `.env.example`, flags missing/empty/duplicate keys, masks sensitive values, and safely appends missing keys.
- **Secret & leak scanner** — `reta check-secrets` (and `reta hook install` git pre-commit hook) scans git status, staged changes, and source files for tracked `.env` files and hardcoded API keys/secrets.
- **AI Context export** — `reta export` (or `make export`) packages source files into a single markdown file with token estimations (ChatGPT, Claude, Gemini) and `.exportignore` support.
- **Project search & jump** — `reta search` and the `rcd` shell shortcut discover `.retasign` projects recursively and jump between them instantly.
- **Port manager** — `reta port list|kill|docker` lists listening ports across Linux (`ss`/`lsof`), macOS (`lsof`), and Windows (`netstat`/`tasklist`) and terminates processes or Docker containers safely.
- **System cleanup** — `reta clean` removes developer/OS caches with before/after disk and RAM summaries.
- **Shell integration** — `reta completion` and `reta shell-init` generate native Bash, Zsh, and PowerShell support.
- **Installation health check** — `reta doctor --fix` verifies binary paths, `base.mk`, shell hooks, tool dependencies, and repairs links automatically.

## Quick Start

### Installation (Prebuilt Binary)

**Linux & macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/eraygdev/reta-toolkit/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/eraygdev/reta-toolkit/main/scripts/install.ps1 | iex
```

### Build from Source

Requires Go 1.26+ (only needed for building from source):

```bash
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta
cd ~/.reta
./install.sh
```

Verify the installation:

```bash
reta doctor
```

## Usage

### Command Overview

```bash
reta help          # List all subcommands, grouped
reta --version     # Show current version
```

### Scaffold or Adopt a Project

```bash
mkdir my-new-project && cd my-new-project
reta init          # Copies templates and writes .retasign
reta init-env      # Generates .env from .env.example
reta hook install  # Optional: install git pre-commit secret scanner
make help          # List shared base.mk targets and project targets
```

To convert an existing project's legacy `Makefile` to use `base.mk`:

```bash
reta adopt-makefile             # Converts Makefile in-place (creates Makefile.bak)
reta init --convert             # Scaffolds missing templates and converts Makefile interactively
```

### Inspect and Sync Environment Files

```bash
reta env check           # List keys missing or empty in .env (CI-friendly exit codes)
reta env list            # List keys with masked values
reta env list --show     # Reveal real values
reta env diff            # Compare keys between .env and .env.example
reta env sync            # Append missing keys from .env.example into .env
```

### Search and Jump Across Projects

```bash
rcd                # Interactive project selector (requires shell-init)
rcd myapp          # Jump directly to matching project
reta search        # List/select projects directly
```

### Manage Running Ports & Containers

```bash
reta port list           # List all listening ports (SYSTEM / DOCKER / USER)
reta port list 8000      # Show only port 8000
reta port kill           # Interactive selector
reta port kill 8000      # Stop process or Docker container on port 8000
reta port docker         # List running Docker containers with uptime and ports
```

### Export AI Context & Scan Secrets

```bash
reta export                    # Export source files to exports/project_export.md
reta export -r ./src --dir .   # Export ./src to ./project_export.md
reta export --clean-exports    # Clean all files inside exports/ with prompt
reta check-secrets             # Scan repo for tracked .env files and leaked keys
reta check-secrets --staged    # Scan only git staged changes (used by pre-commit hook)
```

### Configure Settings

```bash
reta config                    # Show all settings
reta config hints off          # Hide "Next steps" hints
reta config learn on           # Print underlying commands in make targets
reta config search-depth 6     # Max directory depth for `reta search` / `rcd`
reta config --reset            # Restore defaults
```

### Update & Uninstall

```bash
reta update        # Pull latest changes (git install) or update release binary
reta uninstall     # Remove binary/symlinks and optionally ~/.reta
```

## Project Structure

```text
reta-toolkit/
├── assets.go               # Embedded VERSION, templates/, and base.mk
├── cmd/reta/               # CLI entrypoint and command dispatcher
├── internal/
│   ├── adopt/              # Makefile converter & parser
│   ├── clean/              # Cross-platform system cleanup
│   ├── config/             # Persistent settings manager
│   ├── doctor/             # Installation health check & auto-fix
│   ├── envfile/            # init-env and env check/list/diff/sync
│   ├── exporter/           # Source export for AI context & token estimation
│   ├── hook/               # Git pre-commit hook manager
│   ├── lifecycle/          # update and uninstall
│   ├── port/               # Cross-platform port & Docker manager
│   ├── projectinit/        # Project scaffolding (reta init)
│   ├── retasign/           # .retasign project signature helpers
│   ├── search/             # Project discovery and jump cache
│   ├── secrets/            # Hardcoded secret and .env scanner
│   ├── shell/              # Bash, Zsh, and PowerShell completion & rcd
│   └── ui/                 # Shared terminal colors and Next steps
├── lib/base.mk             # Shared GNU Make targets
├── scripts/                # Binary release installers (install.sh, install.ps1)
├── templates/              # Embedded project templates
└── tests/                  # Automated test suite (Bash & PowerShell)
```

## Testing

Run all Go unit tests and CLI integration suites locally:

```bash
./tests/run.sh
```

For Windows PowerShell runtime suite:

```powershell
./tests/test_windows.ps1
```

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contact

Open an issue on [GitHub](https://github.com/eraygdev/reta-toolkit/issues).