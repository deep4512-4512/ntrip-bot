package main

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func startBot() {
	bot, err := tgbotapi.NewBotAPI(telegramToken())
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for up := range updates {
		if handleCallbackQuery(bot, up) {
			continue
		}

		handleMessage(bot, up)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, up tgbotapi.Update) bool {
	if up.CallbackQuery == nil {
		return false
	}

	id := up.CallbackQuery.Message.Chat.ID
	ensureUserRuntime(id)
	logInfo("callback received: chat_id=%d data=%s", id, up.CallbackQuery.Data)
	callback := tgbotapi.NewCallback(up.CallbackQuery.ID, "")
	if _, err := bot.Request(callback); err != nil {
		logError("answer callback: %v", err)
	}

	switch up.CallbackQuery.Data {
	case "dash":
		startDash(bot, id)
	case "status":
		sendMenu(bot, id, buildDashboardText(id))
	case "stop":
		stopDash(id)
		sendMenu(bot, id, "Monitoring stopped")
	case "add":
		beginAddMountFlow(id)
		sendMenu(bot, id, addMountInstructions())
	case "menu":
		resetUserFlowState(id)
		sendMenu(bot, id, "Ready")
	case "settings":
		openSettings(bot, id)
	case "settings_mounts":
		openMountSettings(bot, id)
	case "settings_monitoring_ttl":
		beginUserSettingEdit(bot, id, "monitoring_ttl")
	default:
		handleCallbackAction(bot, id, up.CallbackQuery.Data)
	}

	return true
}

func handleMessage(bot *tgbotapi.BotAPI, up tgbotapi.Update) {
	if up.Message == nil {
		return
	}

	id := up.Message.Chat.ID
	ensureUserRuntime(id)
	text := strings.TrimSpace(up.Message.Text)
	logInfo("message received: chat_id=%d text=%q", id, text)

	if handleEditMessage(bot, id, text) {
		return
	}
	if handleAddMountMessage(bot, id, text) {
		return
	}
	handleMenuMessage(bot, id, text)
}

func resetUserFlowState(id int64) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session := ensureUserSessionLocked(id)
	session.EditSession = nil
	session.AddMode = false
	session.PendingMountSelect = nil
	session.DeleteMountIndex = nil
}

func beginAddMountFlow(id int64) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session := ensureUserSessionLocked(id)
	session.AddMode = true
	session.EditSession = nil
	session.PendingMountSelect = nil
	session.DeleteMountIndex = nil
}

func openSettings(bot *tgbotapi.BotAPI, id int64) {
	sessionMu.Lock()
	session := ensureUserSessionLocked(id)
	session.EditSession = nil
	session.PendingMountSelect = nil
	session.DeleteMountIndex = nil
	sessionMu.Unlock()
	sendWithKeyboard(bot, id, userSettingsText(id), settingsKeyboard())
}

func openMountSettings(bot *tgbotapi.BotAPI, id int64) {
	sessionMu.Lock()
	session := ensureUserSessionLocked(id)
	session.EditSession = nil
	session.PendingMountSelect = nil
	session.DeleteMountIndex = nil
	sessionMu.Unlock()

	if mountCount(id) == 0 {
		sendWithKeyboard(bot, id, "No mount points configured yet.", settingsKeyboard())
		return
	}

	sendWithKeyboard(bot, id, "Choose a mount point:", mountListKeyboard(id))
}

func addMountInstructions() string {
	return "Send one of these formats:\n\nNAME HOST PORT\nfor mount list without auth\n\nNAME HOST PORT USER PASS\nfor mount list with auth\n\nNAME HOST PORT MOUNT\nfor manual add without auth\n\nNAME HOST PORT USER PASS MOUNT\nfor manual add with auth"
}

func parseCallbackIndex(data, prefix string) (int, bool) {
	indexText := strings.TrimPrefix(data, prefix)
	var index int
	if _, err := fmt.Sscanf(indexText, "%d", &index); err != nil {
		return 0, false
	}
	return index, true
}

