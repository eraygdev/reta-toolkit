COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose")

# ═══════════════════════════════════════════════════════════
# reta / lib/base.mk
# Shared make targets for all projects.
# Path: ~/.reta/lib/base.mk
# ═══════════════════════════════════════════════════════════

SHELL := /bin/bash
.DEFAULT_GOAL := help
DIR ?= exports
SRC ?= .

# === Help ===

help:  ## Show all targets (base + project)
	@echo "Available targets:"
	@awk -v dir="$(DIR)" 'BEGIN {FS = ":.*?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:.*?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	    } \
	    desc = $$2; \
	    gsub(/\$$\(DIR\)/, dir, desc); \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, desc \
	  }' $(HOME)/.reta/lib/base.mk
	@proj=$$(awk 'BEGIN {FS = ":.*?## "; pending = ""; first = 1} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:.*?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	      first = 0; \
	    } else if (first) { \
	      print ""; \
	      first = 0; \
	    } \
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
	@awk -v dir="$(DIR)" 'BEGIN {FS = ":.*?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:.*?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	    } \
	    desc = $$2; \
	    gsub(/\$$\(DIR\)/, dir, desc); \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, desc \
	  }' $(HOME)/.reta/lib/base.mk

help-project:  ## Show only project-specific targets
	@echo "Project-specific targets:"
	@awk 'BEGIN {FS = ":.*?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:.*?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	    } \
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
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		osascript -e 'tell application "Terminal" to do script "cd \"$(CURDIR)\" && $(COMPOSE) logs -f --tail 50"' >/dev/null 2>&1 || \
		(echo "Warning: could not open Terminal.app, falling back to 'make logs'"; cd "$(CURDIR)" && $(COMPOSE) logs -f --tail 50); \
	else \
		gnome-terminal --tab -- bash -c "cd \"$(CURDIR)\" && $(COMPOSE) logs -f --tail 50; exec bash" 2>/dev/null || \
		x-terminal-emulator -e "bash -c 'cd \"$(CURDIR)\" && $(COMPOSE) logs -f --tail 50'" 2>/dev/null || \
		konsole --new-tab -e bash -c "cd \"$(CURDIR)\" && $(COMPOSE) logs -f --tail 50" 2>/dev/null || \
		(echo "Warning: could not open new tab, falling back to 'make logs'"; cd "$(CURDIR)" && $(COMPOSE) logs -f --tail 50); \
	fi

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

export:  ## Export source for AI context (DIR=$(DIR))
	@if [ ! -x "$(HOME)/.reta/lib/export-context" ]; then \
		echo "Error: export-context not found at ~/.reta/lib/export-context"; \
		exit 1; \
	fi
	@if [ "$(SRC)" = "." ]; then \
		$(HOME)/.reta/lib/export-context --dir "$(DIR)"; \
	elif [ "$(DIR)" = "." ]; then \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(CURDIR)/project_export.md"; \
	elif [[ "$(DIR)" = /* ]]; then \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(DIR)/project_export.md"; \
	else \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(CURDIR)/$(DIR)/project_export.md"; \
	fi

clean-export:  ## Remove old export and regenerate (DIR=$(DIR))
	@if [ ! -x "$(HOME)/.reta/lib/export-context" ]; then \
		echo "Error: export-context not found at ~/.reta/lib/export-context"; \
		exit 1; \
	fi
	@if [ "$(SRC)" = "." ]; then \
		$(HOME)/.reta/lib/export-context --dir "$(DIR)" --clean; \
	elif [ "$(DIR)" = "." ]; then \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(CURDIR)/project_export.md" --clean; \
	elif [[ "$(DIR)" = /* ]]; then \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(DIR)/project_export.md" --clean; \
	else \
		$(HOME)/.reta/lib/export-context -r "$(SRC)" -o "$(CURDIR)/$(DIR)/project_export.md" --clean; \
	fi

clean-exports:  ## Delete all files inside exports/ (with confirmation)
	@if [ ! -x "$(HOME)/.reta/lib/export-context" ]; then \
		echo "Error: export-context not found at ~/.reta/lib/export-context"; \
		exit 1; \
	fi
	@$(HOME)/.reta/lib/export-context --dir "$(DIR)" --clean-exports

# === Git ===

git-sync:  ## Fetch, rebase, delete gone branches
	@git fetch --prune
	@git pull --rebase
	@git branch -vv | grep ': gone]' | awk '{print $$1}' | xargs -r git branch -D 2>/dev/null || \
	 git branch -vv | grep ': gone]' | awk '{print $$1}' | xargs git branch -D 2>/dev/null || true
	@echo "Sync complete."

git-status:  ## Short git status
	@git status -sb

# === Security ===

check-secrets:  ## Scan repo for tracked .env files and leaked secrets
	@if [ ! -x "$(HOME)/.reta/lib/check-secrets" ]; then \
		echo "Error: check-secrets not found at ~/.reta/lib/check-secrets"; \
		exit 1; \
	fi
	@$(HOME)/.reta/lib/check-secrets

# --- PHONY ---

.PHONY: help help-base help-project \
        up down restart rebuild ps sh \
        logs logs-all logs-t logs-errors logs-warn \
        clean clean-docker \
        export clean-export clean-exports \
        git-sync git-status \
        check-secrets