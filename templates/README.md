<!--
═══════════════════════════════════════════════════════════════════════════
AI INSTRUCTIONS — read before filling this template:

1. Replace every [BRACKETED_PLACEHOLDER] with real content verified from the codebase.
2. Delete sections, table rows, and badges that do not apply:
   - No Docker Hub image → remove Docker Pulls badge.
   - No GitHub Releases binaries → remove Total / Linux / macOS / Windows download badges.
   - No screenshots, GIFs, or demo images found in the repo (e.g. docs/screenshots/, assets/) → remove the Screenshots section. Do NOT invent image paths.
   - Simple script/library → remove Architecture section if redundant.
   - CLI, C app, algorithm benchmark, or dataset repo → delete API/Port/Database lines and adapt Quick Start / Usage / Development commands (e.g., gcc, go build, pytest, npm) instead of forcing `make`.
3. Keep ONLY the Usage subsection(s) that match the project (API, CLI, or Library).
4. Never leave empty `]()` links on static badges; use `![Alt](url)` for non-clickable badges.
5. Do NOT leave "TBD" in the final public README. Either ask/verify the missing detail or remove the unfinished line/section.
6. Verify that a `LICENSE` file actually exists in the repo before linking to `(LICENSE)`.
7. Remove all <!-- HTML comments -->, including this block, from the final output.
8. Keep the tone factual, concise, and engineering-focused. No marketing fluff.
9. Preserve markdown structure cleanly (headings, tables, fenced code blocks).
═══════════════════════════════════════════════════════════════════════════
-->

# [PROJECT_NAME]

> [TAGLINE — one sentence, max 120 characters. What does it do and for whom?]

