package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// буфер
type Buffer struct {
	arr  []int
	pos  int
	size int
	mu   sync.Mutex
}

func NewBuffer(size int) *Buffer {
	return &Buffer{
		arr:  make([]int, size),
		pos:  -1,
		size: size,
	}
}

func (b *Buffer) Push(el int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pos == b.size-1 {
		for i := 1; i < b.size; i++ {
			b.arr[i-1] = b.arr[i]
		}
		b.arr[b.size-1] = el
	} else {
		b.pos++
		b.arr[b.pos] = el
	}
}

func (b *Buffer) Get() []int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pos < 0 {
		return nil
	}

	out := make([]int, b.pos+1)
	copy(out, b.arr[:b.pos+1])
	b.pos = -1
	return out
}

func read(input chan<- int, done <-chan struct{}) {
	defer close(input) // Закрываем канал при завершении
	for {
		select {
		case <-done:
			return
		default:
			var u int
			_, err := fmt.Scanf("%d\n", &u)
			if err != nil {
				fmt.Println("Это не число, попробуй те еще раз")
				continue
			}
			input <- u
		}
	}
}

func filterNegative(input <-chan int, output chan<- int, done <-chan struct{}) {
	defer close(output)
	for {
		select {
		case <-done:
			return
		case r, ok := <-input:
			if !ok {
				return
			}
			if r >= 0 {
				output <- r
			}
		}
	}
}

func filterThree(input <-chan int, output chan<- int, done <-chan struct{}) {
	defer close(output)
	for {
		select {
		case <-done:
			return
		case r, ok := <-input:
			if !ok {
				return
			}
			if r%3 != 0 {
				output <- r
			}
		}
	}
}

func writeBuffer(input <-chan int, buffer *Buffer, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case v, ok := <-input:
			if !ok {
				return
			}
			buffer.Push(v)
		}
	}
}

func printBuffer(buffer *Buffer, ticker *time.Ticker, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			buf := buffer.Get()
			if len(buf) > 0 {
				fmt.Println("Данные буфера:", buf)
			}
		}
	}
}

func main() {
	done := make(chan struct{})
	input := make(chan int)
	go read(input, done)

	filterNeg := make(chan int)
	go filterNegative(input, filterNeg, done)

	filterThreeCh := make(chan int)
	go filterThree(filterNeg, filterThreeCh, done)

	size := 10
	buffer := NewBuffer(size)

	go writeBuffer(filterThreeCh, buffer, done)

	interval := 5
	ticker := time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()
	go printBuffer(buffer, ticker, done)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nПолучен сигнал завершения работы")
	close(done)

	// Даем время на завершение горутин и вывод буфера
	time.Sleep(time.Second * time.Duration(interval+1))

	// Выводим оставшиеся данные в буфере
	finalBuf := buffer.Get()
	if len(finalBuf) > 0 {
		fmt.Println("Финальные данные буфера:", finalBuf)
	}
}
