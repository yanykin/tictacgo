package main

import (
	"html/template"
	"log"
	"net/http"
)

// Реализация протокола Websocket на языке Go.
import "github.com/gorilla/websocket"

const (
	HTTP_PORT_STRING = ":7777"
)

// Исходные размеры игрового поля
const FIELD_SIZE = 10

// Обработчик главной страницы.
func mainPageHandler(writer http.ResponseWriter, r *http.Request) {
    log.Printf("Getting main page...")
	// t := template.New("main_page_template")
	t, err := template.ParseFiles("templates/main_page.html")
    if err != nil {
        log.Fatalf("Error by generating template for main page: %s\n", err)
    }
	log.Printf("Host: %s\n", r.Host)
    data := struct {
		Host string
        FieldSize int
    }{
		Host: r.Host,
		FieldSize: FIELD_SIZE,
	}
	t.Execute(writer, data)
}

var wsUpgrader = websocket.Upgrader {
    ReadBufferSize:  2048,
    WriteBufferSize: 2048,
}

// Сама игровая комната, которая управляет доской и подключёнными игроками.
var currentGameRoom = NewGameRoom()

// Обработчик установления соединения по протоколу WebSocket.
func webSocketHandler(writer http.ResponseWriter, r *http.Request) {
    // Создаём/обновляем WebSocket-соединение
    wsConnection, err := wsUpgrader.Upgrade(writer, r, nil)
	if err != nil {
        log.Fatalf("WebSocket error: %s\n", err)
    }

	// Смотрим, есть ли в распоряжении свободные слоты.
	playerConnection := NewPlayerConnection(wsConnection, currentGameRoom)

	// Если удалось подключиться к игре, то отправляем сообщение в комнату.
	if playerConnection != nil {
		log.Printf("New WebSocket connection.\n")
		currentGameRoom.joinGame <- playerConnection
	}
}

// Сам главный код, запускаемый при инициализации игрового сервера.
func main() {
    log.Printf("Tic-Tac-Go game server is starting up...\n")

	http.HandleFunc("/", mainPageHandler)

	// Навешиваем обработчик на WebSocket-соединения.
	http.HandleFunc("/websocket", webSocketHandler)

	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	if err := http.ListenAndServe(HTTP_PORT_STRING, nil); err != nil {
		log.Fatal("Error by launching game server at ListenAndServe:", err)
	}
}
