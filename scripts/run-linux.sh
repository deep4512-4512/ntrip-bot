#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
APP_NAME=${APP_NAME:-ntrip-bot}
BIN_PATH=${BIN_PATH:-"$ROOT_DIR/dist/$APP_NAME"}

cd "$ROOT_DIR"

if [ ! -x "$BIN_PATH" ]; then
  echo "Binary not found: $BIN_PATH"
  echo "Run scripts/build-linux.sh first."
  exit 1
fi

exec "$BIN_PATH"
