# Project Makefile (powered by ~/.reta/lib/base.mk)
#
# Rules & usage:
#   - Do NOT define `help:` — `base.mk` generates a combined `help` for all targets.
#   - Override any other base target (e.g. `clean`, `up`) by defining it below
#     `include $(RETA_BASE)`. (Converted Makefiles comment out colliding rules
#     first; uncomment any of them to restore your custom behavior.)
#   - Section heading: `# === Section Name ===` (shown as a bold yellow header).
#   - Documented target: `name:  ## Short description` (targets without `##`
#     still run normally, but stay hidden from `make help`).

RETA_BASE := $(HOME)/.reta/lib/base.mk
ifeq ($(wildcard $(RETA_BASE)),)
$(info ❌ [reta] reta toolkit is not installed.)
$(info Install: git clone https://github.com/eraygdev/reta-toolkit.git ~/.reta && ~/.reta/install.sh)
$(error $(RETA_BASE) missing)
endif
include $(RETA_BASE)

# === BASE_MATRIX ===

# Add your project-specific targets below.
#
# Examples:
#
#   # === Database ===
#
#   migrate:  ## Apply database migrations
#   	alembic upgrade head
#
#   seed:  ## Fill the database with sample data
#   	python scripts/seed.py
#
#   # === Training ===
#
#   train:  ## Run model training
#   	poetry run python -m src.train
#
#   evaluate:  ## Evaluate the model on the test set
#   	poetry run python -m src.evaluate
