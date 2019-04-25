package spatialindex

import (
	"math"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/demskie/simplesync"
)

func TestGrid(t *testing.T) {
	if NewGrid(0, math.MinInt64, math.MinInt64, math.MaxInt64, math.MaxInt64) != nil {
		t.Error("NewGrid accepted an invalid input parameter")
	}
	grid := NewGrid(512, math.MinInt64, math.MinInt64, math.MaxInt64, math.MaxInt64)
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
	grid.Move(3, 321, 321)
	err = grid.Move(3, 321, 321)
	if err != nil {
		t.Error(err)
	}
	if grid.Delete(99) == nil {
		t.Error("Delete accepted an id that does not exist")
	}
}

func BenchmarkGridCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewGrid(512, math.MinInt64, math.MinInt64, math.MaxInt64, math.MaxInt64)
	}
}

func BenchmarkGridNeighbor(b *testing.B) {
	grid := NewGrid(512, math.MinInt64, math.MinInt64, math.MaxInt64, math.MaxInt64)
	b.ResetTimer()
	grid.Add(0, 123, 123)
	grid.Add(1, 135, 135)
	grid.Add(2, 123456789, 123456789)
	grid.Add(3, 123456788, 123456788)
	grid.Add(4, 23782, 18914)
	for i := 0; i < b.N; i++ {
		grid.ClosestPoint(-math.MinInt64/2, -math.MinInt64/2)
	}
}

func BenchmarkGridInsertion(b *testing.B) {
	data := getUniformRandomData(1e6)
	grid := NewGrid(512, math.MinInt64, math.MinInt64, math.MaxInt64, math.MaxInt64)
	b.ResetTimer()
	var err error
	for i := uint64(0); i < uint64(b.N); i++ {
		if i < uint64(len(data)) {
			err = grid.Add(i, data[i].X, data[i].Y)
			if err != nil {
				b.Error(err)
			}
			continue
		}
		break
	}
}

var randomData []Point
var randomDataMtx sync.Mutex

func getUniformRandomData(num int) []Point {
	randomDataMtx.Lock()
	defer randomDataMtx.Unlock()
	if len(randomData) > 0 {
		return randomData
	}
	rnum := make([]*rand.Rand, runtime.NumCPU())
	for i := range rnum {
		rnum[i] = rand.New(rand.NewSource(int64(i) + time.Now().UnixNano()))
	}
	tmp := make([][]Point, runtime.NumCPU())
	workers := simplesync.NewWorkerPool(runtime.NumCPU())
	workers.Execute(func(i int) {
		for x := 0; x < num/runtime.NumCPU(); x++ {
			tmp[i] = append(tmp[i], Point{
				ID: 0,
				X:  rnum[i].Int63() * getPosOrNegOne(rnum[i]),
				Y:  rnum[i].Int63() * getPosOrNegOne(rnum[i]),
			})
		}
	})
	result := []Point{}
	for _, val := range tmp {
		result = append(result, val...)
	}
	randomData = result
	return randomData
}

func getPosOrNegOne(rnum *rand.Rand) int64 {
	if rnum.Intn(2) == 0 {
		return 1
	}
	return -1
}
