NAME := alr
GIT_VERSION = $(shell git describe --tags )

DESTDIR ?=
PREFIX ?= /usr/local
BIN := ./$(NAME)
INSTALED_BIN := $(DESTDIR)/$(PREFIX)/bin/$(NAME)
COMPLETIONS_DIR := ./scripts/completion
BASH_COMPLETION := $(COMPLETIONS_DIR)/bash
ZSH_COMPLETION := $(COMPLETIONS_DIR)/zsh
INSTALLED_BASH_COMPLETION := $(DESTDIR)$(PREFIX)/share/bash-completion/completions/$(NAME)
INSTALLED_ZSH_COMPLETION := $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_$(NAME)

ADD_LICENSE_BIN := go run github.com/google/addlicense@4caba19b7ed7818bb86bc4cd20411a246aa4a524
GOLANGCI_LINT_BIN := go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
XGOTEXT_BIN := go run github.com/Tom5521/xgotext@v1.2.0

.PHONY: build install clean clear uninstall check-no-root

build: check-no-root $(BIN)

export CGO_ENABLED := 0
$(BIN):
	go build -ldflags="-X 'gitea.plemya-x.ru/Plemya-x/ALR/internal/config.Version=$(GIT_VERSION)'" -o $@

check-no-root:
	@if [[ "$$(whoami)" == 'root' ]]; then \
		echo "This target shouldn't run as root" 1>&2; \
		exit 1; \
	fi

install: \
	$(INSTALED_BIN) \
	$(INSTALLED_BASH_COMPLETION) \
	$(INSTALLED_ZSH_COMPLETION)
	@echo "Installation done!"

$(INSTALED_BIN): $(BIN)
	install -Dm755 $< $@
	setcap cap_setuid,cap_setgid+ep $(INSTALED_BIN)

$(INSTALLED_BASH_COMPLETION): $(BASH_COMPLETION)
	install -Dm755 $< $@

$(INSTALLED_ZSH_COMPLETION): $(ZSH_COMPLETION)
	install -Dm755 $< $@

uninstall:
	rm -f \
		$(INSTALED_BIN) \
		$(INSTALLED_BASH_COMPLETION) \
		$(INSTALLED_ZSH_COMPLETION)

clean clear:
	rm -f $(BIN)

OLD_FILES=$$(< old-files)
IGNORE_OLD_FILES := $(foreach file,$(shell cat old-files),-ignore $(file))
update-license:
	$(ADD_LICENSE_BIN) -v -f license-header-old-files.tmpl $(OLD_FILES)
	$(ADD_LICENSE_BIN) -v -f license-header.tmpl $(IGNORE_OLD_FILES) .

fmt:
	$(GOLANGCI_LINT_BIN) run --fix

i18n:
	$(XGOTEXT_BIN)  --output ./internal/translations/default.pot
	msguniq --use-first -o ./internal/translations/default.pot ./internal/translations/default.pot
	msgmerge --backup=off -U ./internal/translations/po/ru/default.po ./internal/translations/default.pot
	bash scripts/i18n-badge.sh

test-coverage:
	go test ./... -v -coverpkg=./... -coverprofile=coverage.out
	bash scripts/coverage-badge.sh

update-deps-cve:
	bash scripts/update-deps-cve.sh

e2e-test: clean build
	rm -f ./e2e-tests/alr
	cp alr e2e-tests
	go test -tags=e2e ./...