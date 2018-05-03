package spatialindex

import (
	"math"
	"testing"
)

const (
	oneMillion = 1000000
)

func TestGrid(t *testing.T) {
	grid := NewGrid(oneMillion)
	grid.Add(0, 123, 123)
	grid.Add(1, 135, 135)
	if grid.Add(1, 135, 135) == nil {
		t.Error("id parameter was used twice without error")
	}
	grid.Add(2, 123456789, 123456789)
	grid.Add(3, 123456788, 123456788)
	grid.Add(4, math.MaxInt64/2, math.MinInt64/2)
	err := grid.Add(5, math.MaxInt64, math.MinInt64)
	if err != nil {
		t.Error(err)
	}
	err = grid.Add(6, math.MinInt64, math.MaxInt64)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkGridCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewGrid(oneMillion)
	}
}

func BenchmarkGridNeighbor(b *testing.B) {
	grid := NewGrid(oneMillion)
	grid.Add(0, 123, 123)
	grid.Add(1, 135, 135)
	grid.Add(2, 123456789, 123456789)
	grid.Add(3, 123456788, 123456788)
	grid.Add(4, 23782, 18914)
	for i := 0; i < b.N; i++ {
		grid.ClosestPoint(-math.MinInt64/2, -math.MinInt64/2)
	}
}

func BenchmarkGridMutation(b *testing.B) {
	grid := NewGrid(oneMillion)
	for i := int64(0); i < int64(b.N); i++ {
		grid.Add(i, i*1000, i*1000)
	}
}
