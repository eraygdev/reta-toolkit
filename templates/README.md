<!--
═══════════════════════════════════════════════════════════════════════════
AI INSTRUCTIONS — read before filling this template:

1. Replace every [BRACKETED_PLACEHOLDER] with real content from the code.
2. Delete sections that don't apply (e.g. no Redis → remove Redis mentions).
3. Remove all <!-- HTML comments -->, including this one, from final output.
4. Keep the tone factual and professional. No marketing fluff.
5. If a piece of information is missing, write "TBD" — do not invent.
6. Preserve markdown structure exactly (headings, tables, code blocks).
═══════════════════════════════════════════════════════════════════════════
-->

# [PROJECT_NAME]

> [TAGLINE — one sentence, max 120 characters. What does it do and for whom?]

[![License](https://img.shields.io/badge/License-[LICENSE_TYPE]-blue.svg)](LICENSE)
[![Language](https://img.shields.io/badge/[PRIMARY_LANGUAGE]-[VERSION]-informational)]()
[![Status](https://img.shields.io/badge/status-[STATUS]-brightgreen)]()

---

## Overview

[OVERVIEW — 3-5 sentences. What problem does this solve? Why does it exist? What makes it different from alternatives?]

## Features

- **[FEATURE_NAME]** — [One line: what it does]
- **[FEATURE_NAME]** — [One line: what it does]
- **[FEATURE_NAME]** — [One line: what it does]
- **[FEATURE_NAME]** — [One line: what it does]

## Tech Stack

| Layer | Technology |
|---|---|
| Language | [LANGUAGE + VERSION] |
| Framework | [FRAMEWORK] |
| Database | [DATABASE] |
| Cache | [CACHE — omit row if none] |
| Container | [CONTAINER_RUNTIME] |
| Testing | [TEST_FRAMEWORK] |

## Quick Start

### Prerequisites

- [PREREQUISITE_1 — e.g. Python 3.11+]
- [PREREQUISITE_2 — e.g. Docker]
- [PREREQUISITE_3 — e.g. reta]

### Installation

```bash
# Clone
git clone https://github.com/[GITHUB_USERNAME]/[REPO_NAME].git
cd [REPO_NAME]

# Environment
cp .env.example .env
# Edit .env with your values (or run: reta init-env)

# Install dependencies
[INSTALL_COMMAND — e.g. make install]

# Start services
[START_COMMAND — e.g. make up]
```

The app is now running at:

- API: <http://localhost:[PORT]>
- Docs: <http://localhost:[PORT]/docs>
- Database: `[DATABASE_URL]`

## Configuration

All configuration is via environment variables (see `.env.example`).

| Variable | Description | Default |
|---|---|---|
| `[VAR_NAME]` | [Description] | [Default or "required"] |
| `[VAR_NAME]` | [Description] | [Default or "required"] |
| `[VAR_NAME]` | [Description] | [Default or "required"] |

## Usage

### API

```bash
curl http://localhost:[PORT]/[ENDPOINT]
# [EXPECTED_RESPONSE]
```

### CLI

```bash
[CLI_COMMAND] [ARGS]
# [EXPECTED_OUTPUT]
```

### Library

```python
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

Common commands (from `make help`):

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
make test            # Run full test suite
make test-cov        # Run with coverage report
```

## Deployment

[DEPLOYMENT_NOTES — how is this deployed? Docker? Kubernetes? Cloud platform? Specific commands?]

## Roadmap

- [x] [COMPLETED_ITEM]
- [x] [COMPLETED_ITEM]
- [ ] [PLANNED_ITEM]
- [ ] [PLANNED_ITEM]

## License

[LICENSE_TYPE] License — see [LICENSE](LICENSE) for details.

## Contact

[CONTACT_INFO — email, GitHub issues, Discord, or "Open an issue" if none]