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
make clean
```

They use local `.gocache` and `.gomodcache` directories inside the project.

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
