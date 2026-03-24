# NTRIP Bot

Telegram bot for monitoring NTRIP mount points and reading RTCM streams.

## Features

- persistent NTRIP connections to configured mount points
- live `Monitoring` screen in Telegram
- one-time `Status` snapshot
- add mount points manually or from sourcetable
- per-user configuration by Telegram `chat_id`
- automatic stop for monitoring sessions and idle stream workers
- Linux deployment with install, update, and remove scripts

## Requirements

- Go 1.24+
- Telegram bot token
- access to an NTRIP caster

## Quick Start

Run locally from the project root:

```bash
go run .
```

If `config.json` does not exist, the bot asks for the Telegram token in the console and creates the config automatically.

## Linux Install

Public GitHub release download:

```bash
wget https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz
tar -xzf ntrip-bot-linux-amd64.tar.gz
cd ntrip-bot-linux-amd64
./install.sh
```

One-line install:

```bash
mkdir -p /tmp/ntrip-bot-install && cd /tmp/ntrip-bot-install && wget -O ntrip-bot-linux-amd64.tar.gz https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz && tar -xzf ntrip-bot-linux-amd64.tar.gz && cd ntrip-bot-linux-amd64 && ./install.sh
```

One-line install and check:

```bash
mkdir -p /tmp/ntrip-bot-install && cd /tmp/ntrip-bot-install && wget -O ntrip-bot-linux-amd64.tar.gz https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz && tar -xzf ntrip-bot-linux-amd64.tar.gz && cd ntrip-bot-linux-amd64 && ./install.sh && sudo systemctl status ntrip-bot --no-pager && sudo journalctl -u ntrip-bot -n 50 --no-pager
```

The installer:

- creates `config.json` interactively if needed
- installs the binary into `/opt/ntrip-bot`
- installs `update.sh` and `remove.sh`
- creates and enables the `systemd` service
- can create a dedicated system user and group automatically
- writes service output to `journalctl` and `/var/log/ntrip-bot/service.log`

Default installation path:

```text
/opt/ntrip-bot
```

Default environment file:

```text
/etc/default/ntrip-bot
```

Useful overrides:

- `APP_NAME`
- `INSTALL_DIR`
- `SERVICE_NAME`
- `SERVICE_USER`
- `SERVICE_GROUP`
- `ENV_FILE`
- `AUTO_CREATE_USER`

Example:

```bash
sudo INSTALL_DIR=/srv/ntrip-bot SERVICE_NAME=custom-ntrip SERVICE_USER=botuser SERVICE_GROUP=botuser ./install.sh
```

## Update On Server

Update to the latest release:

```bash
sudo /opt/ntrip-bot/update.sh
```

Update to a specific version:

```bash
sudo RELEASE_TAG=v1.0.4 /opt/ntrip-bot/update.sh
```

The updater downloads the release archive from GitHub, extracts it to a temporary directory, preserves the current `config.json`, installs the new binary, and restarts the service.

## Remove From Server

Remove only the service and executables, but keep config and data:

```bash
sudo /opt/ntrip-bot/remove.sh
```

Remove everything including install directory, environment file, and logs:

```bash
sudo REMOVE_DATA=1 /opt/ntrip-bot/remove.sh
```

## Service Checks

```bash
sudo systemctl status ntrip-bot --no-pager
sudo journalctl -u ntrip-bot -n 50 --no-pager
sudo journalctl -u ntrip-bot -f
tail -f /var/log/ntrip-bot/service.log
```

## Configuration

### `config.json`

Stores the bot token and per-user mount points.

A safe template is included in:

```text
config.example.json
```

Example:

```json
{
  "telegram_token": "123456:your_token_here",
  "users": {
    "123456789": {
      "mounts": [
        {
          "name": "Base1",
          "host": "caster.example.com",
          "port": "2101",
          "user": "user",
          "password": "password",
          "mount": "MOUNT1",
          "timeout": 5,
          "min_sats": 10
        }
      ]
    }
  }
}
```

### `bot_settings.json`

Global bot timing settings.

```json
{
  "dashboard_ttl_minutes": 5,
  "stream_idle_ttl_minutes": 10
}
```

- `dashboard_ttl_minutes` is the auto-stop time for `Monitoring`
- `stream_idle_ttl_minutes` is the idle timeout for background user streams

## Telegram Usage

After `/start` or `/menu`, the bot shows the main menu.

Buttons:

- `Monitoring` starts the live screen
- `Status` sends a one-time snapshot
- `Stop` stops `Monitoring`
- `Add mount` adds a new mount point
- `Settings` opens mount point editing

### Add Mount

Manual mode:

```text
NAME HOST PORT USER PASS MOUNT
```

Mount selection from host:

```text
NAME HOST PORT USER PASS
```

In the second case, the bot requests the sourcetable and shows found mount points as buttons.

## Development

Run locally:

```bash
go run .
```

Build locally:

```bash
go build -o ntrip-bot .
./ntrip-bot
```

With local Go caches:

```bash
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go run .
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go build -o ntrip-bot .
```

Makefile shortcuts:

```bash
make run
make build
make linux-build
make linux-package
make clean
```

Release package build:

```bash
sh ./scripts/build-linux.sh
sh ./scripts/package-linux.sh
```

## Logs

Application log file:

```text
bot.log
```

Logs include:

- config loading
- Telegram user actions
- sourcetable requests
- stream worker start and stop
- connection and RTCM read errors
- processed RTCM statistics

## Project Structure

- `ntrip-bot.go` entry point
- `config.go` config and settings loading
- `logging.go` logger and alert cooldown
- `rtcm.go` RTCM and MSM parsing
- `streams.go` NTRIP connections and stream workers
- `session.go` user runtime sessions and monitoring lifecycle
- `telegram.go` Telegram update handling
- `ui.go` dashboard text and Telegram UI helpers
- `install.sh` interactive Linux installer
- `update.sh` Linux updater
- `remove.sh` Linux removal script
- `scripts/` build and service helper scripts
- `deploy/ntrip-bot.service` `systemd` service template

## Git

Ignored by `.gitignore`:

- `.env`
- `config.json`
- `bot.log`
- `.gocache/`
- `.gomodcache/`
- `dist/`
- `*.exe`

This keeps tokens, local config, logs, caches, release artifacts, and local binaries out of the repository.
