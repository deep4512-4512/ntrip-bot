package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MountConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Mount    string `json:"mount"`
	Timeout  int    `json:"timeout"`
	MinSats  int    `json:"min_sats"`
}

type Config struct {
	TelegramToken string                `json:"telegram_token"`
	Users         map[string]UserConfig `json:"users"`
	Mounts        []MountConfig         `json:"mounts,omitempty"`
}

type UserConfig struct {
	Mounts               []MountConfig `json:"mounts"`
	MonitoringTTLMinutes int           `json:"monitoring_ttl_minutes,omitempty"`
}

type BotSettings struct {
	DashboardTTLMinutes  int `json:"dashboard_ttl_minutes"`
	StreamIdleTTLMinutes int `json:"stream_idle_ttl_minutes"`
}

var (
	cfg         Config
	cfgMu       sync.RWMutex
	botSettings BotSettings
)

func defaultBotSettings() BotSettings {
	return BotSettings{
		DashboardTTLMinutes:  5,
		StreamIdleTTLMinutes: 10,
	}
}

func normalizeUserConfig(userCfg UserConfig) UserConfig {
	if userCfg.Mounts == nil {
		userCfg.Mounts = []MountConfig{}
	}
	if userCfg.MonitoringTTLMinutes < 0 {
		userCfg.MonitoringTTLMinutes = 0
	}
	return userCfg
}

func loadConfig() {
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		logInfo("config.json not found, starting interactive setup")
		cfgMu.Lock()
		cfg = Config{TelegramToken: promptTelegramToken()}
		cfgMu.Unlock()
		if err := saveConfig(); err != nil {
			log.Fatal(err)
		}
		logInfo("config.json created")
	}

	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var nextCfg Config
	if err := json.Unmarshal(data, &nextCfg); err != nil {
		log.Fatal(err)
	}

	if nextCfg.TelegramToken == "" || nextCfg.TelegramToken == "PUT_YOUR_TOKEN_HERE" {
		log.Fatal("invalid token")
	}

	if nextCfg.Users == nil {
		nextCfg.Users = map[string]UserConfig{}
	}
	for key, userCfg := range nextCfg.Users {
		nextCfg.Users[key] = normalizeUserConfig(userCfg)
	}

	cfgMu.Lock()
	cfg = nextCfg
	cfgMu.Unlock()
	logInfo("config loaded: users=%d", len(nextCfg.Users))
}

func promptTelegramToken() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter Telegram bot token: ")
		token, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			log.Fatalf("read telegram token: %v", err)
		}

		token = strings.TrimSpace(token)
		if token == "" {
			fmt.Println("Token cannot be empty.")
			continue
		}

		if !looksLikeTelegramToken(token) {
			fmt.Println("Token format looks invalid. Expected something like 123456:ABCDEF...")
			continue
		}

		return token
	}
}

func looksLikeTelegramToken(token string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		return false
	}
	return len(parts[1]) >= 10
}

func saveConfig() error {
	cfgMu.RLock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	cfgMu.RUnlock()
	if err != nil {
		logError("save config marshal failed: %v", err)
		return err
	}
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		logError("save config write failed: %v", err)
		return err
	}
	logInfo("config saved")
	return nil
}

func loadBotSettings() {
	const settingsFile = "bot_settings.json"

	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		botSettings = defaultBotSettings()
		if err := saveBotSettings(); err != nil {
			log.Fatal(err)
		}
		logInfo("bot_settings.json created with defaults")
		return
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		log.Fatal(err)
	}

	settings := defaultBotSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Fatal(err)
	}

	if settings.DashboardTTLMinutes <= 0 {
		settings.DashboardTTLMinutes = defaultBotSettings().DashboardTTLMinutes
	}
	if settings.StreamIdleTTLMinutes <= 0 {
		settings.StreamIdleTTLMinutes = defaultBotSettings().StreamIdleTTLMinutes
	}

	botSettings = settings
	logInfo(
		"bot settings loaded: dashboard_ttl_minutes=%d stream_idle_ttl_minutes=%d",
		botSettings.DashboardTTLMinutes,
		botSettings.StreamIdleTTLMinutes,
	)
}