func handleCallbackAction(bot *tgbotapi.BotAPI, id int64, data string) {
	switch {
	case strings.HasPrefix(data, "mount_edit:"):
		sessionMu.Lock()
		session := ensureUserSessionLocked(id)
		session.EditSession = nil
		session.DeleteMountIndex = nil
		sessionMu.Unlock()
		index, ok := parseCallbackIndex(data, "mount_edit:")
		if !ok {
			sendWithKeyboard(bot, id, "Invalid mount selection", settingsKeyboard())
			return
		}
		sendWithKeyboard(bot, id, mountDetailsText(id, index), mountEditKeyboard(index))
	case strings.HasPrefix(data, "edit_name:"):
		beginEditField(bot, id, data, "edit_name:", "name")
	case strings.HasPrefix(data, "edit_host:"):
		beginEditField(bot, id, data, "edit_host:", "host")
	case strings.HasPrefix(data, "edit_port:"):
		beginEditField(bot, id, data, "edit_port:", "port")
	case strings.HasPrefix(data, "edit_user:"):
		beginEditField(bot, id, data, "edit_user:", "user")
	case strings.HasPrefix(data, "edit_password:"):
		beginEditField(bot, id, data, "edit_password:", "password")
	case strings.HasPrefix(data, "edit_mount:"):
		beginEditField(bot, id, data, "edit_mount:", "mount")
	case strings.HasPrefix(data, "edit_timeout:"):
		beginEditField(bot, id, data, "edit_timeout:", "timeout")
	case strings.HasPrefix(data, "edit_min_sats:"):
		beginEditField(bot, id, data, "edit_min_sats:", "min_sats")
	case strings.HasPrefix(data, "delete_mount:"):
		index, ok := parseCallbackIndex(data, "delete_mount:")
		if !ok {
			sendWithKeyboard(bot, id, "Invalid mount selection", settingsKeyboard())
			return
		}
		sessionMu.Lock()
		session := ensureUserSessionLocked(id)
		session.EditSession = nil
		session.DeleteMountIndex = &index
		sessionMu.Unlock()
		sendWithKeyboard(bot, id, "Delete this mount point?", confirmDeleteKeyboard(index))
	case strings.HasPrefix(data, "cancel_delete:"):
		index, ok := parseCallbackIndex(data, "cancel_delete:")
		if !ok {
			sendWithKeyboard(bot, id, "Invalid mount selection", settingsKeyboard())
			return
		}
		sessionMu.Lock()
		ensureUserSessionLocked(id).DeleteMountIndex = nil
		sessionMu.Unlock()
		sendWithKeyboard(bot, id, mountDetailsText(id, index), mountEditKeyboard(index))
	case strings.HasPrefix(data, "confirm_delete:"):
		index, ok := parseCallbackIndex(data, "confirm_delete:")
		if !ok {
			sendWithKeyboard(bot, id, "Invalid mount selection", settingsKeyboard())
			return
		}
		sessionMu.Lock()
		session := ensureUserSessionLocked(id)
		deleteIndex := session.DeleteMountIndex
		session.DeleteMountIndex = nil
		sessionMu.Unlock()
		if deleteIndex == nil || *deleteIndex != index {
			sendWithKeyboard(bot, id, "Delete confirmation expired", settingsKeyboard())
			return
		}
		deleted, err := deleteMount(id, index)
		if err != nil {
			logError("delete mount: %v", err)
			sendWithKeyboard(bot, id, "Failed to delete mount", settingsKeyboard())
			return
		}
		reloadMountStreams(id)
		sendWithKeyboard(bot, id, "Mount deleted: "+deleted.Name, settingsKeyboard())
	case strings.HasPrefix(data, "add_mount_pick:"):
		index, ok := parseCallbackIndex(data, "add_mount_pick:")
		if !ok {
			sendMenu(bot, id, "Invalid mount selection")
			return
		}
		sessionMu.Lock()
		session := ensureUserSessionLocked(id)
		selection := session.PendingMountSelect
		sessionMu.Unlock()
		if selection == nil || index < 0 || index >= len(selection.Mounts) {
			sendMenu(bot, id, "Mount selection expired")
			return
		}

		cfg := MountConfig{
			Name:     selection.Name,
			Host:     selection.Host,
			Port:     selection.Port,
			User:     selection.User,
			Password: selection.Password,
			Mount:    selection.Mounts[index],
			Timeout:  5,
			MinSats:  10,
		}
		if err := addMount(id, cfg); err != nil {
			logError("save config: %v", err)
			sendMenu(bot, id, "Failed to save mount")
			return
		}

		sessionMu.Lock()
		session = ensureUserSessionLocked(id)
		session.PendingMountSelect = nil
		session.AddMode = false
		sessionMu.Unlock()
		startMountStream(id, cfg)
		logInfo("mount selected from sourcetable: chat_id=%d name=%s mount=%s", id, cfg.Name, cfg.Mount)
		sendMenu(bot, id, "Mount added: "+cfg.Mount)
	}
}

