package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

func generator(data []string) chan string {
	var inCh = make(chan string)

	go func() {
		defer close(inCh)

		for _, i := range data {
			inCh <- i
		}
	}()

	return inCh
}

func worker(
	id int,
	writer io.Writer,
	waitDurationToDelete time.Duration,
	inCh <-chan string,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	workerText := fmt.Sprintf("[Воркер %d]", id)

	defer fmt.Fprintf(writer, "%s\tУдален\n", workerText)

	fmt.Fprintf(writer, "%s\tДобавлен\n", workerText)

	for {
		select {
		case <-time.After(waitDurationToDelete):
			return
		case str, ok := <-inCh:
			if !ok {
				return
			}

			time.Sleep(waitDurationToDelete * 3)
			fmt.Fprintf(writer, "%s\t\tДанные: %s\n", workerText, str)
		}
	}
}

func manager(
	inCh <-chan string,
	waitDurationToCreateWorker time.Duration,
	waitDurationToDeleteWorker time.Duration,
) {
	wg := &sync.WaitGroup{}
	var workerId int
	outCh := make(chan string)

	for str := range inCh {
		select {
		case <-time.After(waitDurationToCreateWorker):
			wg.Add(1)
			go worker(workerId, os.Stdout, waitDurationToDeleteWorker, outCh, wg)
			outCh <- str
			workerId++
		case outCh <- str:
		}
	}

	close(outCh)
	wg.Wait()
}

func main() {
	var data = []string{
		"data1",
		"data2",
		"data3",
		"data4",
		"data5",
	}

	inCh := generator(data)
	manager(inCh, time.Second, time.Second)
}
