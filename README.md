# spatialindex

NewTree is a wrapper around https://github.com/dhconnelly/rtreego

```Go
func main() {
	tree := spatialindex.NewTree()
	tree.Add(0, 123, 123)
	tree.Add(1, 1234, 1234)
	tree.Add(2, 12345, 12345)
	results, err := tree.NearestNeighbors(0, 2)
	if err != nil {
		fmt.Printf("result: %v\n", err)
	} else {
		fmt.Printf("results: %+v\n", results)
	}
}
// result: [{ID:1 X:1234 Y:1234} {ID:2 X:12345 Y:12345}]
```

NewGrid is an overly simplified nearest neighbor implementation. It is only valid for spatial data that mutates rapidly and has very little bias. The more clustered data is the slower NearestNeighbor lookups will take.

```Go
func main() {
	grid := spatialindex.NewGrid(1000000)
	grid.Add(0, 123, 123)
	grid.Add(1, 1234, 1234)
	grid.Add(2, 12345, 12345)
	results, err := grid.NearestNeighbors(0, 2)
	if err != nil {
		fmt.Printf("result: %v\n", err)
	} else {
		fmt.Printf("result: %+v\n", results)
	}
}
// result: [{ID:1 X:1234 Y:1234} {ID:2 X:12345 Y:12345}]
```

Benchmark results (with evenly spaced spatial data)
```
macbookZeta:~ demskie$ go test github.com/demskie/spatialindex -bench=.
goos: darwin
goarch: amd64
pkg: github.com/demskie/spatialindex
BenchmarkGridCreation-8    	     200	   8927625 ns/op
BenchmarkGridNeighbor-8    	  100000	     12247 ns/op
BenchmarkGridInsertion-8   	2000000000	         0.23 ns/op
BenchmarkTreeCreation-8    	10000000	       153 ns/op
BenchmarkTreeNeighbor-8    	30000000	        49.5 ns/op
BenchmarkTreeInsertion-8   	  200000	     14916 ns/op
```
