COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose")

# ═══════════════════════════════════════════════════════════
# reta / lib/base.mk
# Shared make targets for all projects.
# Path: ~/.reta/lib/base.mk
# ═══════════════════════════════════════════════════════════

SHELL := /bin/bash
.DEFAULT_GOAL := help

# === Help ===

help:  ## Show all targets (base + project)
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    print ""; \
	    printf "  \033[1;33m%s\033[0m\n", line; \
	    next \
	  } \
	  /^[a-zA-Z_-]+:.*?## / { \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 \
	  }' $(HOME)/.reta/lib/base.mk
	@proj=$$(awk 'BEGIN {FS = ":.*?## "} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    print ""; \
	    printf "  \033[1;33m%s\033[0m\n", line; \
	    next \
	  } \
	  /^[a-zA-Z_-]+:.*?## / { \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 \
	  }' $(firstword $(MAKEFILE_LIST))); \
	if [ -n "$$proj" ]; then \
	  echo ""; \
	  printf "  \033[2m────────────────────────────────────────\033[0m\n"; \
	  printf "  \033[1;35mProject-specific targets\033[0m\n"; \
	  printf "%s\n" "$$proj"; \
	fi

help-base:  ## Show only base targets
	@echo "Base targets:"
	@awk 'BEGIN {FS = ":.*?## "} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    print ""; \
	    printf "  \033[1;33m%s\033[0m\n", line; \
	    next \
	  } \
	  /^[a-zA-Z_-]+:.*?## / { \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 \
	  }' $(HOME)/.reta/lib/base.mk

help-project:  ## Show only project-specific targets
	@echo "Project-specific targets:"
	@awk 'BEGIN {FS = ":.*?## "} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    print ""; \
	    printf "  \033[1;33m%s\033[0m\n", line; \
	    next \
	  } \
	  /^[a-zA-Z_-]+:.*?## / { \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 \
	  }' $(firstword $(MAKEFILE_LIST))

# === Docker ===

up:  ## Start the stack
	@$(COMPOSE) up -d

down:  ## Stop the stack
	@$(COMPOSE) down

restart:  ## Restart services
	@$(COMPOSE) restart

rebuild:  ## Rebuild and restart (force-recreate)
	@$(COMPOSE) up -d --build --force-recreate

ps:  ## Show service status
	@$(COMPOSE) ps

sh:  ## Open shell in a container (make sh SERVICE=api)
	@if [ -z "$(SERVICE)" ]; then \
		echo "Error: usage: make sh SERVICE=<service>"; \
		echo ""; \
		echo "Available services:"; \
		$(COMPOSE) config --services 2>/dev/null | sed 's/^/  - /' || echo "  ($(COMPOSE) config unavailable)"; \
		exit 1; \
	fi
	@$(COMPOSE) exec $(SERVICE) sh

# === Logs ===

logs:  ## Stream logs (this terminal)
	@$(COMPOSE) logs -f --tail 50

logs-all:  ## Stream logs from all services (no prefix)
	@$(COMPOSE) logs --no-log-prefix -f --tail 50

logs-t:  ## Stream logs in a new terminal tab
	@gnome-terminal --tab -- bash -c "cd $(PWD) && $(COMPOSE) logs -f --tail 50; exec bash" 2>/dev/null || \
	 x-terminal-emulator -e "bash -c 'cd $(PWD) && $(COMPOSE) logs -f --tail 50'" 2>/dev/null || \
	 konsole --new-tab -e bash -c "cd $(PWD) && $(COMPOSE) logs -f --tail 50" 2>/dev/null || \
	 (echo "Warning: could not open new tab, falling back to 'make logs'"; cd $(PWD) && $(COMPOSE) logs -f --tail 50)

logs-errors:  ## Stream only errors (live filter)
	@$(COMPOSE) logs -f --tail 0 2>&1 | grep --line-buffered -iE "error|fatal|exception|traceback"

logs-warn:  ## Stream only warnings (live filter)
	@$(COMPOSE) logs -f --tail 0 2>&1 | grep --line-buffered -iE "warn"

# === Clean ===

clean:  ## Remove Python caches
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name "*.pyc" -delete 2>/dev/null || true
	@find . -type d \( -name ".pytest_cache" -o -name ".mypy_cache" -o -name ".ruff_cache" \) -exec rm -rf {} + 2>/dev/null || true
	@rm -rf .coverage coverage.xml htmlcov 2>/dev/null || true
	@echo "Cache cleaned."

clean-docker:  ## Safe docker prune (volumes preserved)
	@docker builder prune -f
	@docker image prune -f
	@docker container prune -f
	@echo "Cleaned (volumes preserved)."

# === Export ===

export:  ## Export source for AI context
	@if [ ! -x "$(HOME)/.reta/lib/export-context" ]; then \
		echo "Error: export-context not found at ~/.reta/lib/export-context"; \
		exit 1; \
	fi
	@$(HOME)/.reta/lib/export-context

clean-export:  ## Remove old export and regenerate
	@rm -f project_export.md exports/project_export.md
	@$(MAKE) --no-print-directory export

# === Git ===

git-sync:  ## Fetch, rebase, delete gone branches
	@git fetch --prune
	@git pull --rebase
	@git branch -vv | grep ': gone]' | awk '{print $$1}' | xargs -r git branch -D 2>/dev/null || \
	 git branch -vv | grep ': gone]' | awk '{print $$1}' | xargs git branch -D 2>/dev/null || true
	@echo "Sync complete."

git-status:  ## Short git status
	@git status -sb

# --- PHONY ---

.PHONY: help help-base help-project \
        up down restart rebuild ps sh \
        logs logs-all logs-t logs-errors logs-warn \
        clean clean-docker \
        export clean-export \
        git-sync git-status