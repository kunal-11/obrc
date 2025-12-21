package main

import "testing"

func BenchmarkReadFile(b *testing.B) {
	path := "../../data/measurements.txt"
	for b.Loop() {
		err := ReadFile(path, 3, 1024*1024*4)
		if err != nil {
			b.Error(err)
		}
	}
}