func beginEditField(bot *tgbotapi.BotAPI, id int64, data, prefix, field string) {
	index, ok := parseCallbackIndex(data, prefix)
	if !ok {
		sendWithKeyboard(bot, id, "Invalid mount selection", settingsKeyboard())
		return
	}
	sessionMu.Lock()
	session := ensureUserSessionLocked(id)
	session.EditSession = &editSession{MountIndex: index, Field: field}
	session.DeleteMountIndex = nil
	sessionMu.Unlock()
	sendWithKeyboard(bot, id, mountEditPrompt(field), mountEditKeyboard(index))
}

func beginUserSettingEdit(bot *tgbotapi.BotAPI, id int64, field string) {
	sessionMu.Lock()
	session := ensureUserSessionLocked(id)
	session.EditSession = &editSession{MountIndex: -1, Field: field}
	session.DeleteMountIndex = nil
	sessionMu.Unlock()
	sendWithKeyboard(bot, id, monitoringTimeoutPrompt(id), settingsKeyboard())
}

func normalizeMountFieldInput(field, text string) string {
	if (field == "user" || field == "password") && text == "-" {
		return ""
	}
	return text
}

func requiresNonEmptyMountField(field string) bool {
	switch field {
	case "user", "password":
		return false
	default:
		return true
	}
}

func handleEditMessage(bot *tgbotapi.BotAPI, id int64, text string) bool {
	sessionMu.Lock()
	userSession := ensureUserSessionLocked(id)
	editSession := userSession.EditSession
	sessionMu.Unlock()

	if editSession == nil {
		return false
	}

	if editSession.MountIndex < 0 {
		return handleUserSettingMessage(bot, id, editSession, text)
	}

	value := normalizeMountFieldInput(editSession.Field, text)
	if value == "" && requiresNonEmptyMountField(editSession.Field) {
		sendWithKeyboard(bot, id, "Value cannot be empty", mountEditKeyboard(editSession.MountIndex))
		return true
	}

	_, err := updateMountField(id, editSession.MountIndex, editSession.Field, value)
	if err != nil {
		logError("update mount field: %v", err)
		sendWithKeyboard(bot, id, "Failed to save settings: "+err.Error(), mountEditKeyboard(editSession.MountIndex))
		return true
	}

	sessionMu.Lock()
	ensureUserSessionLocked(id).EditSession = nil
	sessionMu.Unlock()
	reloadMountStreams(id)
	logInfo("mount field saved from telegram: chat_id=%d index=%d field=%s", id, editSession.MountIndex, editSession.Field)
	sendWithKeyboard(
		bot,
		id,
		mountDetailsText(id, editSession.MountIndex),
		mountEditKeyboard(editSession.MountIndex),
	)
	return true
}

func handleUserSettingMessage(bot *tgbotapi.BotAPI, id int64, editSession *editSession, text string) bool {
	switch editSession.Field {
	case "monitoring_ttl":
		applied, err := updateUserMonitoringTTL(id, text)
		if err != nil {
			logError("update user monitoring ttl: %v", err)
			sendWithKeyboard(bot, id, "Failed to save settings: "+err.Error(), settingsKeyboard())
			return true
		}
		sessionMu.Lock()
		ensureUserSessionLocked(id).EditSession = nil
		sessionMu.Unlock()
		msg := fmt.Sprintf("Monitoring timeout saved: %d min\nMaximum allowed: %d min", applied, botSettings.DashboardTTLMinutes)
		if strings.TrimSpace(text) == "0" {
			msg = fmt.Sprintf("Monitoring timeout reset to default: %d min", applied)
		}
		sendWithKeyboard(bot, id, msg, settingsKeyboard())
		return true
	default:
		sessionMu.Lock()
		ensureUserSessionLocked(id).EditSession = nil
		sessionMu.Unlock()
		sendWithKeyboard(bot, id, "Unsupported settings field", settingsKeyboard())
		return true
	}
}

