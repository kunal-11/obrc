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

func parseFile(startOffset, partLen int64, filePath string, result map[string]*CityTotal) {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file: ", err)
		os.Exit(1)
	}
	defer f.Close()

	_, err = f.Seek(startOffset, 0)
	if err != nil {
		fmt.Println("Error seeking file: ", err)
		os.Exit(1)
	}

	buf := make([]byte, 204800)
	itr := Iterator{
		f:   f,
		buf: buf,
	}

	// skip till first newline
	for itr.HasNext() == nil && itr.Next() != '\n' {
	}

	bufName := make([]byte, 0, 128)
	for itr.BytesRead() <= partLen && itr.HasNext() == nil {
		city, err := parseName(&itr, bufName[:0])
		if err != nil {
			fmt.Println("Error parsing name: ", err)
			os.Exit(1)
		}

		temp, err := parseTemp(&itr)
		if err != nil {
			fmt.Println("Error parsing temp: ", err)
			os.Exit(1)
		}

		cur, ok := result[string(city)]
		if !ok {
			cur = &CityTotal{}
			result[string(city)] = cur
		}
		cur.Sum += int64(temp)
		cur.Count += 1
		cur.Min = min(cur.Min, temp)
		cur.Max = max(cur.Max, temp)
	}
}

func parseName(itr *Iterator, name []byte) ([]byte, error) {
	for {
		if err := itr.HasNext(); err != nil {
			return nil, err
		}
		next := itr.Next()
		if next == ';' {
			return name, nil
		}
		name = append(name, next)
	}
}

func parseTemp(itr *Iterator) (int, error) {
	result := 0
	sign := 1
	for {
		if err := itr.HasNext(); err != nil {
			return 0, err
		}
		num := itr.Next()
		if num == byte('\n') {
			return sign * result, nil
		}
		if num == '.' {
			continue
		}
		if num == '-' {
			sign = -1
			continue
		}
		if num >= '0' && num <= '9' {
			result *= 10
			result += int(num - byte('0'))
		}
	}
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
