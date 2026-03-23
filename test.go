package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	server     = "pidt.net"
	port       = "1234"
	mountpoint = "4501"
	username   = "@deep4512"
	password   = "4745"
)

func main() {
	// Подключение к серверу
	conn, err := net.Dial("tcp", server+":"+port)
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	defer conn.Close()

	// Аутентификация
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	request := fmt.Sprintf("GET /%s HTTP/1.1\r\nHost: %s\r\nAuthorization: Basic %s\r\n\r\n",
		mountpoint, server, auth)

	conn.Write([]byte(request))

	// Чтение ответа
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		log.Fatal("Ошибка авторизации")
	}

	fmt.Println("Ответ сервера:", string(buf[:n]))
	if !strings.Contains(string(buf[:n]), "200") {
		log.Fatal("Ошибка подключения к потоку")
	}

	// Чтение RTCM данных
	fmt.Println("\nПолучение RTCM потока...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		buffer := make([]byte, 4096)
		count := 0
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				log.Println("Ошибка чтения:", err)
				return
			}
			if n > 0 {
				count++
				data := buffer[:n]
				fmt.Printf("\n--- Пакет #%d (%d байт) ---\n", count, n)
				fmt.Println(hex.Dump(data))
			}
		}
	}()

	<-sigChan
	fmt.Println("\nЗавершение работы")
}
