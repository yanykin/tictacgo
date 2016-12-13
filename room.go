package main

import (
    "log"
    "time"
)

// Описание комнаты, внутри которой находятся игроки.
type gameRoom struct {
    *playerManager

    // Сама игра.
    *Game

    // Все подключенные соединения.
    connections map[*playerConnection]bool

    // Канал - "хэй, кто-то подключился к игре".
    joinGame chan *playerConnection
    // Канал - "ух, кто-то покинул игру".
    leaveGame chan *playerConnection
    // Канал - "смотрите, кто-то совершил ход".
    gameStateChanged chan bool
    // Канал - "кто-то выиграл партию!"
    someoneWonGame chan bool
}
// Оповестить всех игроков.
func (gr *gameRoom) Broadcast() {
    for k, _ := range gr.connections {
        k.sendStateToClient()
    }
}

// Функция прослушивает каналы на наличие сообщений.
func (gr *gameRoom) run() {
    for {
        select {
        // Хэй, кто-то подключился к игре
        case c := <- gr.joinGame:
            log.Printf( "New player on the server got symbol: %c.\n", c.pl.GetSymbol() )
            // Добавляем текущее соединение.
            gr.connections[c] = true
            gr.Broadcast()
        // Хэй, кто-то покинул игру
        case c := <- gr.leaveGame:
            log.Printf( "Player with symbol %c left the game.\n", c.pl.GetSymbol() )
            // Удаляем игрока из очереди.
            gr.RemovePlayer(c.pl)
            // Удаляем текущее соединение.
            delete(gr.connections, c)
        // Смотрите, кто-то совершил ход
        case <- gr.gameStateChanged:
            gr.Broadcast()
        // Ух, кто-то выиграл партию!
        case <- gr.someoneWonGame:
            gr.Broadcast()
            // Делаем задержку, чтобы все успели посмотреть, что партия закончилась.
            time.Sleep(5 * time.Second)
            // Создаём новую игру и снова оповещаем игроков.
            gr.Game.Reset()
            gr.Broadcast()
        }
    }
}

func NewGameRoom() *gameRoom {
    /*
    gr := new(gameRoom)
    gr.playerManager = NewPlayerManager()
    gr.Game = NewGame()
    */
    gr := &gameRoom{
        playerManager: NewPlayerManager(),
        connections: make(map[*playerConnection]bool),
        Game: NewGame(),
        joinGame: make(chan *playerConnection),
        leaveGame: make(chan *playerConnection),
        gameStateChanged: make(chan bool),
        someoneWonGame: make(chan bool),
    }

    // Запускаем обработчик сообщений.
    log.Printf( "Game room has been created.\n" )
    go gr.run()

    return gr
}