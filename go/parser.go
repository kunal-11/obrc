package main

import (
	"fmt"
	"os"
)

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
