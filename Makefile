APP := ntrip-bot
GOCACHE := $(CURDIR)/.gocache
GOMODCACHE := $(CURDIR)/.gomodcache

.PHONY: run build clean

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run .

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -o $(APP) .

clean:
	rm -f $(APP)
