package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

type MountState struct {
	ChatID    int64
	Name      string
	Data      map[string]map[int]float64
	Connected bool
	LastError string
	UpdatedAt time.Time
	MsgCount  uint64
}

type mountWorker struct {
	chatID int64
	cfg    MountConfig
	mu     sync.RWMutex
	state  MountState
	stop   chan struct{}
	conn   net.Conn
}

var (
	streamMu      sync.Mutex
	streamWorkers = map[string]*mountWorker{}
)

func mountKey(m MountConfig) string {
	return strings.Join([]string{m.Name, m.Host, m.Port, m.User, m.Mount}, "|")
}

func workerKey(chatID int64, m MountConfig) string {
	return fmt.Sprintf("%d|%s", chatID, mountKey(m))
}

func cloneData(src map[string]map[int]float64) map[string]map[int]float64 {
	dst := make(map[string]map[int]float64, len(src))
	for sys, sats := range src {
		dst[sys] = make(map[int]float64, len(sats))
		for sat, snr := range sats {
			dst[sys][sat] = snr
		}
	}
	return dst
}

func (w *mountWorker) snapshot() MountState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return MountState{
		ChatID:    w.state.ChatID,
		Name:      w.state.Name,
		Data:      cloneData(w.state.Data),
		Connected: w.state.Connected,
		LastError: w.state.LastError,
		UpdatedAt: w.state.UpdatedAt,
		MsgCount:  w.state.MsgCount,
	}
}

func (w *mountWorker) setConnected() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.state.Name = w.cfg.Name
	w.state.ChatID = w.chatID
	w.state.Connected = true
	w.state.LastError = ""
}

func (w *mountWorker) setDisconnected(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.conn = nil
	w.state.Name = w.cfg.Name
	w.state.ChatID = w.chatID
	w.state.Connected = false
	if err != nil {
		w.state.LastError = err.Error()
	}
}

func (w *mountWorker) setConn(conn net.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.conn = conn
}

func (w *mountWorker) closeConn() {
	w.mu.Lock()
	conn := w.conn
	w.conn = nil
	w.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (w *mountWorker) updateData(sys string, sats []int, snr []float64) uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state.Data == nil {
		w.state.Data = map[string]map[int]float64{}
	}
	if _, ok := w.state.Data[sys]; !ok {
		w.state.Data[sys] = map[int]float64{}
	}
	for i, sat := range sats {
		val := 0.0
		if snr != nil && i < len(snr) {
			val = snr[i]
		}
		w.state.Data[sys][sat] = val
	}
	w.state.Name = w.cfg.Name
	w.state.ChatID = w.chatID
	w.state.Connected = true
	w.state.LastError = ""
	w.state.UpdatedAt = time.Now()
	w.state.MsgCount++
	return w.state.MsgCount
}

func ensureMountStreams() {
	for _, chatID := range userIDs() {
		for _, m := range mountSnapshot(chatID) {
			startMountStream(chatID, m)
		}
	}
}

func startMountStream(chatID int64, m MountConfig) {
	key := workerKey(chatID, m)

	streamMu.Lock()
	if _, exists := streamWorkers[key]; exists {
		streamMu.Unlock()
		return
	}
	worker := &mountWorker{
		chatID: chatID,
		cfg:    m,
		state: MountState{
			ChatID: chatID,
			Name:   m.Name,
			Data:   map[string]map[int]float64{},
		},
		stop: make(chan struct{}),
	}
	streamWorkers[key] = worker
	streamMu.Unlock()
	logInfo("starting mount stream: chat_id=%d name=%s host=%s port=%s mount=%s", chatID, m.Name, m.Host, m.Port, m.Mount)

	go worker.run()
}

func reloadMountStreams(chatID int64) {
	logInfo("reloading mount streams: chat_id=%d", chatID)
	stopMountStreams(chatID)
	for _, m := range mountSnapshot(chatID) {
		startMountStream(chatID, m)
	}
}

func stopMountStreams(chatID int64) {
	streamMu.Lock()
	workers := make([]*mountWorker, 0)
	for key, worker := range streamWorkers {
		if worker.chatID == chatID {
			workers = append(workers, worker)
			delete(streamWorkers, key)
		}
	}
	streamMu.Unlock()
	logInfo("stopping mount streams: chat_id=%d count=%d", chatID, len(workers))

	for _, worker := range workers {
		close(worker.stop)
		worker.closeConn()
	}
}

func mountStateSnapshot(chatID int64) []MountState {
	streamMu.Lock()
	workers := make([]*mountWorker, 0, len(streamWorkers))
	for _, worker := range streamWorkers {
		if worker.chatID == chatID {
			workers = append(workers, worker)
		}
	}
	streamMu.Unlock()

	states := make([]MountState, 0, len(workers))
	for _, worker := range workers {
		states = append(states, worker.snapshot())
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})
	return states
}

