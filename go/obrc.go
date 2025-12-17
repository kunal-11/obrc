package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"sync"
)

var THREADS = runtime.NumCPU()

func calcParts(filePath string, parts int) ([]int64, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("error reading file stats: %w", err)
	}
	fileSize := stats.Size()
	partLen := fileSize / int64(parts)
	offsets := make([]int64, 0, parts)
	for i := int64(0); i < fileSize; i += partLen {
		offsets = append(offsets, max(i-1, 0))
	}
	return offsets, partLen, nil
}

func calc(filePath string) {
	offsets, partLen, err := calcParts(filePath, THREADS)
	if err != nil {
		fmt.Println("Error splitting file: ", err)
		os.Exit(1)
	}
	results := make([]map[string]*CityTotal, 0, THREADS)

	wg := &sync.WaitGroup{}
	for _, offset := range offsets {
		result := make(map[string]*CityTotal)
		results = append(results, result)
		wg.Add(1)
		go func() {
			defer wg.Done()
			parseFile(offset, partLen, filePath, result)
		}()
	}
	wg.Wait()
	result := mergeMaps(results)

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Print("{")
	for _, city := range keys {
		score := result[city]
		mean := score.Sum / int64(score.Count)
		fmt.Printf("%v=%v/%v/%v, ", city, float64(score.Min)/10, float64(mean)/10, float64(score.Max)/10)
	}
	fmt.Print("}")
}

type CityTotal struct {
	Sum   int64
	Count int
	Min   int
	Max   int
}

func main() {
	f, err := os.Create("./profdata")
	if err != nil {
		fmt.Printf("Error creating pprof file, %v \n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	calc("./../data/measurements.txt")
}
