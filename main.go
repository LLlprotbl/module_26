package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var logChan = make(chan string, 200)

type Buffer struct {
	arr  []int
	pos  int
	size int
	mu   sync.Mutex
}

func NewBuffer(size int) *Buffer {
	logChan <- fmt.Sprintf("Создан новый буфер размером %d", size)
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
		logChan <- fmt.Sprintf("Буфер полный, сдвигаем элементы: %v", b.arr)
		for i := 1; i < b.size; i++ {
			b.arr[i-1] = b.arr[i]
		}
		b.arr[b.size-1] = el
		logChan <- fmt.Sprintf("Добавлен элемент в полный буфер: %v", b.arr)
	} else {
		b.pos++
		b.arr[b.pos] = el
		logChan <- fmt.Sprintf("Добавлен элемент %d в позицию %d: %v", el, b.pos, b.arr[:b.pos+1])
	}
}

func (b *Buffer) Get() []int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pos < 0 {
		logChan <- "Буфер пуст, нечего возвращать"
		return nil
	}

	out := make([]int, b.pos+1)
	copy(out, b.arr[:b.pos+1])
	logChan <- fmt.Sprintf("Возвращаем содержимое буфера: %v", out)
	b.pos = -1
	return out
}

func read(input chan<- int, done <-chan struct{}) {
	defer func() {
		close(input)
		logChan <- "Канал input закрыт"
	}()

	for {
		select {
		case <-done:
			return
		default:
			var u int
			_, err := fmt.Scanf("%d\n", &u)
			if err != nil {
				logChan <- fmt.Sprintf("Ошибка ввода: %v", err)
				continue
			}
			logChan <- fmt.Sprintf("Получено число: %d", u)
			input <- u
		}
	}
}

func filterNegative(input <-chan int, output chan<- int, done <-chan struct{}) {
	defer func() {
		close(output)
		logChan <- "Канал filterNegative закрыт"
	}()

	for {
		select {
		case <-done:
			return
		case r, ok := <-input:
			if !ok {
				return
			}
			logChan <- fmt.Sprintf("Фильтрация числа: %d", r)
			if r >= 0 {
				output <- r
				logChan <- fmt.Sprintf("Число %d прошло фильтр отрицательных", r)
			}
		}
	}
}

func filterThree(input <-chan int, output chan<- int, done <-chan struct{}) {
	defer func() {
		close(output)
		logChan <- "Канал filterThree закрыт"
	}()

	for {
		select {
		case <-done:
			return
		case r, ok := <-input:
			if !ok {
				return
			}
			logChan <- fmt.Sprintf("Проверка числа %d на кратность 3", r)
			if r%3 != 0 {
				output <- r
				logChan <- fmt.Sprintf("Число %d прошло фильтр кратности", r)
			}
		}
	}
}

func writeBuffer(input <-chan int, buffer *Buffer, done <-chan struct{}) {
	defer func() {
		logChan <- "Завершение writeBuffer"
	}()

	for {
		select {
		case <-done:
			return
		case v, ok := <-input:
			if !ok {
				return
			}
			logChan <- fmt.Sprintf("Запись числа %d в буфер", v)
			buffer.Push(v)
		}
	}
}

func printBuffer(buffer *Buffer, ticker *time.Ticker, done <-chan struct{}) {
	defer func() {
		logChan <- "Завершение printBuffer"
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			buf := buffer.Get()
			if len(buf) > 0 {
				logChan <- fmt.Sprintf("Вывод буфера: %v", buf)
				fmt.Println("Данные буфера:", buf)
			}
		}
	}
}

func logger() {
	defer func() {
		logChan <- "Логгер завершает работу"
	}()

	for msg := range logChan {
		log.Println(msg)
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("PIPELINE: ")

	go logger()

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

	time.Sleep(time.Second * time.Duration(interval+1))

	finalBuf := buffer.Get()
	if len(finalBuf) > 0 {
		logChan <- fmt.Sprintf("Финальные данные буфера: %v", finalBuf)
		fmt.Println("Финальные данные буфера:", finalBuf)
	}

	// Даем время на завершение логгера
	time.Sleep(100 * time.Millisecond)
	close(logChan)
}