func connectMount(m MountConfig) (net.Conn, *bufio.Reader, error) {
	logInfo("opening mount connection: name=%s host=%s port=%s mount=%s", m.Name, m.Host, m.Port, m.Mount)
	conn, err := net.DialTimeout("tcp", m.Host+":"+m.Port, 10*time.Second)
	if err != nil {
		return nil, nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(m.User + ":" + m.Password))
	req := fmt.Sprintf(
		"GET /%s HTTP/1.0\r\nHost: %s\r\nUser-Agent: NTRIP ntrip-bot/1.0\r\nAuthorization: Basic %s\r\n\r\n",
		m.Mount, m.Host, auth,
	)
	if _, err := fmt.Fprint(conn, req); err != nil {
		conn.Close()
		return nil, nil, err
	}

	r := bufio.NewReader(conn)
	statusLine, err := r.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if !strings.Contains(statusLine, "200") {
		conn.Close()
		return nil, nil, fmt.Errorf("unexpected status: %s", strings.TrimSpace(statusLine))
	}
	logInfo("mount accepted connection: name=%s status=%s", m.Name, strings.TrimSpace(statusLine))

	if err := consumeOptionalHeaders(r); err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, r, nil
}

func consumeOptionalHeaders(r *bufio.Reader) error {
	for {
		b, err := r.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if b[0] == 0xD3 || b[0] < 0x20 && b[0] != '\r' && b[0] != '\n' && b[0] != '\t' {
			return nil
		}

		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if line == "\r\n" || strings.TrimSpace(line) == "" {
			return nil
		}
	}
}

func fetchSourceTable(host, port, user, password string) ([]string, error) {
	logInfo("fetching sourcetable: host=%s port=%s user=%s", host, port, user)
	conn, err := net.DialTimeout("tcp", host+":"+port, 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return nil, err
	}

	var req strings.Builder
	req.WriteString("GET / HTTP/1.0\r\n")
	req.WriteString("Host: " + host + "\r\n")
	req.WriteString("User-Agent: NTRIP ntrip-bot/1.0\r\n")
	if user != "" || password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
		req.WriteString("Authorization: Basic " + auth + "\r\n")
	}
	req.WriteString("\r\n")

	if _, err := fmt.Fprint(conn, req.String()); err != nil {
		return nil, err
	}

	r := bufio.NewReader(conn)
	statusLine, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	statusLine = strings.TrimSpace(statusLine)
	if !strings.Contains(statusLine, "200") && !strings.Contains(statusLine, "SOURCETABLE") {
		return nil, fmt.Errorf("unexpected response: %s", statusLine)
	}

	var mounts []string
	seen := map[string]struct{}{}
	for {
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STR;") {
			parts := strings.Split(line, ";")
			if len(parts) > 1 {
				mount := strings.TrimSpace(parts[1])
				if mount != "" {
					if _, ok := seen[mount]; !ok {
						seen[mount] = struct{}{}
						mounts = append(mounts, mount)
					}
				}
			}
		}
		if line == "ENDSOURCETABLE" || errors.Is(err, io.EOF) {
			break
		}
	}

	sort.Strings(mounts)
	if len(mounts) == 0 {
		return nil, fmt.Errorf("no mount points found")
	}
	logInfo("sourcetable loaded: host=%s port=%s mounts=%d", host, port, len(mounts))
	return mounts, nil
}

func (w *mountWorker) run() {
	idleTimeout := time.Duration(w.cfg.Timeout) * time.Second
	if idleTimeout <= 0 {
		idleTimeout = 30 * time.Second
	}

	for {
		select {
		case <-w.stop:
			return
		default:
		}

		logInfo("connecting mount %s (%s:%s/%s)", w.cfg.Name, w.cfg.Host, w.cfg.Port, w.cfg.Mount)
		conn, reader, err := connectMount(w.cfg)
		if err != nil {
			logWarn("connect failed for mount %s: %v", w.cfg.Name, err)
			w.setDisconnected(err)
			select {
			case <-w.stop:
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		w.setConn(conn)
		w.setConnected()
		logInfo("stream connected for mount %s", w.cfg.Name)

		for {
			select {
			case <-w.stop:
				w.setDisconnected(nil)
				w.closeConn()
				return
			default:
			}

			if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
				w.setDisconnected(err)
				break
			}

			msg, sats, snr, err := readRTCM(reader)
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					w.setDisconnected(fmt.Errorf("read timeout"))
				} else {
					w.setDisconnected(err)
				}
				break
			}

			sys := system(msg)
			if sys == "" {
				continue
			}
			msgCount := w.updateData(sys, sats, snr)
			if msgCount == 1 || msgCount%100 == 0 {
				logInfo("rtcm processed: mount=%s msg=%d system=%s sats=%d total_messages=%d", w.cfg.Name, msg, sys, len(sats), msgCount)
			}
		}

		w.closeConn()
		logWarn("mount connection closed: name=%s", w.cfg.Name)
		select {
		case <-w.stop:
			return
		case <-time.After(3 * time.Second):
		}
	}
}
