package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type editSession struct {
	MountIndex int
	Field      string
}

type pendingMountSelection struct {
	Name     string
	Host     string
	Port     string
	User     string
	Password string
	Mounts   []string
}

type UserSessionState struct {
	AddMode            bool
	EditSession        *editSession
	PendingMountSelect *pendingMountSelection
	DashboardStop      chan struct{}
	LastActivity       time.Time
}

var (
	sessionMu    sync.Mutex
	userSessions = map[int64]*UserSessionState{}
)

func ensureUserSessionLocked(chatID int64) *UserSessionState {
	session, ok := userSessions[chatID]
	if !ok {
		session = &UserSessionState{}
		userSessions[chatID] = session
	}
	return session
}

func touchUserActivity(chatID int64) {
	sessionMu.Lock()
	ensureUserSessionLocked(chatID).LastActivity = time.Now()
	sessionMu.Unlock()
}

func lastUserActivity(chatID int64) time.Time {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session, ok := userSessions[chatID]
	if !ok {
		return time.Time{}
	}
	return session.LastActivity
}

func isDashboardActive(chatID int64) bool {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session, ok := userSessions[chatID]
	return ok && session.DashboardStop != nil
}

func startDashboardSession(chatID int64) chan struct{} {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	session := ensureUserSessionLocked(chatID)
	if session.DashboardStop != nil {
		close(session.DashboardStop)
	}
	session.DashboardStop = make(chan struct{})
	return session.DashboardStop
}

func clearDashboardSession(chatID int64) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if session, ok := userSessions[chatID]; ok {
		session.DashboardStop = nil
	}
}

func shouldKeepStreamsRunning(chatID int64) bool {
	if isDashboardActive(chatID) {
		return true
	}

	last := lastUserActivity(chatID)
	if last.IsZero() {
		return false
	}
	return time.Since(last) < streamIdleTTL()
}

func startIdleStreamReaper() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ids := userIDs()
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
			for _, chatID := range ids {
				if shouldKeepStreamsRunning(chatID) {
					continue
				}
				if len(mountStateSnapshot(chatID)) == 0 {
					continue
				}
				logInfo("stopping mount streams by idle ttl: chat_id=%d ttl=%s", chatID, streamIdleTTL())
				stopMountStreams(chatID)
			}
		}
	}()
}

func startDash(bot *tgbotapi.BotAPI, id int64) {
	touchUserActivity(id)
	stop := startDashboardSession(id)
	logInfo("dashboard started: chat_id=%d", id)

	msg, err := bot.Send(tgbotapi.NewMessage(id, "Loading..."))
	if err != nil {
		clearDashboardSession(id)
		logError("send loading message: %v", err)
		return
	}

	go func(messageID int) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		lastText := "Loading..."
		expiresAt := time.Now().Add(dashboardTTL())

		for {
			states := mountStateSnapshot(id)
			for _, state := range states {
				if !state.Connected && state.LastError != "" && canAlert(state.Name, 30*time.Second) {
					if _, sendErr := bot.Send(tgbotapi.NewMessage(id, "offline: "+state.Name)); sendErr != nil {
						logError("send alert: %v", sendErr)
					}
				}
			}

			text := buildDashboardText(id)
			if text != lastText {
				edit := tgbotapi.NewEditMessageText(id, messageID, text)
				if _, err := bot.Send(edit); err != nil {
					logError("edit dashboard message: %v", err)
				} else {
					lastText = text
				}
			}

			select {
			case <-stop:
				clearDashboardSession(id)
				logInfo("dashboard stopped: chat_id=%d", id)
				return
			case <-ticker.C:
				if time.Now().After(expiresAt) {
					stopDash(id)
					msg := fmt.Sprintf("Monitoring stopped automatically after %d minutes", botSettings.DashboardTTLMinutes)
					if _, err := bot.Send(tgbotapi.NewMessage(id, msg)); err != nil {
						logError("send monitoring ttl message: %v", err)
					}
					logInfo("dashboard stopped by ttl: chat_id=%d ttl=%s", id, dashboardTTL())
					return
				}
			}
		}
	}(msg.MessageID)
}

func stopDash(id int64) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if session, ok := userSessions[id]; ok && session.DashboardStop != nil {
		close(session.DashboardStop)
		session.DashboardStop = nil
		logInfo("dashboard stop requested: chat_id=%d", id)
	}
}
