#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CONFIG_FILE="$ROOT_DIR/config.json"
CONFIG_EXAMPLE="$ROOT_DIR/config.example.json"

prompt() {
  label=$1
  default_value=${2:-}
  if [ -n "$default_value" ]; then
    printf "%s [%s]: " "$label" "$default_value" >&2
  else
    printf "%s: " "$label" >&2
  fi
  IFS= read -r value || true
  if [ -z "$value" ]; then
    value=$default_value
  fi
  printf "%s" "$value"
}

prompt_secret() {
  label=$1
  printf "%s: " "$label" >&2
  stty -echo
  IFS= read -r value || true
  stty echo
  printf "\n" >&2
  printf "%s" "$value"
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

write_config_interactive() {
  echo "Interactive config setup"

  telegram_token=$(prompt "Telegram bot token")
  while [ -z "$telegram_token" ]; do
    echo "Telegram bot token is required."
    telegram_token=$(prompt "Telegram bot token")
  done
  telegram_token_json=$(json_escape "$telegram_token")

  configure_mount=$(prompt "Configure initial mount now? (y/N)" "N")

  if [ "$configure_mount" = "y" ] || [ "$configure_mount" = "Y" ]; then
    chat_id=$(prompt "Telegram chat id")
    while [ -z "$chat_id" ]; do
      echo "Telegram chat id is required when creating an initial mount."
      chat_id=$(prompt "Telegram chat id")
    done

    mount_name=$(prompt "Mount name" "Base1")
    mount_host=$(prompt "Caster host")
    mount_port=$(prompt "Caster port" "2101")
    mount_user=$(prompt "Caster user")
    mount_password=$(prompt_secret "Caster password")
    mount_path=$(prompt "Mount path")
    mount_timeout=$(prompt "Mount timeout seconds" "5")
    mount_min_sats=$(prompt "Minimum satellites" "10")
    chat_id_json=$(json_escape "$chat_id")
    mount_name_json=$(json_escape "$mount_name")
    mount_host_json=$(json_escape "$mount_host")
    mount_port_json=$(json_escape "$mount_port")
    mount_user_json=$(json_escape "$mount_user")
    mount_password_json=$(json_escape "$mount_password")
    mount_path_json=$(json_escape "$mount_path")

    cat > "$CONFIG_FILE" <<EOF
{
  "telegram_token": "$telegram_token_json",
  "users": {
    "$chat_id_json": {
      "mounts": [
        {
          "name": "$mount_name_json",
          "host": "$mount_host_json",
          "port": "$mount_port_json",
          "user": "$mount_user_json",
          "password": "$mount_password_json",
          "mount": "$mount_path_json",
          "timeout": $mount_timeout,
          "min_sats": $mount_min_sats
        }
      ]
    }
  }
}
EOF
  else
    cat > "$CONFIG_FILE" <<EOF
{
  "telegram_token": "$telegram_token_json",
  "users": {}
}
EOF
  fi

  chmod 0600 "$CONFIG_FILE" || true
  echo "Created $CONFIG_FILE"
}

needs_config_setup() {
  if [ ! -f "$CONFIG_FILE" ]; then
    return 0
  fi

  if grep -q "replace_with_real_token" "$CONFIG_FILE"; then
    return 0
  fi

  return 1
}

if needs_config_setup; then
  if [ -f "$CONFIG_EXAMPLE" ]; then
    echo "config.json is missing or still contains placeholder values."
    write_config_interactive
  else
    echo "config.json not found and config.example.json is missing."
    exit 1
  fi
fi

if [ "$(id -u)" -eq 0 ]; then
  exec sh "$ROOT_DIR/scripts/install-service.sh"
fi

if command -v sudo >/dev/null 2>&1; then
  exec sudo \
    APP_NAME="${APP_NAME:-ntrip-bot}" \
    INSTALL_DIR="${INSTALL_DIR:-/opt/ntrip-bot}" \
    SERVICE_NAME="${SERVICE_NAME:-ntrip-bot}" \
    SERVICE_USER="${SERVICE_USER:-ntrip}" \
    SERVICE_GROUP="${SERVICE_GROUP:-${SERVICE_USER:-ntrip}}" \
    ENV_FILE="${ENV_FILE:-/etc/default/${SERVICE_NAME:-ntrip-bot}}" \
    AUTO_CREATE_USER="${AUTO_CREATE_USER:-1}" \
    sh "$ROOT_DIR/scripts/install-service.sh"
fi

echo "Root privileges are required. Run as root or install sudo."
exit 1
