package main

import (
    "bytes"
    "sync"
)

// Знаки в игре
const PLAYER_SYMBOLS string = "XO123456789"

// Описание игрока.
type player struct {
    // Знак, которым он ходит (точнее, его индекс)
    symbol int
    // Сейчас ли его очередь или нет.
    canMakeMove bool
}
func NewPlayer(symbolIndex int) *player {
    p := new(player)
    p.symbol = symbolIndex
    return p
}
func (p *player) GetSymbol() rune {
    return rune(PLAYER_SYMBOLS[p.symbol])
}
func (p *player) CanMoveNow() bool {
    return p.canMakeMove
}
// Возвращает состояние игры

// Максимально доступное количество игроков (X, O и цифры 1-9)
const MAX_NUMBER_OF_PLAYERS = 11

// Состояния слота
type slotState int
const (
    // Слот свободен для подключения игрока.
    SS_FREE slotState = iota
    // Слот занят активным игроком.
    SS_ACTIVE
    // Слот занят отключившимся игроком.
    SS_DISCONNECTED
)

// Огранизует очередь игроков, подключения новых и отключение текущих.
type playerManager struct {
    // Каждый слот соответствует игроку: ещё неподключённому, играющему или отключившемуся от игры
    slots [MAX_NUMBER_OF_PLAYERS]slotState
    // Текущие игроки в игре.
    players [MAX_NUMBER_OF_PLAYERS]*player
    // Количество свободных мест в игре.
    freeSlots int
    // Количество активных игроков.
    activePlayers int
    // Какой игрок должен ходить (индекс в players)
    currentPlayer int
    // Мьютекс для защиты множественной записи-чтения.
    mutex sync.Mutex
}
// Создаёт новый менеджер.
func NewPlayerManager() *playerManager {
    pm := new(playerManager)
    pm.currentPlayer = -1
    pm.freeSlots = MAX_NUMBER_OF_PLAYERS
    return pm
}
// Символы активных игроков.
func (pm *playerManager) ActivePlayers() string {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    buffer := bytes.Buffer{}
    for _, p := range pm.players {
        if p != nil {
            buffer.WriteRune(p.GetSymbol())
        }
    }
    return buffer.String()
}

// Добавляет нового игрока
func (pm *playerManager) AddPlayer() *player {
    if pm.freeSlots == 0 {
        return nil
    }
    // Ищем свободный слот и занимаем его.
    slotIndex := -1
    for index, slot := range pm.slots {
        if slot == SS_FREE {
            slotIndex = index
            break
        }
    }
    pm.slots[slotIndex] = SS_ACTIVE

    // Создаём нового игрока.
    newPlayer := NewPlayer(slotIndex)
    // Если он самый первый на сервере, то очередь хода достаётся ему.
    if pm.activePlayers == 0 {
        newPlayer.canMakeMove = true
        pm.currentPlayer = 0
    }
    pm.activePlayers += 1
    pm.players[slotIndex] = newPlayer

    return newPlayer
}

// Обработка при отключении игрока
func (pm *playerManager) RemovePlayer(p *player) {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    if p == nil {
        return
    }
    for index, playerValue := range pm.players {
        if p == playerValue {
            // Если это игрок, за которым право хода и при этом есть ещё кто-то,
            // то нужно передать ход.
            if p.CanMoveNow() && pm.activePlayers > 1 {
                pm.MoveToNextPlayer()
            }
            // Удаляем игрока
            pm.slots[index] = SS_DISCONNECTED
            pm.players[index] = nil
        }
    }
}

// Текущий игрок, который должен совершить ход.
func (pm *playerManager) CurrentPlayer() *player {
    return pm.players[pm.currentPlayer]
}

// Возвращает информацию о текущих активных игроках, обрамляя знак очереди скобками.
func (pm *playerManager) GetQueueStatus() string {
    var status bytes.Buffer
    for _, value := range pm.players {
        if value != nil {
            if value.CanMoveNow() {
                status.WriteString("[")
                status.WriteString(string(value.GetSymbol()))
                status.WriteString("]")
            } else {
                status.WriteString(string(value.GetSymbol()))
            }
        }
    }
    return status.String()
}

// Передаёт очередность следующему игроку и возвращает указатель на него.
func (pm *playerManager) MoveToNextPlayer() *player {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()

    // Если активных игроков нет, то ничего не возвращаем.
    if pm.activePlayers == 0 {
        return nil
    }

    // Если у нас ровно один активный игрок, то он и ходит.
    if pm.activePlayers == 1 {
        pm.currentPlayer = 0
        return pm.players[pm.currentPlayer]
    }
    // Иначе ищем следующего по списку игрока, который может сходить.

    // Отбираем право хода у текущего игрока
    pm.CurrentPlayer().canMakeMove = false

    isFound := false
    currentIndex := pm.currentPlayer
    for !isFound {
        currentIndex = (currentIndex + 1) % MAX_NUMBER_OF_PLAYERS
        if pm.slots[currentIndex] == SS_ACTIVE {
            isFound = true
        }
    }
    pm.currentPlayer = currentIndex
    // Даём право хода
    pm.CurrentPlayer().canMakeMove = true
    return pm.CurrentPlayer()
}