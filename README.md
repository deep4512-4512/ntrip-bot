# NTRIP Bot

Telegram bot for monitoring NTRIP mount points and reading RTCM streams.

The bot:
- keeps persistent connections to mount points
- shows current state in Telegram
- can add mount points manually or from sourcetable
- stores mount settings per Telegram user
- automatically stops monitoring and idle stream workers by TTL

## Features

- `Monitoring` for a live auto-updating view
- `Status` for a one-time snapshot
- `Add mount` for adding a mount point
- `Settings` for editing `host` and `mount`
- sourcetable discovery from caster host
- per-user config by `chat_id`
- file logging to `bot.log`

## Requirements

- Go 1.24+
- Telegram bot token
- access to an NTRIP caster

## Quick Start

Linux is the primary target environment.

From the project root:

```bash
go run .
```

If `config.json` does not exist, the bot will ask for the Telegram token in the console and create the config automatically.

## Build

```bash
go build -o ntrip-bot .
./ntrip-bot
```

If you want local Go caches inside the project directory:

```bash
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go run .
```

Or for build:

```bash
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go build -o ntrip-bot .
```

## Makefile

Simple shortcuts are included:

```bash
make run
make build
make linux-build
make linux-package
make clean
```

They use local `.gocache` and `.gomodcache` directories inside the project.

## Linux Deployment Package

For Linux deployment, the repository now includes:

- `scripts/build-linux.sh` to build a Linux binary
- `scripts/run-linux.sh` to run the built binary
- `scripts/package-linux.sh` to create a release archive
- `scripts/install-service.sh` to install the binary and create a `systemd` service
- `deploy/ntrip-bot.service` as the service template

Build a Linux binary:

```bash
sh ./scripts/build-linux.sh
```

Create a release archive:

```bash
sh ./scripts/package-linux.sh
```

This creates a file like:

```text
dist/ntrip-bot-linux-amd64-YYYYMMDD-HHMMSS.tar.gz
```

## Download From GitHub Releases

Once a GitHub Release is published, the latest Linux package can be downloaded directly:

```bash
wget https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz
```

Or with `curl`:

```bash
curl -L -o ntrip-bot-linux-amd64.tar.gz https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz
```

Then install:

```bash
tar -xzf ntrip-bot-linux-amd64.tar.gz
cd ntrip-bot-linux-amd64
./install.sh
```

## One-Line Install

With `wget`:

```bash
mkdir -p /tmp/ntrip-bot-install && cd /tmp/ntrip-bot-install && wget -O ntrip-bot-linux-amd64.tar.gz https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz && tar -xzf ntrip-bot-linux-amd64.tar.gz && cd ntrip-bot-linux-amd64 && ./install.sh
```

With `curl`:

```bash
mkdir -p /tmp/ntrip-bot-install && cd /tmp/ntrip-bot-install && curl -L -o ntrip-bot-linux-amd64.tar.gz https://github.com/deep4512-4512/ntrip-bot/releases/latest/download/ntrip-bot-linux-amd64.tar.gz && tar -xzf ntrip-bot-linux-amd64.tar.gz && cd ntrip-bot-linux-amd64 && ./install.sh
```

After installation:

```bash
sudo systemctl status ntrip-bot --no-pager
sudo journalctl -u ntrip-bot -n 50 --no-pager
sudo journalctl -u ntrip-bot -f
```

## Install As a Service

On the target Linux host:

1. Unpack the archive
2. Run `./install.sh`
3. Answer the interactive questions if `config.json` does not exist yet
4. If needed, run `./install.sh` again after adjusting the config manually

Example:

```bash
tar -xzf ntrip-bot-linux-amd64-YYYYMMDD-HHMMSS.tar.gz
cd ntrip-bot-linux-amd64
./install.sh
```

Default installation path:

```text
/opt/ntrip-bot
```

The installer:
- copies the binary to `/opt/ntrip-bot`
- copies `config.example.json` into the release archive
- copies `bot_settings.json` if it does not exist yet
- copies `config.json` if it is included in the package directory
- can create `config.json` interactively from console input
- creates an optional environment file if it does not exist yet
- can create a system user and group automatically
- creates `/etc/systemd/system/ntrip-bot.service`
- enables and restarts the service
- writes service output to both `journalctl` and `/var/log/ntrip-bot/service.log`

You can override:
- `APP_NAME`
- `INSTALL_DIR`
- `SERVICE_NAME`
- `SERVICE_USER`
- `SERVICE_GROUP`
- `ENV_FILE`
- `AUTO_CREATE_USER`

Example:

```bash
sudo INSTALL_DIR=/srv/ntrip-bot SERVICE_NAME=custom-ntrip SERVICE_USER=botuser SERVICE_GROUP=botuser sh ./scripts/install-service.sh
```

Default environment file:

```text
/etc/default/ntrip-bot
```

Example overrides:

```bash
TZ=UTC
GOTRACEBACK=single
```

Service logs:

```bash
journalctl -u ntrip-bot -f
tail -f /var/log/ntrip-bot/service.log
```

## Windows

Windows still works, but it is now a secondary environment.

Run:

```powershell
go run .
```

Build:

```powershell
go build -o ntrip-bot.exe .
.\ntrip-bot.exe
```

If Windows causes problems with the default Go cache:

```powershell
$env:GOCACHE='c:\Users\deep\go\ntrip-bot\.gocache'
$env:GOMODCACHE='c:\Users\deep\go\ntrip-bot\.gomodcache'
go run .
```

## Configuration

### `config.json`

Created automatically. Stores the bot token and per-user mount points.

A safe template is included in:

```text
config.example.json
```

For Linux deployment, you can either:
- let `./install.sh` create `config.json` interactively
- or copy `config.example.json` to `config.json` and edit it manually

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

Stores global bot timing settings.

Current format:

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

## Logs

The bot writes logs to:

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

## Git

The following files are ignored by `.gitignore`:
- `config.json`
- `bot.log`
- `.gocache/`
- `.gomodcache/`
- `*.exe`

This prevents tokens, logs, local cache data, and build artifacts from being committed.
