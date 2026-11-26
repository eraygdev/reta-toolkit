COMPOSE = $(eval COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose"))$(COMPOSE)

# ═══════════════════════════════════════════════════════════
# reta / lib/base.mk
# Shared make targets for all projects.
# Path: ~/.reta/lib/base.mk
# ═══════════════════════════════════════════════════════════

SHELL := /bin/bash
.DEFAULT_GOAL := help
DIR ?= exports
SRC ?= .

LEARN ?= $(shell if [ "$${RETA_LEARN:-}" = "on" ]; then echo on; elif [ "$${RETA_LEARN:-}" = "off" ]; then echo off; elif [ -f $(HOME)/.config/reta/config ] && grep -qs '^learn=on' $(HOME)/.config/reta/config; then echo on; else echo off; fi)
ifeq ($(LEARN),on)
SHOW_CMD = @printf "  \033[2mCommand: %s\033[0m\n" "$(1)"
else
SHOW_CMD = @:
endif

# === Help ===

help:  ## Show all targets (base + project)
	@echo "Available targets:"
	@awk -v dir="$(DIR)" 'BEGIN {FS = ":([^=][^#]*)?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:([^=][^#]*)?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	    } \
	    desc = $$2; \
	    gsub(/\$$\(DIR\)/, dir, desc); \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, desc \
	  }' $(HOME)/.reta/lib/base.mk
	@proj=$$(awk 'BEGIN {FS = ":([^=][^#]*)?## "; pending = ""; first = 1} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:([^=][^#]*)?## / { \
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
	@awk -v dir="$(DIR)" 'BEGIN {FS = ":([^=][^#]*)?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:([^=][^#]*)?## / { \
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
	@awk 'BEGIN {FS = ":([^=][^#]*)?## "; pending = ""} \
	  /^# === / { \
	    line = $$0; \
	    sub(/^# === /, "", line); \
	    sub(/ ===$$/, "", line); \
	    pending = line; \
	    next \
	  } \
	  /^[a-zA-Z0-9_.-]+:([^=][^#]*)?## / { \
	    if (pending != "") { \
	      print ""; \
	      printf "  \033[1;33m%s\033[0m\n", pending; \
	      pending = ""; \
	    } \
	    printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 \
	  }' $(firstword $(MAKEFILE_LIST))

# === Docker ===

up:  ## Start the stack
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) up -d)
	@$(COMPOSE) up -d

down:  ## Stop the stack
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) down)
	@$(COMPOSE) down

restart:  ## Restart services
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) restart)
	@$(COMPOSE) restart

rebuild:  ## Rebuild and restart (force-recreate)
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) up -d --build --force-recreate)
	@$(COMPOSE) up -d --build --force-recreate

ps:  ## Show service status
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) ps)
	@$(COMPOSE) ps

sh:  ## Open shell in a container (make sh SERVICE=api)
	@if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ] && [ ! -f compose.yml ] && [ ! -f compose.yaml ]; then \
		echo "Error: No compose file found. This project is not Docker-based."; \
		exit 1; \
	fi
	@if [ -z "$(SERVICE)" ]; then \
		echo "Error: usage: make sh SERVICE=<service>"; \
		echo ""; \
		echo "Available services:"; \
		$(COMPOSE) config --services 2>/dev/null | sed 's/^/  - /' || echo "  ($(COMPOSE) config unavailable)"; \
		exit 1; \
	fi
	$(call SHOW_CMD,$(COMPOSE) exec $(SERVICE) sh)
	@$(COMPOSE) exec $(SERVICE) sh

# === Logs ===

logs:  ## Stream logs (this terminal)
	$(call SHOW_CMD,$(COMPOSE) logs -f --tail 50)
	@$(COMPOSE) logs -f --tail 50

logs-all:  ## Stream logs from all services (no prefix)
	$(call SHOW_CMD,$(COMPOSE) logs --no-log-prefix -f --tail 50)
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
	$(call SHOW_CMD,docker builder prune -f && docker image prune -f && docker container prune -f)
	@docker builder prune -f
	@docker image prune -f
	@docker container prune -f
	@echo "Cleaned (volumes preserved)."

