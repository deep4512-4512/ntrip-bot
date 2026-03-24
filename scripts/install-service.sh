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
SERVICE_USER=${SERVICE_USER:-ntrip}
SERVICE_GROUP=${SERVICE_GROUP:-$SERVICE_USER}
ENV_FILE=${ENV_FILE:-/etc/default/$SERVICE_NAME}
AUTO_CREATE_USER=${AUTO_CREATE_USER:-1}

mkdir -p "$INSTALL_DIR"

if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
  if [ "$AUTO_CREATE_USER" = "1" ]; then
    groupadd --system "$SERVICE_GROUP"
  else
    echo "Group $SERVICE_GROUP does not exist."
    exit 1
  fi
fi

if ! id "$SERVICE_USER" >/dev/null 2>&1; then
  if [ "$AUTO_CREATE_USER" = "1" ]; then
    useradd --system --home-dir "$INSTALL_DIR" --shell /usr/sbin/nologin --gid "$SERVICE_GROUP" "$SERVICE_USER"
  else
    echo "User $SERVICE_USER does not exist."
    exit 1
  fi
fi

install -m 0755 "$PACKAGE_DIR/$APP_NAME" "$INSTALL_DIR/$APP_NAME"
if [ -f "$PACKAGE_DIR/update.sh" ]; then
  install -m 0755 "$PACKAGE_DIR/update.sh" "$INSTALL_DIR/update.sh"
fi

if [ -f "$PACKAGE_DIR/bot_settings.json" ] && [ ! -f "$INSTALL_DIR/bot_settings.json" ]; then
  install -m 0644 "$PACKAGE_DIR/bot_settings.json" "$INSTALL_DIR/bot_settings.json"
fi

if [ -f "$PACKAGE_DIR/config.json" ] && [ ! -f "$INSTALL_DIR/config.json" ]; then
  install -m 0600 "$PACKAGE_DIR/config.json" "$INSTALL_DIR/config.json"
fi

if [ ! -f "$ENV_FILE" ]; then
  mkdir -p "$(dirname "$ENV_FILE")"
  cat > "$ENV_FILE" <<EOF
# Optional environment overrides for $SERVICE_NAME
# TZ=UTC
# GOTRACEBACK=single
EOF
  chmod 0644 "$ENV_FILE"
fi

chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"

sed \
  -e "s|__APP_NAME__|$APP_NAME|g" \
  -e "s|__INSTALL_DIR__|$INSTALL_DIR|g" \
  -e "s|__ENV_FILE__|$ENV_FILE|g" \
  -e "s|__SERVICE_NAME__|$SERVICE_NAME|g" \
  -e "s|__SERVICE_USER__|$SERVICE_USER|g" \
  -e "s|__SERVICE_GROUP__|$SERVICE_GROUP|g" \
  "$PACKAGE_DIR/deploy/ntrip-bot.service" > "/etc/systemd/system/${SERVICE_NAME}.service"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"
systemctl status "$SERVICE_NAME" --no-pager
