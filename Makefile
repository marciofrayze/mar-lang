GOCACHE ?= $(CURDIR)/.gocache
GO_MIN_VERSION := 1.26
ELM_REQUIRED_VERSION := 0.19.1
GO_VERSION := $(shell go version | awk '{print $$3}' 2>/dev/null | sed 's/^go//')

.PHONY: all check check-go check-elm admin website compiler-assets mar todo dev-todo test clean distclean

define print_title
	@sh -c 'if [ -n "$$NO_COLOR" ] || ! [ -t 1 ]; then printf "\n%s\n" "$(1)"; else printf "\n\033[1;36m%s\033[0m\n" "$(1)"; fi'
endef

define print_info
	@sh -c 'printf "  %s\n" "$(1)"'
endef

define print_ok
	@sh -c 'if [ -n "$$NO_COLOR" ] || ! [ -t 1 ]; then printf "  %s\n" "$(1)"; else printf "  \033[1;32m%s\033[0m\n" "$(1)"; fi'
endef

all: mar
	$(call print_title,Mar compiler ready)
	$(call print_ok,./mar)
	@printf "\n"

check: check-go check-elm

check-go:
	@command -v go >/dev/null 2>&1 || { \
		echo "Go $(GO_MIN_VERSION)+ is required for this step. Install Go and try again."; \
		exit 1; \
	}
	@GO_VERSION="$$(go version | awk '{print $$3}' | sed 's/^go//')"; \
	if [ -z "$$GO_VERSION" ]; then \
		echo "Could not determine Go version."; \
		exit 1; \
	fi; \
	if ! printf '%s\n%s\n' "$(GO_MIN_VERSION)" "$$GO_VERSION" | sort -V -C; then \
		echo "Go $(GO_MIN_VERSION)+ is required. Found $$GO_VERSION."; \
		exit 1; \
	fi

check-elm:
	@command -v elm >/dev/null 2>&1 || { \
		echo "Elm $(ELM_REQUIRED_VERSION) is required for this step. Install Elm and try again."; \
		exit 1; \
	}
	@ELM_VERSION="$$(elm --version 2>/dev/null)"; \
	if [ "$$ELM_VERSION" != "$(ELM_REQUIRED_VERSION)" ]; then \
		echo "Elm $(ELM_REQUIRED_VERSION) is required. Found $$ELM_VERSION."; \
		exit 1; \
	fi

admin: check-elm
	$(call print_title,Admin UI)
	$(call print_info,Building admin/dist/app.js with Elm $(ELM_REQUIRED_VERSION))
	@cd admin && elm make src/Main.elm --output=dist/app.js
	$(call print_ok,Output: admin/dist/app.js)

website: check-elm
	$(call print_title,Website)
	$(call print_info,Building website/dist/app.js with Elm $(ELM_REQUIRED_VERSION))
	@cd website && elm make src/Main.elm --output=dist/app.js
	$(call print_ok,Output: website/dist/app.js)

compiler-assets: check-go admin
	$(call print_title,Compiler assets)
	$(call print_info,Refreshing embedded admin assets and runtime stubs)
	@GOCACHE="$(GOCACHE)" ./scripts/build-compiler-assets.sh
	$(call print_ok,Embedded admin assets: internal/cli/compiler_assets/admin)
	$(call print_ok,Runtime stubs: internal/cli/runtime_stubs)

mar: check-go compiler-assets
	$(call print_title,Mar compiler)
	$(call print_info,Building ./mar with Go $(GO_VERSION))
	@GOCACHE="$(GOCACHE)" go build -o mar ./cmd/mar
	$(call print_ok,Output: ./mar)

todo: mar
	$(call print_title,Example app)
	$(call print_info,Compiling examples/todo.mar into dist/)
	@./mar compile examples/todo.mar

dev-todo: mar
	$(call print_title,Example app)
	$(call print_info,Starting dev mode for examples/todo.mar)
	@./mar dev examples/todo.mar

test: check-go
	$(call print_title,Tests)
	$(call print_info,Running app bundle, CLI, and runtime-stub tests)
	@GOCACHE="$(GOCACHE)" go test ./internal/appbundle ./internal/cli ./cmd/mar-app

clean:
	$(call print_title,Clean)
	$(call print_info,Removing Go cache)
	@rm -rf "$(GOCACHE)"

distclean: clean
	$(call print_info,Removing dist/)
	@rm -rf "$(CURDIR)/dist"
