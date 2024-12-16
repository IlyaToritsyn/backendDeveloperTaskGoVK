package main

import (
	"bytes"
	"fmt"
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
	isWgDone              bool
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
	workerId1 := rand.Int()
	//workerId2 := rand.Int()
	//workerId3 := rand.Int()

	buffer := &bytes.Buffer{}

	workerInChClosed := make(chan string)
	close(workerInChClosed)

	workerWg := &sync.WaitGroup{}

	workerText1 := fmt.Sprintf("[Воркер %d]", workerId1)
	//workerText2 := fmt.Sprintf("[Воркер %d]", workerId2)
	//workerText3 := fmt.Sprintf("[Воркер %d]", workerId3)

	workerOutput1 := fmt.Sprintf(`%s	Добавлен
%s		Данные: data1
%s		Данные: data2
%s	Удален
`, workerText1, workerText1, workerText1, workerText1)
	//	workerOutput2 := fmt.Sprintf(`%s	Добавлен
	//%s		Данные: data1
	//%s	Удален
	//`, workerText2, workerText2, workerText2)
	//	workerOutput3 := fmt.Sprintf(`%s	Добавлен
	//%s	Удален
	//`, workerText3, workerText3)

	testCases := []testCase{
		{
			id: 1,
			data: testWorkerData{
				id:                   workerId1,
				writer:               buffer,
				waitDurationToDelete: time.Duration(100+rand.Intn(901)) * time.Millisecond,
				inCh:                 nil,
				wg:                   workerWg,
			},
			expectedResult: testWorkerResult{
				output:                workerOutput1,
				isDeletedDueToTimeout: false,
				isWgDone:              true,
			},
		},
		//{
		//	id: 2,
		//	data: testWorkerData{
		//		id:                   workerId2,
		//		writer:               buffer,
		//		waitDurationToDelete: time.Duration(100+rand.Intn(901)) * time.Millisecond,
		//		inCh:                 nil,
		//		wg:                   workerWg,
		//	},
		//	expectedResult: testWorkerResult{
		//		output:                workerOutput2,
		//		isDeletedDueToTimeout: true,
		//		isWgDone:              true,
		//	},
		//},
		//{
		//	id: 3,
		//	data: testWorkerData{
		//		id:                   workerId3,
		//		writer:               buffer,
		//		waitDurationToDelete: time.Duration(100+rand.Intn(901)) * time.Millisecond,
		//		inCh:                 workerInChClosed,
		//		wg:                   workerWg,
		//	},
		//	expectedResult: testWorkerResult{
		//		output:                workerOutput3,
		//		isDeletedDueToTimeout: false,
		//		isWgDone:              true,
		//	},
		//},
	}

	for _, tc := range testCases {
		workerData := tc.data.(testWorkerData)

		workerData.wg.Add(1)
		go worker(workerData.id, workerData.writer, workerData.waitDurationToDelete,
			workerData.inCh, workerData.wg)

		switch tc.id {
		case 1:
			workerData.inCh = make(chan string)
			//workerData.inCh <- "data1"
			//workerData.inCh <- "data2"
			close(workerData.inCh)
		case 2:
			workerData.inCh = make(chan string)
			workerData.inCh <- "data1"

			time.Sleep(workerData.waitDurationToDelete + 100*time.Millisecond)

			workerData.inCh <- "data2"
			close(workerData.inCh)
		}

		expectedResult := tc.expectedResult.(testWorkerResult)
		actualResult := testWorkerResult{}
		actualResult.output = buffer.String()
		actualResult.isDeletedDueToTimeout = len(workerData.inCh) > 0
		workerData.wg.Wait()
		actualResult.isWgDone = true

		if !reflect.DeepEqual(expectedResult, actualResult) {
			t.Errorf("[%d] Wrong result:\n\tExpected:\t%+v\n\tActual:\t\t%+v", tc.id,
				expectedResult, actualResult)
		}

		buffer.Reset()
	}
}
