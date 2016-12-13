package main

//-----
// Описание бизнес-логики игры.
//-----

import (
	"sync"
	"bytes"
	"fmt"
	"log"
)

// Длина выигрышной серии.
const WINNING_LENGTH = 5

// Кортеж из двух значений.
type cell struct {
	row, column int
}
func (c cell) String() string {
	return fmt.Sprintf("(%d, %d)", c.row, c.column)
}

// Информация о выигрышной ситуации.
// Если какой-то из срезов не nil, то это выигрыш.
type winState struct {
	// Горизонталь.
	Horizontal []cell `json:",omitempty"`
	// Вертикаль.
	Vertical []cell `json:",omitempty"`
	// Главная диагональ.
	MainDiagonal []cell `json:",omitempty"`
	// Побочная диагональ.
	SideDiagonal []cell `json:",omitempty"`
}

// Есть ли выигрыш вообще.
func (ws *winState) HasWinning() bool {
	return ws.Horizontal != nil ||
		ws.Vertical != nil ||
		ws.MainDiagonal != nil ||
		ws.SideDiagonal != nil
}

// Описание самой игры.
type Game struct {
	// Информация о выигрыше определённого игрока.
	*winState

	// Игровое поле.
	field map[cell]rune

	// Мьютекс на запись в поле.
	mu sync.Mutex

	// Символ выигравшего игрока.
	winner rune
}

func NewGame() *Game {
	game := Game{}
	game.field = make(map[cell]rune)
	game.winState = &winState{}
	return &game
}

// Сбрасывает игру.
func (g *Game) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.winState = &winState{}
	g.field = make(map[cell]rune)
	g.winner = 0
}

// Проверяет, свободна ли клетка.
func (g *Game) isCellFree(row, column int) bool {
	_, isPresent := g.field[cell{row, column}]
	return !isPresent
}

// Возвращает символ в клетке.
func (g *Game) get(row, column int) rune {
	return g.field[cell{row, column}]
}
func (g *Game) getOrDefault(c cell, defaultValue rune) rune {
	value, isPresent := g.field[c]
	if isPresent {
		return value
	} else {
		return defaultValue
	}
}

// Вспомогательная функция, которая в строчке ищет самую длинную
// подпоследовательность из заданного символа и возвращает начало и длину.
func findLongestSubstring(seq string, symbol rune) (offset, length int) {

	maxLength := 0
	maxLengthIndex := -1

	counter := 0
	for i, c := range seq {
		if c == symbol {
			counter += 1
		} else {
			counter = 0
		}
		// Сраниваем текущее значение счётчика с максимальным.
		if counter > maxLength {
			maxLength = counter
			maxLengthIndex = i
		}
	}

	return maxLengthIndex - maxLength + 1, maxLength
}

// Проверяет, есть ли выигрышная комбинация в окрестности заданной клетки.
// Если выигрыш есть, то возвращает один или несколько массив выигрышных клеток (от N до 2N-1),
// если выигрышная линия есть по горизонтали, вертикали или диагонали
func (g *Game) CheckWin(row, column int) {

	// Если клетка пуста, то она априори не даёт выигрышной серии.
	if g.isCellFree(row, column) {
		return
	}
	symbol := g.get(row, column)

	// Сборщик строк.
	var buffer bytes.Buffer
	// Найденная максимальная длина подпоследовательности заданного символа.
	var start, maxLength int

	// Линии, которые требуются проверить.
	var horizontal, vertical, mainDiagonal, sideDiagonal [2 * WINNING_LENGTH - 1]cell
	for index, offset := 0, -WINNING_LENGTH + 1; offset <= WINNING_LENGTH - 1; offset, index = offset + 1, index + 1 {
		horizontal[index] = cell{row, column + offset}
		vertical[index] = cell{row + offset, column}
		mainDiagonal[index] = cell{row + offset, column + offset}
		sideDiagonal[index] = cell{row - offset, column + offset}
	}

	log.Printf("Checking for winning...\n")
	names := [...]string{"horizontal", "vertical", "main diagonal", "side diagonal"}
	lines := [...]*[]cell{&g.Horizontal, &g.Vertical, &g.MainDiagonal, &g.SideDiagonal}
	for index, row := range [...][2*WINNING_LENGTH - 1]cell{horizontal, vertical, mainDiagonal, sideDiagonal} {
		buffer.Reset()
		for _, cell := range row {
			buffer.WriteRune(g.getOrDefault(cell, '-'))
		}
		log.Printf("\t%s: <%s>\n", names[index], buffer.String())
		start, maxLength = findLongestSubstring(buffer.String(), symbol)
		if maxLength >= WINNING_LENGTH {
			log.Printf("\t\tWIN! (%d, %d)\n", start, start+maxLength)
			*lines[index] = horizontal[start:start+maxLength]
			g.winner = symbol
		}
	}
}

// Заполняет клетку указанным символом
func (g *Game) Set(row, column int, symbol rune) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.field[cell{row, column}] = symbol
}

// Возвращает состояние игры.
func (g *Game) GetState() *gameState {
	g.mu.Lock()
	defer g.mu.Unlock()
	return NewGameState(g)
}

// Снимок состояния клетки.
type cellState struct {
	Row, Column int
	Symbol rune
}
// Снимок состояния игры, используемое для передачи клиенту.
type gameState struct {
	// Текущие занятые клетки.
	Cells []cellState
	// Выигрышные линии, если они есть.
	WinLines *winState
	// Победитель, если есть.
	Winner rune
}
func NewGameState(g *Game) *gameState {
	gs := new(gameState)
	gs.Cells = make([]cellState, 0)
	gs.WinLines = g.winState
	gs.Winner = g.winner

	for key, value := range g.field {
		gs.Cells = append(gs.Cells, cellState{
			Row: key.row,
			Column: key.column,
			Symbol: value,
		})
	}
	return gs
}