func saveBotSettings() error {
	data, err := json.MarshalIndent(botSettings, "", "  ")
	if err != nil {
		logError("save bot settings marshal failed: %v", err)
		return err
	}
	if err := os.WriteFile("bot_settings.json", data, 0644); err != nil {
		logError("save bot settings write failed: %v", err)
		return err
	}
	return nil
}

func streamIdleTTL() time.Duration {
	return time.Duration(botSettings.StreamIdleTTLMinutes) * time.Minute
}

func userKey(chatID int64) string {
	return strconv.FormatInt(chatID, 10)
}

func ensureUserConfigLocked(chatID int64) UserConfig {
	key := userKey(chatID)
	userCfg, ok := cfg.Users[key]
	if !ok {
		userCfg = UserConfig{}
		if len(cfg.Mounts) > 0 {
			userCfg.Mounts = append(userCfg.Mounts, cfg.Mounts...)
			logInfo("legacy mounts copied to user config: chat_id=%d mounts=%d", chatID, len(userCfg.Mounts))
		}
		cfg.Users[key] = normalizeUserConfig(userCfg)
		return cfg.Users[key]
	}
	return normalizeUserConfig(userCfg)
}

func ensureUserConfig(chatID int64) {
	cfgMu.Lock()
	beforeUsers := len(cfg.Users)
	beforeMounts := 0
	if userCfg, ok := cfg.Users[userKey(chatID)]; ok {
		beforeMounts = len(userCfg.Mounts)
	}
	userCfg := ensureUserConfigLocked(chatID)
	changed := len(cfg.Users) != beforeUsers || len(userCfg.Mounts) != beforeMounts
	if changed && len(cfg.Mounts) > 0 {
		cfg.Mounts = nil
	}
	cfgMu.Unlock()

	if changed {
		if err := saveConfig(); err != nil {
			logError("save config after user init failed: %v", err)
		}
	}
}

func ensureUserRuntime(chatID int64) {
	touchUserActivity(chatID)
	ensureUserConfig(chatID)
	for _, m := range userMountSnapshot(chatID) {
		startMountStream(chatID, m)
	}
}

func mountSnapshot(chatID int64) []MountConfig {
	ensureUserConfig(chatID)
	return userMountSnapshot(chatID)
}

func userMountSnapshot(chatID int64) []MountConfig {
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	userCfg, ok := cfg.Users[userKey(chatID)]
	if !ok {
		return nil
	}

	mounts := make([]MountConfig, len(userCfg.Mounts))
	copy(mounts, userCfg.Mounts)
	return mounts
}

func addMount(chatID int64, m MountConfig) error {
	cfgMu.Lock()
	userCfg := ensureUserConfigLocked(chatID)
	userCfg.Mounts = append(userCfg.Mounts, m)
	userCfg = normalizeUserConfig(userCfg)
	cfg.Users[userKey(chatID)] = userCfg
	cfgMu.Unlock()
	logInfo("mount added to config: chat_id=%d name=%s host=%s port=%s mount=%s", chatID, m.Name, m.Host, m.Port, m.Mount)

	return saveConfig()
}

func mountCount(chatID int64) int {
	ensureUserConfig(chatID)
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	userCfg, ok := cfg.Users[userKey(chatID)]
	if !ok {
		return 0
	}
	return len(userCfg.Mounts)
}

func mountAt(chatID int64, index int) (MountConfig, bool) {
	ensureUserConfig(chatID)
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	userCfg, ok := cfg.Users[userKey(chatID)]
	if !ok || index < 0 || index >= len(userCfg.Mounts) {
		return MountConfig{}, false
	}
	return userCfg.Mounts[index], true
}

