package spatialindex

import (
	"math"
	"testing"
)

func TestTreeLookup(t *testing.T) {
	tree := NewTree()
	err := tree.Add(0, 5, 5)
	if err != nil {
		t.Error(err)
	}
	err = tree.Add(1, 6, 6)
	if err != nil {
		t.Error(err)
	}
	if tree.Add(1, 2, 3) == nil {
		t.Error("Add accepted a duplicate id parameter")
	}
	neighbors, err := tree.NearestNeighbors(0, 99)
	if err == nil {
		t.Error("NearestNeighbor accepted an invalid num parameter")
	}
	if len(neighbors) != 1 {
		t.Errorf("returned %v neighbors", len(neighbors))
	}
	if neighbors[0].ID != 1 || neighbors[0].X != 6 || neighbors[0].Y != 6 {
		t.Error("NearestNeighbor function is broken")
	}
	_, err = tree.NearestNeighbors(9000, 99)
	if err == nil {
		t.Error("NearestNeighbor accepted an invalid id parameter")
	}
	err = tree.Delete(1)
	if err != nil {
		t.Error(err)
	}
	if tree.Delete(9000) == nil {
		t.Error("Delete accepted an invalid id parameter")
	}

}

func BenchmarkTreeCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTree()
	}
}

func BenchmarkTreeNeighbor(b *testing.B) {
	tree := NewTree()
	tree.Add(0, 123, 123)
	tree.Add(1, 135, 135)
	tree.Add(2, 123456789, 123456789)
	tree.Add(3, 123456788, 123456788)
	tree.Add(4, 23782, 18914)
	for i := 0; i < b.N; i++ {
		tree.NearestNeighbors(-math.MinInt64/2, -math.MinInt64/2)
	}
}

func BenchmarkTreeInsertion(b *testing.B) {
	data := getUniformRandomData(oneMillion)
	b.ResetTimer()
	tree := NewTree()
	var err error
	for i := int64(0); i < int64(b.N); i++ {
		if i < int64(len(data)) {
			err = tree.Add(uint64(i), float64(data[i].X), float64(data[i].Y))
			if err != nil {
				b.Error(err)
			}
			continue
		}
	}
}
