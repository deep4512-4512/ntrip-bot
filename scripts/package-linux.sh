#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
APP_NAME=${APP_NAME:-ntrip-bot}
VERSION=${VERSION:-$(date +%Y%m%d-%H%M%S)}
PACKAGE_ROOT="$ROOT_DIR/dist/${APP_NAME}-linux-amd64"
ARCHIVE_PATH="$ROOT_DIR/dist/${APP_NAME}-linux-amd64-${VERSION}.tar.gz"

sh "$ROOT_DIR/scripts/build-linux.sh"

rm -rf "$PACKAGE_ROOT"
mkdir -p "$PACKAGE_ROOT/scripts" "$PACKAGE_ROOT/deploy"

cp "$ROOT_DIR/dist/$APP_NAME" "$PACKAGE_ROOT/$APP_NAME"
cp "$ROOT_DIR/bot_settings.json" "$PACKAGE_ROOT/bot_settings.json"
cp "$ROOT_DIR/config.example.json" "$PACKAGE_ROOT/config.example.json"
cp "$ROOT_DIR/install.sh" "$PACKAGE_ROOT/install.sh"
cp "$ROOT_DIR/update.sh" "$PACKAGE_ROOT/update.sh"
cp "$ROOT_DIR/remove.sh" "$PACKAGE_ROOT/remove.sh"
cp "$ROOT_DIR/README.md" "$PACKAGE_ROOT/README.md"
cp "$ROOT_DIR/scripts/run-linux.sh" "$PACKAGE_ROOT/scripts/run-linux.sh"
cp "$ROOT_DIR/scripts/install-service.sh" "$PACKAGE_ROOT/scripts/install-service.sh"
cp "$ROOT_DIR/deploy/ntrip-bot.service" "$PACKAGE_ROOT/deploy/ntrip-bot.service"

chmod +x "$PACKAGE_ROOT/$APP_NAME" "$PACKAGE_ROOT/install.sh" "$PACKAGE_ROOT/update.sh" "$PACKAGE_ROOT/remove.sh" "$PACKAGE_ROOT/scripts/run-linux.sh" "$PACKAGE_ROOT/scripts/install-service.sh"

tar -C "$ROOT_DIR/dist" -czf "$ARCHIVE_PATH" "${APP_NAME}-linux-amd64"
echo "Created $ARCHIVE_PATH"
