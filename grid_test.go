package spatialindex

import "testing"
import "math"

func TestGrid(t *testing.T) {
	grid := NewGrid(1000, 1000)
	grid.Add(0, 123, 123)
	grid.Add(1, 135, 135)
	if grid.Add(1, 135, 135) == nil {
		t.Error("id parameter was used twice without error")
	}
	grid.Add(2, 123456789, 123456789)
	grid.Add(3, 123456788, 123456788)
	grid.ClosestNeighbor(-123456789, -123456789)
	grid.Add(4, math.MaxInt64/2, math.MinInt64/2)
	err := grid.Add(5, math.MaxInt64, math.MinInt64)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkGridCreation(b *testing.B) {
	grid := NewGrid(1000, 1000)
	for i := 0; i < b.N; i++ {
		grid.Reset()
	}
}

func BenchmarkGridNeighbor(b *testing.B) {
	grid := NewGrid(1000, 1000)
	grid.Add(0, 123, 123)
	grid.Add(1, 135, 135)
	grid.Add(2, 123456789, 123456789)
	grid.Add(3, 123456788, 123456788)
	grid.Add(4, 23782, 18914)
	grid.Add(5, -123456789, -123456789)
	for i := 0; i < b.N; i++ {
		grid.ClosestNeighbor(-123456789, -123456789)
	}
}

func BenchmarkGridMutation(b *testing.B) {
	grid := NewGrid(1000, 1000)
	for i := int64(0); i < int64(b.N); i++ {
		grid.Add(i, i*1000, i*1000)
	}
}
