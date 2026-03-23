APP := ntrip-bot
GOCACHE := $(CURDIR)/.gocache
GOMODCACHE := $(CURDIR)/.gomodcache

.PHONY: run build clean linux-build linux-package

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run .

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o $(APP) .

linux-build:
	sh ./scripts/build-linux.sh

linux-package:
	sh ./scripts/package-linux.sh

clean:
	rm -f $(APP)
