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

var THREADS = runtime.NumCPU()

type jobParts struct {
	start, end int64
}

func findNewLine(file *os.File, offset int64) (int64, error) {
	buf := make([]byte, 2048)
	read, err := file.ReadAt(buf, offset)
	if err != io.EOF && err != nil {
		return 0, err
	}
	for i := range read {
		if buf[i] == '\n' {
			return offset + int64(i), nil
		}
	}
	if err != io.EOF {
		return 0, fmt.Errorf("newline not found after %v bytes", read)
	}
	return offset + int64(read), io.EOF
}

func calcParts(file *os.File, parts int) ([]jobParts, error) {
	stats, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error reading file stats: %w", err)
	}
	fileSize := stats.Size()
	partLen := fileSize / int64(parts)

	offsets := make([]jobParts, 0, parts)
	offset := int64(0)
	for range parts {
		end, err := findNewLine(file, offset+partLen)
		if err == io.EOF {
			end = min(end, fileSize-1)
		} else if err != nil {
			return nil, err
		}
		offsets = append(offsets, jobParts{start: offset, end: end})
		offset = end + 1
	}
	return offsets, nil
}

func calc(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file: ", err)
		os.Exit(1)
	}
	defer file.Close()

	offsets, err := calcParts(file, THREADS)
	if err != nil {
		fmt.Println("Error splitting file: ", err)
		os.Exit(1)
	}
	results := make([]map[string]*jobResult, 0, THREADS)

	wg := &sync.WaitGroup{}
	for _, offset := range offsets {
		result := make(map[string]*jobResult, 128)
		results = append(results, result)
		job := &job{
			f:      file,
			start:  offset.start,
			end:    offset.end,
			result: result,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			job.run()
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
