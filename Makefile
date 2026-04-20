# Makefile — thin wrapper around Taskfile.yaml.
#
# ADR-043 references `make generate` and `make check-generated`.
# This file honors that mental model while the heavy lifting happens in Taskfile.yaml.
#
# On Debian/Ubuntu: task is `go install github.com/go-task/task/v3/cmd/task@latest`.
# If Task isn't installed, the first target prints guidance and exits.
#
# Every target here is a pass-through. Prefer `task <target>` directly for tab completion
# and task --watch; use `make` when your fingers haven't learned Task yet or in CI
# shims that expect Make.

TASK := task

# Guard: fail fast with actionable message if Task isn't installed.
define require_task
	@command -v $(TASK) >/dev/null 2>&1 || { \
		echo "task not installed. Install: go install github.com/go-task/task/v3/cmd/task@latest"; \
		echo "Or see: https://taskfile.dev/installation/"; \
		exit 1; \
	}
endef

.PHONY: help
help:
	$(call require_task)
	@$(TASK) --list

.PHONY: check
check:
	$(call require_task)
	@$(TASK) check

.PHONY: fmt
fmt:
	$(call require_task)
	@$(TASK) fmt

.PHONY: test
test:
	$(call require_task)
	@$(TASK) test

# ADR-043 references this target explicitly.
.PHONY: generate
generate:
	$(call require_task)
	@$(TASK) gen

# ADR-043 Phase 1 item 5 CI gate.
.PHONY: check-generated
check-generated:
	$(call require_task)
	@$(TASK) check:openapi:generated

.PHONY: install
install:
	$(call require_task)
	@$(TASK) install

.PHONY: hooks
hooks:
	$(call require_task)
	@$(TASK) hooks

.PHONY: build
build:
	$(call require_task)
	@$(TASK) build

.PHONY: clean
clean:
	$(call require_task)
	@$(TASK) clean

# Escape hatches for specific sub-tasks — pass everything through.
.PHONY: %
%:
	$(call require_task)
	@$(TASK) $@
