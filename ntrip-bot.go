package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ===== CONFIG (без .env) =====
const (
	TelegramToken = "8745411743:AAE7WuJ2sNpLyqlLRf2mFTcPZjvTz3zu-CU"

	NtripHost     = "pidt.net"
	NtripPort     = "1234"
	NtripUser     = "@deep4512"
	NtripPassword = "4745"
	NtripMount    = "4501"
)

// ===== BIT READER =====

type BitReader struct {
	data []byte
	pos  int
}

func (b *BitReader) ReadBits(n int) uint64 {
	var val uint64
	for i := 0; i < n; i++ {
		byteIndex := b.pos / 8
		bitIndex := 7 - (b.pos % 8)
		bit := (b.data[byteIndex] >> bitIndex) & 1
		val = (val << 1) | uint64(bit)
		b.pos++
	}
	return val
}

// ===== SIGNAL NAMES =====

func signalName(system string, sig int) string {
	switch system {
	case "GPS":
		switch sig {
		case 1:
			return "L1"
		case 3:
			return "L1C"
		case 5:
			return "L2"
		case 7:
			return "L5"
		}
	case "GLONASS":
		switch sig {
		case 1:
			return "G1"
		case 5:
			return "G2"
		}
	case "GALILEO":
		switch sig {
		case 1:
			return "E1"
		case 5:
			return "E5"
		case 7:
			return "E5b"
		}
	case "BEIDOU":
		switch sig {
		case 1:
			return "B1"
		case 5:
			return "B2"
		case 7:
			return "B3"
		}
	}
	return fmt.Sprintf("S%d", sig)
}

// ===== MSM PARSER =====

func parseMSM(payload []byte, msgType int) ([]int, []float64, []int) {
	br := &BitReader{data: payload}

	br.ReadBits(12)
	br.ReadBits(12)
	br.ReadBits(30)
	br.ReadBits(1)
	br.ReadBits(3)
	br.ReadBits(7)
	br.ReadBits(2)
	br.ReadBits(2)

	var sats []int
	for i := 0; i < 64; i++ {
		if br.ReadBits(1) == 1 {
			sats = append(sats, i+1)
		}
	}

	var signals []int
	for i := 0; i < 32; i++ {
		if br.ReadBits(1) == 1 {
			signals = append(signals, i+1)
		}
	}

	sigCount := len(signals)

	cellMask := make([]bool, len(sats)*sigCount)
	for i := range cellMask {
		cellMask[i] = br.ReadBits(1) == 1
	}

	for range sats {
		br.ReadBits(8)
		br.ReadBits(4)
		br.ReadBits(10)
		if msgType%10 >= 5 {
			br.ReadBits(14)
		}
	}

	if msgType%10 != 7 {
		return sats, nil, signals
	}

	var snrs []float64
	for _, active := range cellMask {
		if !active {
			continue
		}

		br.ReadBits(20)
		br.ReadBits(24)
		br.ReadBits(10)
		cnr := br.ReadBits(10)
		br.ReadBits(15)

		snrs = append(snrs, float64(cnr)*0.0625)
	}

	final := make([]float64, len(sats))
	counts := make([]int, len(sats))

	idx := 0
	for i := 0; i < len(sats); i++ {
		for j := 0; j < sigCount && idx < len(snrs); j++ {
			final[i] += snrs[idx]
			counts[i]++
			idx++
		}
	}

	for i := range final {
		if counts[i] > 0 {
			final[i] /= float64(counts[i])
		}
	}

	return sats, final, signals
}

// ===== RTCM =====

func readRTCMFrame(r *bufio.Reader) (int, []int, []float64, []int, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, nil, nil, nil, err
		}

		if b != 0xD3 {
			continue
		}

		header := make([]byte, 2)
		_, err = io.ReadFull(r, header)
		if err != nil {
			return 0, nil, nil, nil, err
		}

		length := int(header[0]&0x03)<<8 | int(header[1])

		payload := make([]byte, length)
		_, err = io.ReadFull(r, payload)
		if err != nil {
			return 0, nil, nil, nil, err
		}

		io.ReadFull(r, make([]byte, 3))

		msgType := int(payload[0])<<4 | int(payload[1]>>4)

		if msgType >= 1070 && msgType <= 1137 {
			sats, snrs, signals := parseMSM(payload, msgType)
			return msgType, sats, snrs, signals, nil
		}

		return msgType, nil, nil, nil, nil
	}
}

func detectSystem(msgType int) string {
	switch {
	case msgType >= 1070 && msgType < 1080:
		return "GPS"
	case msgType >= 1080 && msgType < 1090:
		return "GLONASS"
	case msgType >= 1090 && msgType < 1100:
		return "GALILEO"
	case msgType >= 1120 && msgType < 1130:
		return "BEIDOU"
	default:
		return ""
	}
}

