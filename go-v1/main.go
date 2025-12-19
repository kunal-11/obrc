package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
)

func main() {
	f, err := os.Create("./profdata")
	if err != nil {
		fmt.Printf("Error creating pprof file, %v \n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	file, err := os.Open("../data/measurements.txt")
	if err != nil {
		fmt.Print("error opening file: ", err)
		os.Exit(1)
	}
	defer file.Close()

	j := job{
		file:    file,
		bufLen:  1024 * 1024 * 32,
		workers: runtime.NumCPU(),
	}
	j.run()

	result := mergeMaps(j.results)

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Print("{")
	for _, city := range keys {
		score := result[city]
		mean := score.sum / int64(score.count)
		fmt.Printf("%v=%v/%v/%v, ", city, float32(score.min)/10, float32(mean)/10, float32(score.max)/10)
	}
	fmt.Print("}")
}

func mergeMaps(maps []map[string]*jobResult) map[string]*jobResult {
	res := make(map[string]*jobResult, 1024*8)
	for _, m := range maps {
		for k, v := range m {
			val, ok := res[k]
			if !ok {
				res[k] = v
			} else {
				val.count += v.count
				val.sum += v.sum
				val.max = max(val.max, v.max)
				val.min = min(val.min, v.min)
			}
		}
	}
	return res
}

type job struct {
	// Options
	file    *os.File
	bufLen  int64
	workers int

	channel chan []byte
	results []map[string]*jobResult
}

type jobResult struct {
	sum   int64
	count int
	min   int
	max   int
}

func (j *job) run() {
	j.channel = make(chan []byte, j.workers)
	go func() {
		defer close(j.channel)
		j.reader()
	}()

	wg := sync.WaitGroup{}
	for range j.workers {
		result := make(map[string]*jobResult, 1024*8)
		j.results = append(j.results, result)
		wg.Add(1)
		go func() {
			defer wg.Done()
			j.worker(result)
		}()
	}
	wg.Wait()
}

func (j *job) reader() {
	start := int64(0)
	for {
		buf := make([]byte, j.bufLen)
		readLen, err := j.file.ReadAt(buf, start)
		if err == io.EOF {
			j.channel <- buf[:readLen]
			break
		} else if err != nil {
			fmt.Print("error reading file: ", err)
			os.Exit(1)
		}
		i := readLen - 1
		for i >= 0 && buf[i] != '\n' {
			i--
		}
		start += int64(i + 1)
		j.channel <- buf[:i+1]
	}
}

func (j *job) worker(result map[string]*jobResult) {
	for buf := range j.channel {
		for i := 0; i < len(buf); {
			// parse city name
			j := i
			for buf[j] != ';' {
				j++
			}
			name := buf[i:j]
			i = j + 1

			// parse temperature
			num := 0
			sign := 1
			if buf[i] == '-' {
				sign = -1
				i++
			}
			for buf[i] != '\n' {
				if buf[i] != '.' {
					num = num*10 + int(buf[i]-'0')
				}
				i++
			}
			temp := num * sign
			i++

			// update map
			cur, ok := result[string(name)]
			if !ok {
				cur = &jobResult{
					max: temp,
					min: temp,
				}
				result[string(name)] = cur
			}

			cur.count += 1
			cur.sum += int64(temp)
			cur.max = max(cur.max, temp)
			cur.min = min(cur.min, temp)
		}
	}
}
