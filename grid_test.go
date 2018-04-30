package spatialindex

import "testing"

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
	val, err := grid.AddWithoutID(183, 123)
	for i := 0; i < 4; i++ {
		if val == i {
			t.Errorf("AddWithoutID returned the wrong value: %v\n", val)
		}
	}
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkGridCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewGrid(1000, 1000)
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
	for i := 0; i < b.N; i++ {
		grid.Add(i, int64(i*1000), int64(i*1000))
	}
}
