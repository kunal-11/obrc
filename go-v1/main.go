package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
)

const BUF_LEN = 1024 * 1024 * 4

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, BUF_LEN)
	},
}

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
		workers: runtime.NumCPU(),
	}
	j.run()
	result := mergeMaps(j.results)

	keys := make([]uint64, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(l, r uint64) int {
		return strings.Compare(result[l].name, result[r].name)
	})
	fmt.Print("{")
	for _, city := range keys {
		score := result[city]
		fmt.Printf("%s=%v/%v/%v, ", score.name, float32(score.min)/10, float32(score.sum/int64(score.count))/10, float32(score.max)/10)
	}
	fmt.Print("}")
}

func mergeMaps(maps []map[uint64]*jobResult) map[uint64]*jobResult {
	res := maps[0]
	for _, m := range maps[1:] {
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
	workers int

	channel chan []byte
	results []map[uint64]*jobResult
}

type jobResult struct {
	sum   int64
	count int
	min   int16
	max   int16

	name string
}

func (j *job) run() {
	j.channel = make(chan []byte, 128)
	go func() {
		defer close(j.channel)
		j.reader()
	}()

	j.results = make([]map[uint64]*jobResult, 0, j.workers)
	wg := sync.WaitGroup{}
	for range j.workers {
		result := make(map[uint64]*jobResult, 1024*8)
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
		buf := bufferPool.Get().([]byte)
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

const (
	// FNV-1 64-bit prime and offset
	FNVPrime  uint64 = 1099511628211
	FNVOffset uint64 = 14695981039346656037
)

func (j *job) worker(result map[uint64]*jobResult) {
	for buf := range j.channel {
		for i := 0; i < len(buf); i++ {
			// parse city name
			endi := i
			h := FNVOffset
			for buf[endi] != ';' {
				h ^= uint64(buf[endi])
				h *= FNVPrime
				endi++
			}

			// update map
			cur, ok := result[h]
			if !ok {
				cur = &jobResult{
					max:  -1000,
					min:  1000,
					name: string(buf[i:endi]),
				}
				result[h] = cur
			}
			i = endi + 1

			// parse temperature
			num := int16(0)
			negative := false
			if buf[i] == '-' {
				negative = true
				i++
			}
			for buf[i] != '\n' {
				if buf[i] != '.' {
					num = num*10 + int16(buf[i]-'0')
				}
				i++
			}
			if negative {
				num *= -1
			}

			cur.count += 1
			cur.sum += int64(num)
			cur.max = max(cur.max, num)
			cur.min = min(cur.min, num)
		}
		bufferPool.Put(buf[:cap(buf)])
	}
}
