# reta

> Personal developer toolkit for scaffolding projects, managing secrets, and automating shared workflows.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go&logoColor=white)]()
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)]()
[![Test Suite](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml/badge.svg)](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml)

---

## Overview

`reta` is a single-binary developer toolkit that bundles the small, repetitive tasks every project needs: scaffolding a new repo with consistent structure, generating `.env` files with auto-filled secrets, sharing a common set of `make` targets across projects, scanning for secret leaks, and managing processes on local ports.

Compiled as a zero-dependency Go binary, `reta` runs natively across **Linux, macOS, and Windows** without requiring Python, `pip`, or Bash 5 on user machines.

## Features

- **Single cross-platform binary** — built in Go with embedded templates and `base.mk`; no Python or `bcrypt` runtime dependencies
- **Project scaffolding** — `reta init` copies templates into the current directory and writes a `.retasign` signature; existing `Makefile`s can be automatically converted (`--convert`) while backing up the original
- **Secret generation** — `reta init-env` reads `.env.example`, resolves `GENERATE_SECRET` / `AUTO_HASH` markers (built-in bcrypt), and prompts for missing keys
- **Env inspection & syncing** — `reta env check|list|diff|sync` validates `.env` against `.env.example`, flags missing/empty/duplicate keys, masks secret values, and safely appends missing keys
- **Secret & leak scanner** — `reta check-secrets` (and `reta hook install` git pre-commit hook) scans git status and source files for tracked `.env` files and hardcoded keys
- **AI Context export** — `reta export` (or `make export`) packages source files into markdown with estimated token counts and `.exportignore` support
- **Project search & jump** — `reta search` and the `rcd` shell shortcut discover `.retasign` projects and jump between them instantly
- **Port manager** — `reta port list|kill|docker` lists listening ports across Linux (`ss`/`lsof`), macOS (`lsof`), and Windows (`netstat`/`tasklist`) and terminates processes or Docker containers
- **System cleanup** — `reta clean` removes developer/OS caches with before/after disk and RAM summaries
- **Shell integration** — `reta completion` and `reta shell-init` generate native Bash, Zsh, and PowerShell support
- **Installation health check** — `reta doctor` verifies binary paths, `base.mk`, shell hooks, and tool dependencies

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

Requires Go 1.22+ (only needed for building from source):

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

### Command overview

```bash
reta help          # List all subcommands, grouped
reta --version     # Show current version
```

### Scaffold a new project

```bash
mkdir my-new-project && cd my-new-project
reta init          # Copies templates (or converts existing Makefile interactively)
reta init-env      # Generates .env from .env.example
reta hook install  # Optional: install git pre-commit secret scanner
make help          # Shared base.mk targets (Linux/macOS or Windows with GNU make)
```

To convert an existing project's `Makefile` without prompting:

```bash
reta init --convert
```

### Inspect and sync environment files

```bash
reta env check           # List keys missing or empty in .env
reta env list            # List keys with masked values
reta env list --show     # Reveal real values
reta env diff            # Compare keys between .env and .env.example
reta env sync            # Append missing keys from .env.example into .env
```

`reta env check` exits non-zero when keys are missing or empty, making it safe for CI pipelines.

### Search and jump across projects

```bash
rcd                # Interactive project selector (requires shell-init)
rcd myapp          # Jump directly to matching project
reta search        # List/select projects directly
```

### Manage running ports

```bash
reta port list           # List all listening ports (SYSTEM / DOCKER / USER)
reta port list 8000      # Show only port 8000
reta port kill           # Interactive selector
reta port kill 8000      # Stop process or Docker container on port 8000
reta port docker         # List running Docker containers
```

### Export AI context & scan secrets

```bash
reta export                    # Export source files to exports/project_export.md
reta export -r ./src --dir .   # Export ./src to ./project_export.md
reta check-secrets             # Scan repo for tracked .env files and leaked keys
```

### Configure

```bash
reta config                    # Show all settings
reta config hints off          # Hide "Next steps" hints
reta config hints on           # Show them again
reta config learn on           # Print underlying commands in make targets
reta config search-depth 5     # Max directory depth for `reta search` / `rcd`
reta config --reset            # Restore defaults
```

Temporary override for a single shell:

```bash
RETA_HINTS=off reta env check
```

### Update & Uninstall

```bash
reta update        # Pull latest changes (git install) or update release binary
reta uninstall     # Remove binary/symlinks and optionally ~/.reta
```

## Project Structure

```text
reta-toolkit/
├── assets.go               # Embedded VERSION, templates/, and lib/base.mk
├── cmd/reta/               # CLI entrypoint and command dispatcher
├── internal/
│   ├── adopt/              # Makefile converter
│   ├── clean/              # Cross-platform system cleanup
│   ├── config/             # Persistent settings manager
│   ├── doctor/             # Installation health check
│   ├── envfile/            # init-env and env check/list/diff/sync
│   ├── exporter/           # Source export for AI context
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
└── tests/                  # Automated test suite
```

## Testing

Run all Go unit tests and CLI integration tests locally:

```bash
./tests/run.sh
```

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contact

Open an issue on [GitHub](https://github.com/eraygdev/reta-toolkit/issues).