[![License](https://img.shields.io/badge/License-[LICENSE_TYPE]-blue.svg)](LICENSE)
![Language](https://img.shields.io/badge/[PRIMARY_LANGUAGE]-[VERSION]-informational)
![Status](https://img.shields.io/badge/status-[STATUS]-brightgreen)
[![CI](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/actions/workflows/[WORKFLOW_FILE]/badge.svg)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/actions/workflows/[WORKFLOW_FILE])
[![Release](https://img.shields.io/github/v/release/[GITHUB_USERNAME]/[REPO_NAME])](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/releases)

[![Stars](https://img.shields.io/github/stars/[GITHUB_USERNAME]/[REPO_NAME]?style=flat)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/stargazers)
[![Forks](https://img.shields.io/github/forks/[GITHUB_USERNAME]/[REPO_NAME]?style=flat)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/network/members)
[![Pull Requests](https://img.shields.io/github/issues-pr/[GITHUB_USERNAME]/[REPO_NAME]?label=PRs%20open)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/pulls)
[![Closed PRs](https://img.shields.io/github/issues-pr-closed/[GITHUB_USERNAME]/[REPO_NAME]?label=PRs%20closed)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/pulls?q=is%3Apr+is%3Aclosed)
[![Docker Pulls](https://img.shields.io/docker/pulls/[DOCKER_USERNAME]/[IMAGE_NAME])](https://hub.docker.com/r/[DOCKER_USERNAME]/[IMAGE_NAME])

[![Total Downloads](https://img.shields.io/github/downloads/[GITHUB_USERNAME]/[REPO_NAME]/total?label=total%20downloads)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/releases)
[![Linux Downloads](https://img.shields.io/github/downloads/[GITHUB_USERNAME]/[REPO_NAME]/latest/[LINUX_ASSET_NAME]?label=linux%20downloads)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/releases)
[![macOS Downloads](https://img.shields.io/github/downloads/[GITHUB_USERNAME]/[REPO_NAME]/latest/[MACOS_ASSET_NAME]?label=macos%20downloads)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/releases)
[![Windows Downloads](https://img.shields.io/github/downloads/[GITHUB_USERNAME]/[REPO_NAME]/latest/[WINDOWS_ASSET_NAME]?label=windows%20downloads)](https://github.com/[GITHUB_USERNAME]/[REPO_NAME]/releases)

---

## Overview

[OVERVIEW — 3-5 sentences. What problem does this solve? Why does it exist? What key engineering trade-offs or architectural choices make it effective?]

## Screenshots

### [SCREENSHOT_1_TITLE]

![[SCREENSHOT_1_ALT]]([SCREENSHOT_1_PATH])

### [SCREENSHOT_2_TITLE]

![[SCREENSHOT_2_ALT]]([SCREENSHOT_2_PATH])

## Architecture

[ARCHITECTURE_SUMMARY — 2-3 sentences explaining component boundaries, data flow, or concurrency/storage model.]

```text
[Client / CLI] ──▶ [Core Service / Engine] ──▶ [Storage / External API]
```

## Features

- **[FEATURE_NAME]** — [One line: what it does and why it matters]
- **[FEATURE_NAME]** — [One line: what it does and why it matters]
- **[FEATURE_NAME]** — [One line: what it does and why it matters]
- **[FEATURE_NAME]** — [One line: what it does and why it matters]

## Tech Stack

| Layer | Technology |
| --- | --- |
| Language | [LANGUAGE + VERSION] |
| Framework | [FRAMEWORK] |
| Database | [DATABASE] |
| Cache | [CACHE] |
| Container | [CONTAINER_RUNTIME] |
| Testing | [TEST_FRAMEWORK] |

## Quick Start

### Prerequisites

- [PREREQUISITE_1 — e.g. Go 1.26+ or Python 3.12+]
- [PREREQUISITE_2 — e.g. Docker / GCC]
- [PREREQUISITE_3 — e.g. reta (optional)]

### Installation

```bash
# Clone
git clone https://github.com/[GITHUB_USERNAME]/[REPO_NAME].git
cd [REPO_NAME]

# Environment (omit if no .env is used)
cp .env.example .env
# Edit .env with your values (or run: reta init-env)

# Build / Install dependencies
[INSTALL_COMMAND — e.g. make install / go build ./... / gcc main.c -o app]

# Run
[START_COMMAND — e.g. make up / ./app]
```

The service is now running at (omit for CLI / library / dataset repos):

- API: <http://localhost:[PORT]>
- Docs: <http://localhost:[PORT]/docs>
- Database: `[DATABASE_URL]`

## Configuration

All configuration is managed via environment variables (see `.env.example`).

| Variable | Description | Default |
| --- | --- | --- |
| `[VAR_NAME]` | [Description] | `[DEFAULT_OR_REQUIRED]` |
| `[VAR_NAME]` | [Description] | `[DEFAULT_OR_REQUIRED]` |
| `[VAR_NAME]` | [Description] | `[DEFAULT_OR_REQUIRED]` |

## Usage

### API

```bash
curl -s http://localhost:[PORT]/[ENDPOINT]
# [EXPECTED_RESPONSE]
```

### CLI

```bash
[CLI_COMMAND] [ARGS]
# [EXPECTED_OUTPUT]
```

### Library

```[LANGUAGE]
[MINIMAL_WORKING_EXAMPLE — 5-10 lines]
```

## Project Structure

```text
[REPO_NAME]/
├── [DIR_1]/                # [Description]
├── [DIR_2]/                # [Description]
├── [DIR_3]/                # [Description]
├── .env.example            # Environment template
├── Makefile                # Common tasks (run `make help`)
└── README.md               # You are here
```

## Development

Common workflow commands:

```bash
make up              # Start the stack
make down            # Stop the stack
make logs            # Stream logs
make test            # Run tests
make lint            # Lint code
make format          # Format code
```

## Testing

```bash
[TEST_COMMAND — e.g. make test / go test ./... / pytest]
[COVERAGE_COMMAND — e.g. make test-cov / go test -cover ./...]
```

## Deployment

[DEPLOYMENT_NOTES — Docker, binary releases, cloud provider, or local compilation instructions.]

## Roadmap

- [x] [COMPLETED_ITEM]
- [x] [COMPLETED_ITEM]
- [ ] [PLANNED_ITEM]
- [ ] [PLANNED_ITEM]

## License

[LICENSE_TYPE] License — see [LICENSE](LICENSE) for details.

## Contact

[CONTACT_INFO — email, LinkedIn, or "Open an issue in this repository."]