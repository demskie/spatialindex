package spatialindex

import "testing"

func TestTreeLookup(t *testing.T) {
	tree := NewTree()

	err := tree.Insert(0, 5, 5)
	if err != nil {
		t.Error(err)
	}

	err = tree.Insert(1, 6, 6)
	if err != nil {
		t.Error(err)
	}

	neighbors, err := tree.NearestNeighbors(0, 99)
	if err != nil {
		t.Error(err)
	}
	if len(neighbors) != 1 {
		t.Errorf("returned %v neighbors", len(neighbors))
	}
	if neighbors[0].ID != 1 || neighbors[0].X != 6 || neighbors[0].Y != 6 {
		t.Error("NearestNeighbor function is broken")
	}
}

func BenchmarkTreeCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewTree()
	}
}

func BenchmarkTreeNeighbor(b *testing.B) {
	tree := NewTree()
	tree.Insert(0, 123, 123)
	tree.Insert(1, 135, 135)
	tree.Insert(2, 123456789, 123456789)
	tree.Insert(3, 123456788, 123456788)
	tree.Insert(4, 23782, 18914)
	tree.Insert(5, -123456789, -123456789)
	for i := 0; i < b.N; i++ {
		tree.NearestNeighbors(5, 1)
	}
}

func BenchmarkTreeInsertion(b *testing.B) {
	data := getUniformRandomData(oneMillion)
	b.ResetTimer()
	tree := NewTree()
	var err error
	for i := int64(0); i < int64(b.N); i++ {
		if i < int64(len(data)) {
			err = tree.Insert(uint64(i), float64(data[i].X), float64(data[i].Y))
			if err != nil {
				b.Error(err)
			}
			continue
		}
	}
}