func updateMountField(chatID int64, index int, field, value string) (MountConfig, error) {
	cfgMu.Lock()
	userCfg := ensureUserConfigLocked(chatID)
	if index < 0 || index >= len(userCfg.Mounts) {
		cfgMu.Unlock()
		return MountConfig{}, fmt.Errorf("mount index out of range")
	}

	switch field {
	case "name":
		userCfg.Mounts[index].Name = value
	case "host":
		userCfg.Mounts[index].Host = value
	case "port":
		userCfg.Mounts[index].Port = value
	case "user":
		userCfg.Mounts[index].User = value
	case "password":
		userCfg.Mounts[index].Password = value
	case "mount":
		userCfg.Mounts[index].Mount = value
	case "timeout":
		timeout, err := strconv.Atoi(value)
		if err != nil || timeout <= 0 {
			cfgMu.Unlock()
			return MountConfig{}, fmt.Errorf("timeout must be a positive integer")
		}
		userCfg.Mounts[index].Timeout = timeout
	case "min_sats":
		minSats, err := strconv.Atoi(value)
		if err != nil || minSats < 0 {
			cfgMu.Unlock()
			return MountConfig{}, fmt.Errorf("min_sats must be zero or a positive integer")
		}
		userCfg.Mounts[index].MinSats = minSats
	default:
		cfgMu.Unlock()
		return MountConfig{}, fmt.Errorf("unsupported field: %s", field)
	}

	updated := userCfg.Mounts[index]
	userCfg = normalizeUserConfig(userCfg)
	cfg.Users[userKey(chatID)] = userCfg
	cfgMu.Unlock()
	logInfo("mount updated: chat_id=%d index=%d field=%s value=%s name=%s", chatID, index, field, value, updated.Name)

	if err := saveConfig(); err != nil {
		return MountConfig{}, err
	}
	return updated, nil
}

func deleteMount(chatID int64, index int) (MountConfig, error) {
	cfgMu.Lock()
	userCfg := ensureUserConfigLocked(chatID)
	if index < 0 || index >= len(userCfg.Mounts) {
		cfgMu.Unlock()
		return MountConfig{}, fmt.Errorf("mount index out of range")
	}

	deleted := userCfg.Mounts[index]
	userCfg.Mounts = append(userCfg.Mounts[:index], userCfg.Mounts[index+1:]...)
	userCfg = normalizeUserConfig(userCfg)
	cfg.Users[userKey(chatID)] = userCfg
	cfgMu.Unlock()
	logInfo("mount deleted: chat_id=%d index=%d name=%s host=%s mount=%s", chatID, index, deleted.Name, deleted.Host, deleted.Mount)

	if err := saveConfig(); err != nil {
		return MountConfig{}, err
	}
	return deleted, nil
}

func userMonitoringTTLMinutes(chatID int64) int {
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	userCfg, ok := cfg.Users[userKey(chatID)]
	if !ok || userCfg.MonitoringTTLMinutes <= 0 {
		return botSettings.DashboardTTLMinutes
	}
	if userCfg.MonitoringTTLMinutes > botSettings.DashboardTTLMinutes {
		return botSettings.DashboardTTLMinutes
	}
	return userCfg.MonitoringTTLMinutes
}

func userMonitoringTTL(chatID int64) time.Duration {
	return time.Duration(userMonitoringTTLMinutes(chatID)) * time.Minute
}

func updateUserMonitoringTTL(chatID int64, value string) (int, error) {
	ttl, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("monitoring timeout must be a number in minutes")
	}
	if ttl < 0 {
		return 0, fmt.Errorf("monitoring timeout cannot be negative")
	}
	if ttl > botSettings.DashboardTTLMinutes {
		return 0, fmt.Errorf("maximum monitoring timeout is %d minutes", botSettings.DashboardTTLMinutes)
	}

	cfgMu.Lock()
	userCfg := ensureUserConfigLocked(chatID)
	userCfg.MonitoringTTLMinutes = ttl
	userCfg = normalizeUserConfig(userCfg)
	cfg.Users[userKey(chatID)] = userCfg
	cfgMu.Unlock()

	if err := saveConfig(); err != nil {
		return 0, err
	}

	applied := userMonitoringTTLMinutes(chatID)
	logInfo("user monitoring timeout updated: chat_id=%d requested=%d applied=%d", chatID, ttl, applied)
	return applied, nil
}

func monitoringTTLDescription(chatID int64) string {
	cfgMu.RLock()
	userCfg, ok := cfg.Users[userKey(chatID)]
	cfgMu.RUnlock()

	if !ok || userCfg.MonitoringTTLMinutes <= 0 {
		return fmt.Sprintf("default (%d min)", botSettings.DashboardTTLMinutes)
	}
	return fmt.Sprintf("%d min", userMonitoringTTLMinutes(chatID))
}

func telegramToken() string {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfg.TelegramToken
}

func userIDs() []int64 {
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	ids := make([]int64, 0, len(cfg.Users))
	for key := range cfg.Users {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
