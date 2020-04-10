package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var dataSignerMd5Mutex *sync.Mutex

func dataSignerMd5Wrapper(data string) string {
	dataSignerMd5Mutex.Lock()
	defer dataSignerMd5Mutex.Unlock()
	return DataSignerMd5(data)
}

func newChan() chan interface{} {
	return make(chan interface{}, 1000)
}

func jobExecutor(jobToRun job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer func() {
		close(out)
		wg.Done()
	}()
	jobToRun(in, out)
}

// ExecutePipeline выполняет набор переданных функций, связывая их каналами.
func ExecutePipeline(jobs ...job) {
	dataSignerMd5Mutex = &sync.Mutex{}
	var in chan interface{}
	var out chan interface{}
	wg := &sync.WaitGroup{}
	for _, j := range jobs {
		if out == nil {
			in = newChan()
		} else {
			in = out
		}
		out = newChan()

		wg.Add(1)
		go jobExecutor(j, in, out, wg)
	}

	wg.Wait()
}

/*
SingleHash считает значение crc32(data)+"~"+crc32(md5(data)),
где data — то, что пришло на вход.
*/
func SingleHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for inputDataRaw := range in {
		wg.Add(1)
		go func(inputDataRaw interface{}) {
			defer wg.Done()
			inputInt, ok := inputDataRaw.(int)
			if !ok {
				panic("Can not convert input data to int.")
			}
			inputIntStr := strconv.Itoa(inputInt)

			md5Result := make(chan string)
			go func(res chan string) {
				res <- dataSignerMd5Wrapper(inputIntStr)
			}(md5Result)

			crc32Result := make(chan string)
			go func(res chan string) {
				res <- DataSignerCrc32(inputIntStr)
			}(crc32Result)

			crc32md5Result := make(chan string)
			go func(md5resChan, res chan string) {
				res <- DataSignerCrc32(<-md5resChan)
			}(md5Result, crc32md5Result)

			out <- (<-crc32Result + "~" + <-crc32md5Result)
		}(inputDataRaw)
	}
	wg.Wait()
}

/*
MultiHash для каждого th от 0 до 5 включительно считает значение crc32(th+data)
(конкатенация числа th, приведённого к строке и строки data).
То есть 6 хешей для каждого входящего значения.
На выход выдаётся конкатенация результатов в порядке расчёта (0..5).
*/
func MultiHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for inputDataRaw := range in {
		wg.Add(1)
		go func(inputDataRaw interface{}) {
			defer wg.Done()
			inputData, ok := inputDataRaw.(string)
			if !ok {
				panic("Can not convert input data to string.")
			}

			const maxTh = 5
			subResultChans := make([]chan string, 0)
			var i int
			for i = 0; i <= maxTh; i++ {
				subResultCh := make(chan string)
				th := i
				go func(subResultChan chan string) {
					subResultChan <- DataSignerCrc32(strconv.Itoa(th) + inputData)
				}(subResultCh)
				subResultChans = append(subResultChans, subResultCh)
			}

			result := ""
			for i = 0; i <= maxTh; i++ {
				result = result + <-subResultChans[i]
			}

			out <- result
		}(inputDataRaw)
	}
	wg.Wait()
}

/*
CombineResults получает все данные из входного канала, сортирует,
объединяет отсортированный результат через «_» в одну строку и выдаёт её в выходной канал.
*/
func CombineResults(in, out chan interface{}) {
	results := make([]string, 0)
	for inputDataRaw := range in {

		inputData, ok := inputDataRaw.(string)
		if !ok {
			panic("Can not convert input data to string.")
		}
		results = append(results, inputData)
	}

	sort.Strings(results)

	out <- strings.Join(results, "_")
}

func main() {
	jobs := []job{
		job(func(in, out chan interface{}) {
			out <- 0
			out <- 1
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			for inputDataRaw := range in {
				inputData, ok := inputDataRaw.(string)
				if !ok {
					panic("Can not convert input data to string.")
				}
				fmt.Println(inputData)
			}
		}),
	}
	ExecutePipeline(jobs...)
}
