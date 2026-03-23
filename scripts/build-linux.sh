#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
APP_NAME=${APP_NAME:-ntrip-bot}
OUTPUT_DIR=${OUTPUT_DIR:-"$ROOT_DIR/dist"}
GOCACHE_DIR=${GOCACHE:-"$ROOT_DIR/.gocache"}
GOMODCACHE_DIR=${GOMODCACHE:-"$ROOT_DIR/.gomodcache"}

mkdir -p "$OUTPUT_DIR" "$GOCACHE_DIR" "$GOMODCACHE_DIR"

cd "$ROOT_DIR"
GOCACHE="$GOCACHE_DIR" GOMODCACHE="$GOMODCACHE_DIR" GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/$APP_NAME" .

chmod +x "$OUTPUT_DIR/$APP_NAME"
echo "Built $OUTPUT_DIR/$APP_NAME"
