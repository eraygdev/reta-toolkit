# reta

> Personal developer toolkit for scaffolding projects, managing secrets, and automating shared workflows.

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Bash](https://img.shields.io/badge/bash-5.0+-4EAA25?logo=gnu-bash&logoColor=white)]()
[![Python](https://img.shields.io/badge/python-3.10+-3776AB?logo=python&logoColor=white)]()
[![Status](https://img.shields.io/badge/status-beta-yellow)]()

---

## Overview

`reta` is a single-command toolkit that bundles the small, repetitive tasks every project needs: scaffolding a new repo with consistent structure, generating `.env` files with auto-filled secrets, sharing a common set of `make` targets across projects, and killing zombie processes on local ports.

Unlike full dotfiles managers (chezmoi, yadm), `reta` focuses on **project-level tooling** — what happens inside each repo — rather than managing `$HOME` itself. Every project includes the same `~/.reta/lib/base.mk`, so improvements to the shared toolkit propagate everywhere without touching individual projects.

## Features

- **Subcommand dispatcher** — a single `reta` command routes to `libexec/reta-*` scripts; new subcommands appear automatically in `reta help`
- **Project scaffolding** — `reta init` copies templates (`Makefile`, `.env.example`, `.gitignore`, `README.md`) into the current directory without overwriting existing files
- **Secret generation** — `reta init-env` reads `.env.example`, replaces `GENERATE_SECRET` / `AUTO_HASH` markers with real values, and prompts for missing keys
- **Env inspection** — `reta env check|list|diff` validates `.env` against `.env.example`, flags missing/empty/duplicate keys, and masks secret values
- **Shared make targets** — `lib/base.mk` is `include`d by every project, giving them `up`, `down`, `logs`, `clean`, `git-sync`, `export`, and more
- **Installation health check** — `reta doctor` verifies symlinks, file permissions, `PATH`, and dependencies
- **Port killer** — `reta killport` lists listening ports interactively (via `fzf` or a numbered menu) and terminates the chosen process with SIGTERM, escalating to SIGKILL if needed
- **Contextual hints** — commands end with "Next steps" suggestions; can be disabled globally via `reta config hints off`
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
- `lsof` (for `reta killport`)
- `fzf` (optional, improves `reta killport`)
- Docker (optional, only if projects use `make up`)

### Installation

**Quick install (single command):**

```bash
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && cd ~/.reta && ./install.sh
```

**Or step by step:**

```bash
git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta
cd ~/.reta
./install.sh
```

> **Important:** The `~/.reta` target is required. `install.sh` installs the
> toolkit by symlinking `~/.local/bin/reta` to it — if you clone somewhere else,
> `reta` will break whenever that source folder moves or is deleted.

If `install.sh` reports that `~/.local/bin` is not on `PATH`, add it to your shell config:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
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
reta init          # Copies Makefile, .env.example, .gitignore, README.md
reta init-env      # Generates .env from .env.example
make help          # Base targets now available
```

### Inspect environment files

```bash
reta env check           # List keys missing or empty in .env
reta env list            # List keys with masked values
reta env list --show     # Reveal real values
reta env diff            # Compare keys between .env and .env.example
```

`reta env` also warns about duplicate keys — a common source of silent
misbehavior across `docker --env-file`, `python-dotenv`, and `source .env`.

### Manage running ports

```bash
reta killport
# ? Select a port to kill:
#   8001  python3     PID=12345  *:8001
#   5432  postgres    PID=67890  127.0.0.1:5432
```

### Configure

```bash
reta config                # Show all settings
reta config hints off      # Hide "Next steps" hints
reta config hints on       # Show them again
reta config --reset        # Restore defaults
```

Temporary override for a single shell:

```bash
RETA_HINTS=off reta env check
```

Settings live in `~/.config/reta/config` and survive `reta uninstall`, so a
fresh reinstall picks up where you left off.

### Update the toolkit

```bash
reta update        # git pull --rebase + rerun install.sh
```

`reta update` shows the version delta (`0.1.0 -> 0.1.1`), a `git diff --stat`
summary, and a clear error message if the pull fails (uncommitted changes or
real merge conflicts).

### Uninstall

```bash
reta uninstall --keep    # remove PATH symlink only; keep ~/.reta
reta uninstall           # remove symlink + ~/.reta (with confirmation)
```

`reta uninstall` refuses to delete anything that doesn't look like a `reta`
installation (verifies `bin/reta` and `libexec/` exist) and never touches your
`~/.bashrc` or `~/.zshrc`.

## Project Structure

```text
reta-toolkit/
├── bin/reta                # Dispatcher — the only command on PATH
├── lib/
│   ├── base.mk             # Shared make targets (included by every project)
│   ├── export-context      # Source export for AI context
│   ├── ui.sh               # Shared bash colors and helpers
│   └── ui.py               # Shared python colors and helpers
├── libexec/                # Subcommands (not on PATH)
│   ├── reta-config
│   ├── reta-doctor
│   ├── reta-env
│   ├── reta-init
│   ├── reta-init-env
│   ├── reta-killport
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

```bash
./install.sh
reta doctor
```

To test changes without touching your real projects:

```bash
rm -rf /tmp/reta-test && mkdir /tmp/reta-test && cd /tmp/reta-test
reta init
make help
```

Colors are centralized in `lib/ui.sh` and `lib/ui.py`. Both honor `NO_COLOR`
and automatically disable ANSI codes when output is piped or redirected.

## Testing

TBD — there is no automated test suite yet. Manual verification via `reta doctor` and a fresh `/tmp` project is the current baseline.

## Roadmap

- [x] Subcommand dispatcher (`reta <cmd>`)
- [x] Project scaffolding (`reta init`)
- [x] `.env` generation (`reta init-env`)
- [x] Env inspection (`reta env check|list|diff`)
- [x] Persistent settings (`reta config`)
- [x] Port killer (`reta killport`)
- [x] Installation health check (`reta doctor`)
- [x] Self-update (`reta update`)
- [x] Clean uninstall (`reta uninstall`)
- [x] Context-aware "next steps" hints
- [x] Shared UI helpers (bash + python)
- [ ] Shell completion for bash and zsh
- [ ] Automated test suite (bats or plain bash)
- [ ] Per-project template variants (e.g. `reta init --template python-api`)
- [ ] macOS-specific fixes for `logs-t` and `xargs -r`

## License

MIT License — see [LICENSE](LICENSE) for details.

## Contact

Open an issue on [GitHub](https://github.com/eraygdev/reta-toolkit/issues).