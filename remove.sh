#!/usr/bin/env sh
set -eu

APP_NAME=${APP_NAME:-ntrip-bot}
INSTALL_DIR=${INSTALL_DIR:-/opt/ntrip-bot}
SERVICE_NAME=${SERVICE_NAME:-ntrip-bot}
ENV_FILE=${ENV_FILE:-/etc/default/$SERVICE_NAME}
REMOVE_DATA=${REMOVE_DATA:-0}

run_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
    return
  fi

  echo "Root privileges are required. Run as root or install sudo."
  exit 1
}

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

run_root systemctl stop "$SERVICE_NAME" 2>/dev/null || true
run_root systemctl disable "$SERVICE_NAME" 2>/dev/null || true

if [ -f "$SERVICE_FILE" ]; then
  run_root rm -f "$SERVICE_FILE"
fi

run_root systemctl daemon-reload

if [ "$REMOVE_DATA" = "1" ]; then
  run_root rm -rf "$INSTALL_DIR"
  run_root rm -f "$ENV_FILE"
  run_root rm -rf "/var/log/$SERVICE_NAME"
  echo "Removed service, install directory, environment file, and logs."
else
  if [ -f "$INSTALL_DIR/$APP_NAME" ]; then
    run_root rm -f "$INSTALL_DIR/$APP_NAME"
  fi
  if [ -f "$INSTALL_DIR/update.sh" ]; then
    run_root rm -f "$INSTALL_DIR/update.sh"
  fi
  if [ -f "$INSTALL_DIR/remove.sh" ]; then
    run_root rm -f "$INSTALL_DIR/remove.sh"
  fi
  echo "Removed service files. Config and data in $INSTALL_DIR were preserved."
fi
