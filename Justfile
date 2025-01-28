# Quick Labs - Comprehensive Task Runner

# Default task
default:
    @just --list

# 🧰 Initialize Git Hooks, and install pre-commit hooks
hooks-init:
    @echo "🪝 Initializing Git Hooks..."
    @chmod +x scripts/hooks/pre-commit-init.sh
    @./scripts/hooks/pre-commit-init.sh init

# 🔍 Run pre-commit hooks across repository
hooks-run:
    @echo "🔍 Running pre-commit hooks across repository..."
    @./scripts/hooks/pre-commit-init.sh run
    @echo "Hooks execution completed! ✅"
