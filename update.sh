#!/usr/bin/env sh
set -eu

APP_NAME=${APP_NAME:-ntrip-bot}
REPO=${REPO:-deep4512-4512/ntrip-bot}
INSTALL_DIR=${INSTALL_DIR:-/opt/ntrip-bot}
SERVICE_NAME=${SERVICE_NAME:-ntrip-bot}
SERVICE_USER=${SERVICE_USER:-ntrip}
SERVICE_GROUP=${SERVICE_GROUP:-$SERVICE_USER}
ENV_FILE=${ENV_FILE:-/etc/default/$SERVICE_NAME}
AUTO_CREATE_USER=${AUTO_CREATE_USER:-1}
RELEASE_TAG=${RELEASE_TAG:-latest}

TMP_ROOT=${TMPDIR:-/tmp}
WORK_DIR=$(mktemp -d "$TMP_ROOT/${APP_NAME}-update-XXXXXX")
ARCHIVE_PATH="$WORK_DIR/${APP_NAME}-linux-amd64.tar.gz"

cleanup() {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT INT TERM

if [ "$RELEASE_TAG" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/${APP_NAME}-linux-amd64.tar.gz"
else
  DOWNLOAD_URL="https://github.com/$REPO/releases/download/$RELEASE_TAG/${APP_NAME}-linux-amd64.tar.gz"
fi

download() {
  if command -v curl >/dev/null 2>&1; then
    curl -L -o "$ARCHIVE_PATH" "$DOWNLOAD_URL"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -O "$ARCHIVE_PATH" "$DOWNLOAD_URL"
    return
  fi

  echo "Neither curl nor wget is installed."
  exit 1
}

download
tar -xzf "$ARCHIVE_PATH" -C "$WORK_DIR"

PACKAGE_DIR=$(find "$WORK_DIR" -mindepth 1 -maxdepth 1 -type d -name "${APP_NAME}-linux-amd64" | head -n 1)
if [ -z "$PACKAGE_DIR" ]; then
  echo "Extracted package directory not found."
  exit 1
fi

if [ ! -f "$INSTALL_DIR/config.json" ] && [ -f "$PACKAGE_DIR/config.example.json" ]; then
  cp "$PACKAGE_DIR/config.example.json" "$PACKAGE_DIR/config.json"
fi

if [ "$(id -u)" -eq 0 ]; then
  exec env \
    APP_NAME="$APP_NAME" \
    INSTALL_DIR="$INSTALL_DIR" \
    SERVICE_NAME="$SERVICE_NAME" \
    SERVICE_USER="$SERVICE_USER" \
    SERVICE_GROUP="$SERVICE_GROUP" \
    ENV_FILE="$ENV_FILE" \
    AUTO_CREATE_USER="$AUTO_CREATE_USER" \
    sh "$PACKAGE_DIR/scripts/install-service.sh"
fi

if command -v sudo >/dev/null 2>&1; then
  exec sudo env \
    APP_NAME="$APP_NAME" \
    INSTALL_DIR="$INSTALL_DIR" \
    SERVICE_NAME="$SERVICE_NAME" \
    SERVICE_USER="$SERVICE_USER" \
    SERVICE_GROUP="$SERVICE_GROUP" \
    ENV_FILE="$ENV_FILE" \
    AUTO_CREATE_USER="$AUTO_CREATE_USER" \
    sh "$PACKAGE_DIR/scripts/install-service.sh"
fi

echo "Root privileges are required. Run as root or install sudo."
exit 1