# === Export ===

export:  ## Export source for AI context (DIR=$(DIR))
	@if command -v reta >/dev/null 2>&1 && reta export --help >/dev/null 2>&1; then \
		EXPORTER="reta export"; \
	elif [ -x "$(HOME)/.reta/lib/export-context" ]; then \
		EXPORTER="$(HOME)/.reta/lib/export-context"; \
	else \
		echo "Error: export-context not found"; \
		exit 1; \
	fi; \
	if [ "$(SRC)" = "." ]; then \
		$$EXPORTER --dir "$(DIR)"; \
	elif [ "$(DIR)" = "." ]; then \
		$$EXPORTER -r "$(SRC)" -o "$(CURDIR)/project_export.md"; \
	else \
		case "$(DIR)" in \
			/*) $$EXPORTER -r "$(SRC)" -o "$(DIR)/project_export.md" ;; \
			*)  $$EXPORTER -r "$(SRC)" -o "$(CURDIR)/$(DIR)/project_export.md" ;; \
		esac; \
	fi

clean-export:  ## Remove old export and regenerate (DIR=$(DIR))
	@if command -v reta >/dev/null 2>&1 && reta export --help >/dev/null 2>&1; then \
		EXPORTER="reta export"; \
	elif [ -x "$(HOME)/.reta/lib/export-context" ]; then \
		EXPORTER="$(HOME)/.reta/lib/export-context"; \
	else \
		echo "Error: export-context not found"; \
		exit 1; \
	fi; \
	if [ "$(SRC)" = "." ]; then \
		$$EXPORTER --dir "$(DIR)" --clean; \
	elif [ "$(DIR)" = "." ]; then \
		$$EXPORTER -r "$(SRC)" -o "$(CURDIR)/project_export.md" --clean; \
	else \
		case "$(DIR)" in \
			/*) $$EXPORTER -r "$(SRC)" -o "$(DIR)/project_export.md" --clean ;; \
			*)  $$EXPORTER -r "$(SRC)" -o "$(CURDIR)/$(DIR)/project_export.md" --clean ;; \
		esac; \
	fi

clean-exports:  ## Delete all files inside exports/ (with confirmation)
	@if command -v reta >/dev/null 2>&1 && reta export --help >/dev/null 2>&1; then \
		reta export --dir "$(DIR)" --clean-exports; \
	elif [ -x "$(HOME)/.reta/lib/export-context" ]; then \
		$(HOME)/.reta/lib/export-context --dir "$(DIR)" --clean-exports; \
	else \
		echo "Error: export-context not found"; \
		exit 1; \
	fi

# === Git ===

git-sync:  ## Fetch, rebase, delete gone branches
	$(call SHOW_CMD,git fetch --prune && git pull --rebase)
	@git fetch --prune
	@git pull --rebase
	@git branch -vv | awk '/: gone\]/ { if ($$1 == "*" || $$1 == "+") next; print $$1 }' | xargs -r git branch -D 2>/dev/null || \
	 git branch -vv | awk '/: gone\]/ { if ($$1 == "*" || $$1 == "+") next; print $$1 }' | xargs git branch -D 2>/dev/null || true
	@echo "Sync complete."

git-status:  ## Short git status
	$(call SHOW_CMD,git status -sb)
	@git status -sb

# === Security ===

check-secrets:  ## Scan repo for tracked .env files and leaked secrets
	$(call SHOW_CMD,reta check-secrets)
	@if command -v reta >/dev/null 2>&1 && reta check-secrets --help >/dev/null 2>&1; then \
		reta check-secrets; \
	elif [ -x "$(HOME)/.reta/lib/check-secrets" ]; then \
		$(HOME)/.reta/lib/check-secrets; \
	else \
		echo "Error: check-secrets not found"; \
		exit 1; \
	fi

# --- PHONY ---

.PHONY: help help-base help-project \
        up down restart rebuild ps sh \
        logs logs-all logs-t logs-errors logs-warn \
        clean clean-docker \
        export clean-export clean-exports \
        git-sync git-status \
        check-secrets