package main

import "testing"

func BenchmarkCompare(b *testing.B) {
	g1, g2 := RandomGuess(), RandomGuess()
	for i := 0; i < b.N; i++ {
		_ = Compare(g1, g2)
	}
}

func BenchmarkFL(b *testing.B) {
	answer := Compare(RandomGuess(), RandomGuess())
	fl := FrequencyList{}
	for i := 0; i < b.N; i++ {
		fl.Incr(answer)
	}
}

