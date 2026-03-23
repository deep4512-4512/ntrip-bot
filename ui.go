package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func formatMountState(state MountState) string {
	var b strings.Builder
	totalSats := 0
	for _, sats := range state.Data {
		totalSats += len(sats)
	}

	b.WriteString("================================\n")
	b.WriteString("Mount: " + state.Name + "\n")

	if state.Connected {
		b.WriteString("Status: " + onlineEmoji() + " Online\n")
	} else {
		b.WriteString("Status: " + offlineEmoji() + " Offline\n")
	}
	if state.LastError != "" {
		b.WriteString("Error: " + state.LastError + "\n")
	}
	if !state.UpdatedAt.IsZero() {
		b.WriteString("Last packet: " + state.UpdatedAt.Format("15:04:05") + "\n")
	} else {
		b.WriteString("Last packet: waiting for data\n")
	}
	b.WriteString("RTCM messages: " + fmt.Sprint(state.MsgCount) + "\n")
	b.WriteString("Visible satellites: " + fmt.Sprint(totalSats) + " " + satelliteEmoji() + "\n")

	var systems []string
	for sys := range state.Data {
		systems = append(systems, sys)
	}
	sort.Strings(systems)

	if len(systems) == 0 {
		b.WriteString("\nNo satellite data yet\n")
		return b.String()
	}

	for _, sys := range systems {
		sats := state.Data[sys]
		b.WriteString("\n" + systemEmoji(sys) + " " + sys + ": " + fmt.Sprint(len(sats)) + " " + satelliteEmoji() + "\n")

		var keys []int
		for k := range sats {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		if len(keys) > 0 {
			var items []string
			for _, k := range keys {
				items = append(items, fmt.Sprintf("%02d", k))
			}
			b.WriteString("  " + strings.Join(items, " ") + "\n")
		}
	}

	return b.String()
}

func systemEmoji(sys string) string {
	switch sys {
	case "GPS":
		return "\U0001F1FA\U0001F1F8"
	case "GLO":
		return "\U0001F1F7\U0001F1FA"
	case "GAL":
		return "\U0001F1EA\U0001F1FA"
	case "BDS":
		return "\U0001F1E8\U0001F1F3"
	default:
		return satelliteEmoji()
	}
}

func satelliteEmoji() string {
	return "\U0001F6F0\uFE0F"
}

func onlineEmoji() string {
	return "\U0001F7E2"
}

func offlineEmoji() string {
	return "\U0001F534"
}

func settingsEmoji() string {
	return "\u2699"
}

func dashboardEmoji() string {
	return "\U0001F4E1"
}

func statusEmoji() string {
	return "\u2139"
}

func stopEmoji() string {
	return "\u23F9"
}

func addEmoji() string {
	return "\u2795"
}

func backEmoji() string {
	return "\u21A9"
}

func hostEmoji() string {
	return "\U0001F310"
}

func mountEmoji() string {
	return "\U0001F4E1"
}

func homeEmoji() string {
	return "\U0001F3E0"
}

func cancelEmoji() string {
	return "\u2716"
}

func buildDashboardText(chatID int64) string {
	var out strings.Builder
	states := mountStateSnapshot(chatID)

	out.WriteString(dashboardEmoji() + " NTRIP Monitoring\n")
	out.WriteString("Updated: " + time.Now().Format("15:04:05") + "\n")
	for _, state := range states {
		out.WriteString("\n" + formatMountState(state) + "\n")
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return dashboardEmoji() + " NTRIP Monitoring\n\nNo data yet"
	}
	return text
}

func sendMenu(bot *tgbotapi.BotAPI, id int64, text string) {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = keyboard()
	if _, err := bot.Send(msg); err != nil {
		logError("send menu: %v", err)
	}
}

func quickMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(homeEmoji() + " Menu"),
		),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func sendQuickMenuHint(bot *tgbotapi.BotAPI, id int64, text string) {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = quickMenuKeyboard()
	if _, err := bot.Send(msg); err != nil {
		logError("send quick menu: %v", err)
	}
}

func sendWithKeyboard(bot *tgbotapi.BotAPI, id int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = markup
	if _, err := bot.Send(msg); err != nil {
		logError("send with keyboard: %v", err)
	}
}

func settingsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(satelliteEmoji()+" Mounts", "settings_mounts"),
			tgbotapi.NewInlineKeyboardButtonData(backEmoji()+" Back", "menu"),
		),
	)
}

func mountListKeyboard(chatID int64) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, m := range mountSnapshot(chatID) {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(satelliteEmoji()+" "+m.Name, fmt.Sprintf("mount_edit:%d", i)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(backEmoji()+" Back", "settings"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func mountEditKeyboard(index int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(hostEmoji()+" Edit host", fmt.Sprintf("edit_host:%d", index)),
			tgbotapi.NewInlineKeyboardButtonData(mountEmoji()+" Edit mount", fmt.Sprintf("edit_mount:%d", index)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(backEmoji()+" Back", "settings_mounts"),
			tgbotapi.NewInlineKeyboardButtonData(homeEmoji()+" Menu", "menu"),
		),
	)
}

func pendingMountKeyboard(chatID int64) tgbotapi.InlineKeyboardMarkup {
	sessionMu.Lock()
	session, ok := userSessions[chatID]
	var selection *pendingMountSelection
	if ok {
		selection = session.PendingMountSelect
	}
	sessionMu.Unlock()

	if selection == nil {
		return settingsKeyboard()
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, mount := range selection.Mounts {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(mountEmoji()+" "+mount, fmt.Sprintf("add_mount_pick:%d", i)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(cancelEmoji()+" Cancel", "menu"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func settingsText() string {
	return "Settings\n\nChoose a mount point to edit host or mount path."
}

func mountDetailsText(chatID int64, index int) string {
	m, ok := mountAt(chatID, index)
	if !ok {
		return "Mount not found"
	}
	return fmt.Sprintf(
		"Mount settings\n\nName: %s\nHost: %s\nPort: %s\nMount: %s\nUser: %s",
		m.Name, m.Host, m.Port, m.Mount, m.User,
	)
}

func keyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(dashboardEmoji()+" Monitoring", "dash"),
			tgbotapi.NewInlineKeyboardButtonData(statusEmoji()+" Status", "status"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(stopEmoji()+" Stop", "stop"),
			tgbotapi.NewInlineKeyboardButtonData(addEmoji()+" Add mount", "add"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(settingsEmoji()+" Settings", "settings"),
		),
	)
}
