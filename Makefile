PREFIX ?= /usr/local
GIT_VERSION = $(shell git describe --tags )

alr:
	CGO_ENABLED=0 go build -ldflags="-X 'go.elara.ws/alr/internal/config.Version=$(GIT_VERSION)'"

clean:
	rm -f alr

install: alr installmisc
	install -Dm755 alr $(DESTDIR)$(PREFIX)/bin/alr

installmisc:
	install -Dm755 scripts/completion/bash $(DESTDIR)$(PREFIX)/share/bash-completion/completions/alr
	install -Dm755 scripts/completion/zsh $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_alr

uninstall:
	rm -f /usr/local/bin/alr

.PHONY: install clean uninstall installmisc alr