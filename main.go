package main

import (
	"fmt"
	"math/bits"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Генерация матрицы Уолша (итеративно, без рекурсии)
func walshMatrix(n int) [][]int {
	if n == 0 || (n&(n-1)) != 0 {
		panic("n должно быть степенью двойки и > 0")
	}

	// Итеративная генерация матрицы Уолша (порядок Адамара)
	m := make([][]int, n)
	for i := range m {
		m[i] = make([]int, n)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			// Битовая магия для вычисления элемента Уолша
			x := i & j
			bits := bits.OnesCount(uint(x))
			if bits%2 == 0 {
				m[i][j] = 1
			} else {
				m[i][j] = -1
			}
		}
	}
	return m
}

//Станции

type Station struct {
	id   int
	name string
	word string
	code []int
	bits []int
}

var messages = map[string]string{
	"A": "GOD",
	"B": "CAT",
	"C": "HAM",
	"D": "SUN",
}

func asciiToBits(s string) []int {
	var bits []int
	for _, ch := range s {
		for i := 7; i >= 0; i-- {
			bit := 0
			if ch&(1<<i) != 0 {
				bit = 1
			}
			bits = append(bits, bit)
		}
	}
	return bits
}

func spreadBit(bit int, code []int) []int {
	val := 1
	if bit == 0 {
		val = -1
	}
	chips := make([]int, len(code))
	for i, c := range code {
		chips[i] = val * c
	}
	return chips
}

func newStation(id int, code []int) *Station {
	name := string(rune('A' + id))
	word := messages[name]

	dataBits := asciiToBits(word)

	var spread []int
	for _, b := range dataBits {
		spread = append(spread, spreadBit(b, code)...)
	}

	return &Station{
		id:   id,
		name: name,
		word: word,
		code: code,
		bits: spread,
	}
}

// Канал CDMA
type Channel struct {
	mu     sync.Mutex
	signal []int
}

func NewChannel() *Channel {
	return &Channel{}
}

func (ch *Channel) AddStation(s *Station) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.signal) == 0 {
		ch.signal = make([]int, len(s.bits))
		copy(ch.signal, s.bits)
	} else {
		for i := range ch.signal {
			ch.signal[i] += s.bits[i]
		}
	}
}

func (ch *Channel) Decode(stationCode []int) string {
	ch.mu.Lock()
	signal := make([]int, len(ch.signal))
	copy(signal, ch.signal)
	ch.mu.Unlock()

	chipLen := len(stationCode)
	bitsCount := len(signal) / chipLen

	var recoveredBits []int
	for i := 0; i < bitsCount; i++ {
		start := i * chipLen

		sum := 0
		for j := 0; j < chipLen; j++ {
			sum += signal[start+j] * stationCode[j]
		}

		// После корреляции: положительная сумма 1, иначе 0
		bit := 0
		if sum > 0 {
			bit = 1
		}
		recoveredBits = append(recoveredBits, bit)
	}

	// Биты → байты → строка
	var bytes []byte
	for i := 0; i < len(recoveredBits)-7; i += 8 { // -7 чтобы не вылезти за границы
		b := uint8(0)
		for j := 0; j < 8; j++ {
			if recoveredBits[i+j] == 1 {
				b |= (1 << (7 - j))
			}
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

// Анимация
var colors = []*color.Color{
	color.New(color.FgRed),
	color.New(color.FgGreen),
	color.New(color.FgYellow),
	color.New(color.FgBlue),
}

func printSignal(signal []int, tick int) {
	fmt.Print("\033[H\033[2J") // очистка экрана
	println(color.New(color.Bold, color.FgCyan).Sprintf("=== CDMA эфир | кадр %d ===", tick))
	fmt.Print("Суммарный сигнал (чипы): ")
	for i, v := range signal {
		switch {
		case v > 0:
			color.Set(color.FgHiWhite)
			fmt.Print("+")
		case v < 0:
			color.Set(color.FgHiBlack)
			fmt.Print("−")
		default:
			color.Set(color.FgMagenta)
			fmt.Print("·")
		}
		if (i+1)%8 == 0 {
			fmt.Print(" ")
		}
	}
	color.Unset()
	println(color.New(color.Bold).Sprint("Приёмники декодируют:"))
}

func main() {
	println(color.New(color.Bold, color.FgHiMagenta).Sprintf(`
      ╔══════════════════════════════════════╗
      ║      CDMA с кодами Уолша на Go       ║
      ║   4 базовые станции вещают в эфире   ║
      ╚══════════════════════════════════════╝
	`))
	time.Sleep(2 * time.Second)

	// Генерация кодов Уолша длиной 8
	walsh := walshMatrix(8)
	codes := walsh[:4] // берём первые 4 строки

	channel := NewChannel()
	var stations []*Station

	for i := 0; i < 4; i++ {
		s := newStation(i, codes[i])
		stations = append(stations, s)
		channel.AddStation(s)
	}

	tick := 0
	for {
		printSignal(channel.signal, tick)

		for i, s := range stations {
			decoded := channel.Decode(s.code)
			col := colors[i]
			status := "✔"
			if decoded != s.word {
				status = "✘"
			}
			col.Printf("  📡 Станция %s → %s \"%s\"\n", s.name, status, decoded)
		}

		println("\nНажмите Ctrl+C для выхода")
		time.Sleep(800 * time.Millisecond)
		tick++
	}
}