func handleAddMountMessage(bot *tgbotapi.BotAPI, id int64, text string) bool {
	sessionMu.Lock()
	userSession := ensureUserSessionLocked(id)
	addMode := userSession.AddMode
	sessionMu.Unlock()

	if !addMode {
		return false
	}

	parts := strings.Fields(text)
	if len(parts) == 6 {
		logInfo("manual mount add requested: chat_id=%d name=%s host=%s port=%s mount=%s", id, parts[0], parts[1], parts[2], parts[5])
		cfg := MountConfig{
			Name:     parts[0],
			Host:     parts[1],
			Port:     parts[2],
			User:     parts[3],
			Password: parts[4],
			Mount:    parts[5],
			Timeout:  5,
			MinSats:  10,
		}
		if err := addMount(id, cfg); err != nil {
			logError("save config: %v", err)
			if _, sendErr := bot.Send(tgbotapi.NewMessage(id, "failed to save mount")); sendErr != nil {
				logError("send save error: %v", sendErr)
			}
			return true
		}

		startMountStream(id, cfg)
		resetUserFlowState(id)
		sendMenu(bot, id, "Mount added")
		return true
	}

	if len(parts) == 4 {
		logInfo("manual mount add without auth requested: chat_id=%d name=%s host=%s port=%s mount=%s", id, parts[0], parts[1], parts[2], parts[3])
		cfg := MountConfig{
			Name:     parts[0],
			Host:     parts[1],
			Port:     parts[2],
			User:     "",
			Password: "",
			Mount:    parts[3],
			Timeout:  5,
			MinSats:  10,
		}
		if err := addMount(id, cfg); err != nil {
			logError("save config: %v", err)
			if _, sendErr := bot.Send(tgbotapi.NewMessage(id, "Failed to save mount")); sendErr != nil {
				logError("send save error: %v", sendErr)
			}
			return true
		}

		startMountStream(id, cfg)
		resetUserFlowState(id)
		sendMenu(bot, id, "Mount added in online check mode")
		return true
	}

	if len(parts) == 5 {
		name, host, port, user, password := parts[0], parts[1], parts[2], parts[3], parts[4]
		logInfo("mount list requested: chat_id=%d name=%s host=%s port=%s", id, name, host, port)
		mounts, err := fetchSourceTable(host, port, user, password)
		if err != nil {
			logError("fetch sourcetable: %v", err)
			sendMenu(bot, id, "Failed to load mount list from host")
			return true
		}
		sessionMu.Lock()
		ensureUserSessionLocked(id).PendingMountSelect = &pendingMountSelection{
			Name:     name,
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			Mounts:   mounts,
		}
		sessionMu.Unlock()
		logInfo("mount list prepared for selection: chat_id=%d name=%s mounts=%d", id, name, len(mounts))
		sendWithKeyboard(bot, id, "Choose mount point for "+name+":", pendingMountKeyboard(id))
		return true
	}

	if len(parts) == 3 {
		name, host, port := parts[0], parts[1], parts[2]
		logInfo("mount list requested without auth: chat_id=%d name=%s host=%s port=%s", id, name, host, port)
		mounts, err := fetchSourceTable(host, port, "", "")
		if err != nil {
			logError("fetch sourcetable: %v", err)
			sendMenu(bot, id, "Failed to load mount list from host")
			return true
		}
		sessionMu.Lock()
		ensureUserSessionLocked(id).PendingMountSelect = &pendingMountSelection{
			Name:     name,
			Host:     host,
			Port:     port,
			User:     "",
			Password: "",
			Mounts:   mounts,
		}
		sessionMu.Unlock()
		logInfo("mount list prepared for unauthenticated selection: chat_id=%d name=%s mounts=%d", id, name, len(mounts))
		sendWithKeyboard(bot, id, "Choose mount point for "+name+":", pendingMountKeyboard(id))
		return true
	}

	sendMenu(bot, id, addMountInstructions())
	return true
}

func handleMenuMessage(bot *tgbotapi.BotAPI, id int64, text string) {
	if text == "/start" || text == "/menu" {
		sendQuickMenuHint(bot, id, "Quick access enabled")
		sendMenu(bot, id, "Ready")
	}

	if text == homeEmoji()+" Menu" || text == "Menu" {
		sendMenu(bot, id, "Ready")
	}
}
