#!/usr/bin/env sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this script as root."
  exit 1
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PACKAGE_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

APP_NAME=${APP_NAME:-ntrip-bot}
INSTALL_DIR=${INSTALL_DIR:-/opt/ntrip-bot}
SERVICE_NAME=${SERVICE_NAME:-ntrip-bot}
SERVICE_USER=${SERVICE_USER:-root}
SERVICE_GROUP=${SERVICE_GROUP:-$SERVICE_USER}

mkdir -p "$INSTALL_DIR"

install -m 0755 "$PACKAGE_DIR/$APP_NAME" "$INSTALL_DIR/$APP_NAME"

if [ -f "$PACKAGE_DIR/bot_settings.json" ] && [ ! -f "$INSTALL_DIR/bot_settings.json" ]; then
  install -m 0644 "$PACKAGE_DIR/bot_settings.json" "$INSTALL_DIR/bot_settings.json"
fi

if [ -f "$PACKAGE_DIR/config.json" ] && [ ! -f "$INSTALL_DIR/config.json" ]; then
  install -m 0600 "$PACKAGE_DIR/config.json" "$INSTALL_DIR/config.json"
fi

chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"

sed \
  -e "s|__APP_NAME__|$APP_NAME|g" \
  -e "s|__INSTALL_DIR__|$INSTALL_DIR|g" \
  -e "s|__SERVICE_USER__|$SERVICE_USER|g" \
  -e "s|__SERVICE_GROUP__|$SERVICE_GROUP|g" \
  "$PACKAGE_DIR/deploy/ntrip-bot.service" > "/etc/systemd/system/${SERVICE_NAME}.service"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"
systemctl status "$SERVICE_NAME" --no-pager
