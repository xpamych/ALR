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

.PHONY: build install clean clear uninstall check-no-root

build: check-no-root $(BIN)

export CGO_ENABLED := 0
$(BIN):
	go build -ldflags="-X 'gitea.plemya-x.ru/xpamych/ALR/internal/config.Version=$(GIT_VERSION)'" -o $@

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

IGNORE_OLD_FILES := $(foreach file,$(shell cat old-files),-ignore $(file))
update-license:
	go run github.com/google/addlicense@latest -v -f license-header-old-files.tmpl $$(< old-files)
	go run github.com/google/addlicense@latest -v -f license-header.tmpl $(IGNORE_OLD_FILES) .
