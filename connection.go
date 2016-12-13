package main

// Информация о соединении игрока через WebSocket

import (
    "log"
	"github.com/gorilla/websocket"
    "sync"
)

// Информация о ходе, сериализуемая в JSON.
type moveInfo struct {
    Row, Column int
}

// Информация о том, что кто-то выиграл.
type winningInfo struct {
    // Флаг, что кто-то выиграл.
    IsWinner bool
    // Символ выигравшего игрока.
    Symbol rune
    // Информация о выигрыше по каждому из возможных направлений.
    Horizontal []moveInfo `json:",omitempty"`
    Vertical []moveInfo `json:",omitempty"`
    MainDiagonal []moveInfo `json:",omitempty"`
    SideDiagonal []moveInfo `json:",omitempty"`
}

// Информация о состоянии игры и игрока.
type gameInfo struct {
    CanMove bool
    ActivePlayers string
    Board *gameState

}

type playerConnection struct {
    // Расширяем тип данных "Игрок"
    pl *player
    // Само соединение.
    ws *websocket.Conn
    // Комната для общения.
    room *gameRoom
}

// Мьютексы для блокировки множественной записи.
var jsonMutex = sync.Mutex{}

// Прослушивание сообщений, пришедших через WebSocket от клиентского приложения.
func (pc *playerConnection) runReceiverFromClient() {
    // Цикл обработки сообщений.
    for {
        // Формально говоря, от клиента мы можем получить информацию о ходе.
        move := moveInfo{}
        err := pc.ws.ReadJSON(&move)
        if err != nil {
            pc.sendErrorToClient("Error reading JSON!")
            break
        }

        // Вполне возможно, что данные отослались не в его ход.
        if !pc.pl.CanMoveNow() {
            pc.sendErrorToClient("Please, wait your turn!")
        }
        // Совершаем ход.
        symbol := rune(pc.pl.GetSymbol())
        pc.room.Set(move.Row, move.Column, symbol)

        log.Printf("Player %c made move: (%d, %d)\n", symbol, move.Row, move.Column)

        // Проверяем, не выиграл ли игрок партию.
        pc.room.CheckWin(move.Row, move.Column)
        if pc.room.HasWinning() {
            log.Printf("Player %c has won the game!\n", symbol)
            names := []string{"horizontal", "vertical", "main diagonal", "side diagonal"}
            for index, value := range [...][]cell{pc.room.Horizontal, pc.room.Vertical, pc.room.MainDiagonal, pc.room.SideDiagonal} {
                var start, finish cell
                if value != nil {
                    length := len(value)
                    start = value[0]
                    finish = value[length-1]
                    log.Printf("\tWinning line: %s from %s to %s (%d elements)\n", names[index], start, finish, length)
                }
            }
            // Сообщаем остальным игрокам, что партия выиграна.
            pc.room.someoneWonGame <- true
        } else {
            // Меняем очерёдность.
            pc.room.MoveToNextPlayer()

            log.Printf("Players queue: %s\n", pc.room.GetQueueStatus())

            // Уведомляем комнату, что состояние игры изменилось.
            pc.room.gameStateChanged <- true
        }
    }
    // Как только цикл закончился - сообщаем комнате, что пользователь покинул игру.
    pc.room.leaveGame <- pc
    // Закрываем связанное соединение.
    pc.ws.Close()
}

// Отправка клиенту сообщения об ошибке
func (pc *playerConnection) sendErrorToClient(message string) error {
    jsonMutex.Lock()
    defer jsonMutex.Unlock()
    return pc.ws.WriteJSON(&struct {
        errorText string
    }{message})
}

// Отправка состояния игры (и игрока в частности) через WebSocket обратно в клиент.
func (pc *playerConnection) sendStateToClient() {
    go func() {
        // Отправляем их как JSON через WebSocket, блокируя мьютекс.
        jsonMutex.Lock()
        defer jsonMutex.Unlock()
        // Собираем данные об игре.
        currentGameInfo := gameInfo{
            CanMove: pc.pl.CanMoveNow(),
            ActivePlayers: pc.room.ActivePlayers(),
            Board: pc.room.GetState(),
        }
        err := pc.ws.WriteJSON(&currentGameInfo)
		// err := pc.ws.WriteMessage(websocket.TextMessage, []byte(msg))
        // Если что-то пошло не так - покидаем игру.
		if err != nil {
			pc.room.leaveGame <- pc
			pc.ws.Close()
		}
	}()
}

func NewPlayerConnection(ws *websocket.Conn, room *gameRoom) *playerConnection {
    // Сначала пытаемся создать игрока в комнате
    newPlayer := room.AddPlayer()
    if newPlayer == nil {
        log.Fatalf("No more available slots in the game.\n")
        return nil
    }
    pc := &playerConnection{newPlayer, ws, room}

    // Соединение создали, теперь надо научиться слушать сообщения.
    go pc.runReceiverFromClient()

    return pc
}