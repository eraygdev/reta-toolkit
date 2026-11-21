# reta

> Personal developer toolkit for scaffolding projects, managing secrets, and automating shared workflows.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Bash](https://img.shields.io/badge/bash-5.0+-4EAA25?logo=gnu-bash&logoColor=white)]()
[![Python](https://img.shields.io/badge/python-3.10+-3776AB?logo=python&logoColor=white)]()
[![Status](https://img.shields.io/badge/status-beta-yellow)]()
[![Test Suite](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml/badge.svg)](https://github.com/eraygdev/reta-toolkit/actions/workflows/test.yml)

---

## Overview

`reta` is a single-command toolkit that bundles the small, repetitive tasks every project needs: scaffolding a new repo with consistent structure, generating `.env` files with auto-filled secrets, sharing a common set of `make` targets across projects, and managing processes on local ports.

Unlike full dotfiles managers (chezmoi, yadm), `reta` focuses on **project-level tooling** — what happens inside each repo — rather than managing `$HOME` itself. Every project includes the same `~/.reta/lib/base.mk`, so improvements to the shared toolkit propagate everywhere without touching individual projects.

## Features

- **Subcommand dispatcher** — a single `reta` command routes to `libexec/reta-*` scripts; new subcommands appear automatically in `reta help`
- **Project scaffolding** — `reta init` copies templates into the current directory; if an existing `Makefile` is found, it can automatically convert and adopt it (`--convert` or interactive prompt) while backing up the original
- **Secret generation** — `reta init-env` reads `.env.example`, replaces `GENERATE_SECRET` / `AUTO_HASH` markers with real values, and prompts for missing keys; `AUTO_HASH` requires the Python `bcrypt` package
- **Env inspection & syncing** — `reta env check|list|diff|sync` validates `.env` against `.env.example`, flags missing/empty/duplicate keys, masks secret values, and safely appends missing keys without touching existing ones
- **Shared make targets** — `lib/base.mk` is `include`d by every project, giving them `up`, `down`, `logs`, `clean`, `git-sync`, `export`, `clean-exports`, and `check-secrets`
- **Secret & leak scanner** — `make check-secrets` scans git status and source code for tracked `.env` files, missing `.gitignore` rules, and hardcoded API keys/private keys
- **AI Context export** — `make export` packages source files into markdown with estimated token counts; supports custom source paths (`make export SRC=./src`) and dynamic output directories (`DIR=...`)
- **Installation health check** — `reta doctor` verifies symlinks, file permissions, `PATH`, templates, completion links, and tool dependencies
- **Port manager** — `reta port list` shows listening ports; `reta port kill` terminates a process by port or via an interactive numbered menu, using SIGTERM then SIGKILL
- **Contextual hints** — setup commands finish with "Next steps" suggestions where applicable; can be disabled globally via `reta config hints off`
- **Clean uninstall** — `reta uninstall` removes only its own files; refuses to touch anything that doesn't look like a `reta` install

## Tech Stack

| Layer | Technology |
|---|---|
| Core scripts | Bash 5 |
| Utility scripts | Python 3.10+ |
| Task runner | GNU Make |
| Optional | Docker (for `make up/down/logs`) |

## Quick Start

### Prerequisites

- Bash 5.0+
- Python 3.10+
- GNU Make
- Git
- `ss` (Linux) or `lsof` (macOS/Linux) — for `reta port`
- Python `bcrypt` package — required by `reta init-env` when `.env.example` contains `AUTO_HASH` (`pip install bcrypt`)
- Docker (optional, only if projects use `make up`)

### Installation

**Quick install (single command):**

```
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && cd ~/.reta && ./install.sh
```

**Or step by step:**

```
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta
cd ~/.reta
./install.sh
```

> **Important:** The `~/.reta` target is required. `install.sh` installs the
> toolkit by symlinking `~/.local/bin/reta` to it — if you clone somewhere else,
> `reta` will break whenever that source folder moves or is deleted.

> **Important (Team / Collaborative Work):** 
> `reta` is designed for project-level tooling via shared `make` targets (`lib/base.mk`). 
> For team workflows, **every developer on the project must have `reta` installed at `~/.local/bin/reta` and `~/.reta`**, because the project's `Makefile` relies on `include $(HOME)/.reta/lib/base.mk`. Make sure all team members run `install.sh` before working with shared make targets.

If `install.sh` reports that `~/.local/bin` is not on `PATH`, add it to your shell config:

```
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Verify the installation:

```
reta doctor
```

## Usage

### Command overview

```
reta help          # List all subcommands, grouped
reta --version     # Show current version
```

### Scaffold a new project

```
mkdir my-new-project && cd my-new-project
reta init          # Copies templates (or converts existing Makefile interactively)
reta init-env      # Generates .env from .env.example
make help          # Base targets now available
```

To convert an existing project's `Makefile` without prompting:
```
reta init --convert
```

### Inspect and sync environment files

```
reta env check           # List keys missing or empty in .env
reta env list            # List keys with masked values
reta env list --show     # Reveal real values
reta env diff            # Compare keys between .env and .env.example
reta env sync            # Append missing keys from .env.example into .env
```

`reta env check` exits non-zero when keys are missing or empty, so it is safe
to use in CI pipelines. `reta env` also warns about duplicate keys — a common
source of silent misbehavior across `docker --env-file`, `python-dotenv`, and
`source .env`.

### Manage running ports

```
reta port list           # list all listening ports
reta port list 8000      # show only port 8000
reta port kill           # interactive selector (kill by PID)
reta port kill 8000      # kill whatever listens on port 8000
```

`reta port` uses `ss` when available, falling back to `lsof`.

### Configure

```
reta config                # Show all settings
reta config hints off      # Hide "Next steps" hints
reta config hints on       # Show them again
reta config --reset        # Restore defaults
```

Temporary override for a single shell:

```
RETA_HINTS=off reta env check
```

Settings live in `~/.config/reta/config` and survive `reta uninstall`, so a
fresh reinstall picks up where you left off.

### Update the toolkit

```
reta update        # git pull --rebase + rerun install.sh
```

`reta update` shows the version delta, a `git diff --stat`
summary, and a clear error message if the pull fails (uncommitted changes or
real merge conflicts).

### Uninstall

```
reta uninstall --keep    # remove PATH symlink only; keep ~/.reta
reta uninstall           # remove symlink + ~/.reta (with confirmation)
```

`reta uninstall` refuses to delete anything that doesn't look like a `reta`
installation (verifies `bin/reta` and `libexec/` exist) and never touches your
`~/.bashrc` or `~/.zshrc`.

## Project Structure

```
reta-toolkit/
├── bin/reta                # Dispatcher — the only command on PATH
├── lib/
│   ├── base.mk             # Shared make targets (included by every project)
│   ├── export-context      # Source export for AI context
│   ├── check-secrets       # Secret and leak scanner
│   ├── adopt-makefile      # Makefile converter
│   ├── ui.sh               # Shared bash colors and helpers
│   └── ui.py               # Shared python colors and helpers
├── libexec/                # Subcommands (not on PATH)
│   ├── reta-config
│   ├── reta-doctor
│   ├── reta-env
│   ├── reta-init
│   ├── reta-init-env
│   ├── reta-port
│   ├── reta-uninstall
│   └── reta-update
├── templates/              # Files copied into new projects
│   ├── Makefile
│   ├── .env.example
│   ├── .gitignore
│   └── README.md
├── install.sh              # Symlinks ~/.reta and ~/.local/bin/reta
├── VERSION
└── README.md               # You are here
```

## Development

After editing a subcommand or `libexec` script, re-run `install.sh` to refresh executable bits:

```
./install.sh
reta doctor
```

To test changes without touching your real projects:

```
rm -rf /tmp/reta-test && mkdir /tmp/reta-test && cd /tmp/reta-test
reta init
make help
```

Colors are centralized in `lib/ui.sh` and `lib/ui.py`. Both honor `NO_COLOR`
and automatically disable ANSI codes when output is piped or redirected.

## Testing

`reta` includes a zero-dependency automated test suite (unit, CLI, and integration tests with GNU Make). 

Run all tests locally:
```
./tests/run.sh
```

## Roadmap

- [x] Subcommand dispatcher (`reta <cmd>`)
- [x] Project scaffolding (`reta init`)
- [x] Makefile adapter (`reta init --convert`)
- [x] `.env` generation & syncing (`reta init-env`, `reta env sync`)
- [x] Env inspection (`reta env check|list|diff`)
- [x] Persistent settings (`reta config`)
- [x] Port manager (`reta port list|kill`)
- [x] Secret leak scanner (`make check-secrets`)
- [x] AI Context export with token estimate (`make export SRC=...`)
- [x] Installation health check (`reta doctor`)
- [x] Self-update (`reta update`)
- [x] Clean uninstall (`reta uninstall`)
- [x] Context-aware "next steps" hints
- [x] Shared UI helpers (bash + python)
- [x] Shell completion for bash and zsh
- [x] Automated test suite (pure bash + python unittest)
- [ ] Per-project template variants (e.g. `reta init --template python-api`)

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contact

Open an issue on [GitHub](https://github.com/eraygdev/reta-toolkit/issues).
