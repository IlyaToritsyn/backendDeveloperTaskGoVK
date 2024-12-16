package main

import (
	"bytes"
	"io"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

type testCase struct {
	id             int
	data           any
	expectedResult any
}

type testGeneratorResult struct {
	data       []string
	isChClosed bool
}

type testWorkerData struct {
	id                   int
	writer               io.Writer
	waitDurationToDelete time.Duration
	inCh                 chan string
	wg                   *sync.WaitGroup
}

type testWorkerResult struct {
	output                string
	isDeletedDueToTimeout bool
}

func TestGenerator(t *testing.T) {
	generatorDataFilled := []string{"data1", "data2"}
	var generatorDataEmpty []string

	testCases := []testCase{
		{
			id:   1,
			data: generatorDataFilled,
			expectedResult: testGeneratorResult{
				data:       generatorDataFilled,
				isChClosed: true,
			},
		},
		{
			id:   2,
			data: generatorDataEmpty,
			expectedResult: testGeneratorResult{
				data:       generatorDataEmpty,
				isChClosed: true,
			},
		},
	}

	for _, tc := range testCases {
		inCh := generator(tc.data.([]string))

		expectedResult := tc.expectedResult.(testGeneratorResult)
		actualResult := testGeneratorResult{}

	Loop:
		for {
			select {
			case <-time.After(time.Second):
				t.Fatalf("[%d] Receive from channel timed out\n\tExpected:\t%+v\n\tActual:\t\t%+v",
					tc.id, expectedResult, actualResult)
			case str, ok := <-inCh:
				if !ok {
					actualResult.isChClosed = true

					break Loop
				}

				actualResult.data = append(actualResult.data, str)
			}
		}

		if !reflect.DeepEqual(expectedResult, actualResult) {
			t.Errorf("[%d] Wrong result:\n\tExpected:\t%+v\n\tActual:\t\t%+v", tc.id,
				expectedResult, actualResult)
		}
	}
}

func TestWorker(t *testing.T) {
	buffer := &bytes.Buffer{}

	workerOutputData := `[Воркер 1]	Добавлен
[Воркер 1]		Данные: data1
[Воркер 1]	Удален
`
	workerOutputNoData := `[Воркер 1]	Добавлен
[Воркер 1]	Удален
`

	testCases := []testCase{
		{
			id: 1,
			data: testWorkerData{
				id:                   1,
				writer:               buffer,
				waitDurationToDelete: 1,
				inCh:                 make(chan string),
				wg:                   &sync.WaitGroup{},
			},
			expectedResult: testWorkerResult{
				output:                workerOutputData,
				isDeletedDueToTimeout: false,
			},
		},

		// Тест НЕ стабильный
		{
			id: 2,
			data: testWorkerData{
				id:                   1,
				writer:               buffer,
				waitDurationToDelete: 1,
				inCh:                 make(chan string),
				wg:                   &sync.WaitGroup{},
			},
			expectedResult: testWorkerResult{
				output:                workerOutputNoData,
				isDeletedDueToTimeout: true,
			},
		},

		{
			id: 3,
			data: testWorkerData{
				id:                   1,
				writer:               buffer,
				waitDurationToDelete: time.Duration(100+rand.Intn(901)) * time.Millisecond,
				inCh:                 make(chan string),
				wg:                   &sync.WaitGroup{},
			},
			expectedResult: testWorkerResult{
				output:                workerOutputNoData,
				isDeletedDueToTimeout: false,
			},
		},
	}

	for _, tc := range testCases {
		workerData := tc.data.(testWorkerData)

		workerData.wg.Add(1)
		go worker(workerData.id, workerData.writer, workerData.waitDurationToDelete,
			workerData.inCh, workerData.wg)

		expectedResult := tc.expectedResult.(testWorkerResult)
		actualResult := testWorkerResult{}

		switch tc.id {
		case 1:
			select {
			case workerData.inCh <- "data1":
			case <-time.After(time.Second):
				actualResult.isDeletedDueToTimeout = true
			}
		case 2:
			time.Sleep(workerData.waitDurationToDelete * 6 / 5)

			select {
			case workerData.inCh <- "data1":
			case <-time.After(time.Second):
				actualResult.isDeletedDueToTimeout = true
			}
		}

		close(workerData.inCh)
		workerData.wg.Wait()
		actualResult.output = buffer.String()

		if !reflect.DeepEqual(expectedResult, actualResult) {
			t.Errorf("[%d] Wrong result:\n\tExpected:\t%#v\n\tActual:\t\t%#v\n\tData:\t\t%+v",
				tc.id, expectedResult, actualResult, workerData)
		}

		buffer.Reset()
	}
}