// ===== UI =====

func systemEmoji(sys string) string {
	switch sys {
	case "GPS":
		return "🇺🇸🛰️"
	case "GLONASS":
		return "🇷🇺🛰️"
	case "GALILEO":
		return "🇪🇺🛰️"
	case "BEIDOU":
		return "🇨🇳🛰️"
	default:
		return "🛰️"
	}
}

// ===== NTRIP =====

func connectNTRIP() (string, error) {
	conn, err := net.DialTimeout("tcp", NtripHost+":"+NtripPort, 10*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	auth := base64.StdEncoding.EncodeToString([]byte(NtripUser + ":" + NtripPassword))

	req := fmt.Sprintf("GET /%s HTTP/1.0\r\nAuthorization: Basic %s\r\n\r\n", NtripMount, auth)
	conn.Write([]byte(req))

	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')

	if !strings.Contains(line, "200") {
		return "", fmt.Errorf("mount point unavailable")
	}

	seen := map[string]map[int]float64{
		"GPS": {}, "GLONASS": {}, "GALILEO": {}, "BEIDOU": {},
	}

	signalMap := map[string]map[string]struct{}{
		"GPS": {}, "GLONASS": {}, "GALILEO": {}, "BEIDOU": {},
	}

	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			goto DONE
		default:
			msgType, sats, snrs, signals, err := readRTCMFrame(reader)
			if err != nil {
				continue
			}

			sys := detectSystem(msgType)
			if sys == "" || sats == nil {
				continue
			}

			for _, sig := range signals {
				name := signalName(sys, sig)
				signalMap[sys][name] = struct{}{}
			}

			for i, s := range sats {
				if snrs != nil && i < len(snrs) {
					seen[sys][s] = snrs[i]
				} else {
					if _, ok := seen[sys][s]; !ok {
						seen[sys][s] = 0
					}
				}
			}
		}
	}

DONE:
	totalSats := 0

	for _, sats := range seen {
		totalSats += len(sats)
	}
	var result strings.Builder
	result.WriteString(fmt.Sprintf(
		"📡 *NTRIP STATUS*\n🛰️ Total Satellites: *%d*\n\n",
		totalSats,
	))

	for sys, sats := range seen {
		if len(sats) == 0 {
			continue
		}

		var sum float64
		var count int

		var satList []string
		for s := range sats {
			satList = append(satList, fmt.Sprintf("%d", s))
			if sats[s] > 0 {
				sum += sats[s]
				count++
			}
		}
		sort.Strings(satList)

		var sigList []string
		for s := range signalMap[sys] {
			sigList = append(sigList, s)
		}
		sort.Strings(sigList)

		emoji := systemEmoji(sys)

		if count > 0 {
			avg := sum / float64(count)
			result.WriteString(fmt.Sprintf(
				"%s *%s*\n🛰️ Satellites: `%d`\n📶 SNR: `%.1f dB`\n📡 Signals: `%s`\n🔢 PRN: `%s`\n\n",
				emoji, sys, len(sats), avg,
				strings.Join(sigList, ","),
				strings.Join(satList, ","),
			))
		} else {
			result.WriteString(fmt.Sprintf(
				"%s *%s*\n🛰️ Satellites: `%d`\n📡 Signals: `%s`\n🔢 PRN: `%s`\n\n",
				emoji, sys, len(sats),
				strings.Join(sigList, ","),
				strings.Join(satList, ","),
			))
		}
	}

	return result.String(), nil
}

// ===== TELEGRAM =====

func startBot() {
	var bot *tgbotapi.BotAPI
	var err error

	for i := 0; i < 5; i++ {
		bot, err = tgbotapi.NewBotAPI(TelegramToken)
		if err == nil {
			break
		}
		log.Println("Telegram error:", err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("Telegram init failed: %v", err)
	}

	log.Println("Bot started:", bot.Self.UserName)

	updates := bot.GetUpdatesChan(tgbotapi.NewUpdate(0))

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Command() == "status" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⏳ Checking NTRIP...")
			sent, _ := bot.Send(msg)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			ch := make(chan string, 1)

			go func() {
				res, err := connectNTRIP()
				if err != nil {
					ch <- "❌ " + err.Error()
					return
				}
				ch <- res
			}()

			select {
			case res := <-ch:
				edit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sent.MessageID, res)
				edit.ParseMode = "Markdown"
				bot.Send(edit)
			case <-ctx.Done():
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "⏱ Timeout"))
			}
		}
	}
}

func main() {
	startBot()
}
