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

```Terminal
Intel(R) Core(TM) i5-6600K CPU @ 3.50GHz

BenchmarkTreeCreation-4       10000000         192 ns/op         208 B/op          5 allocs/op
BenchmarkGridCreation-4            100    10944119 ns/op    40132655 B/op          3 allocs/op

BenchmarkTreeInsertion-4        200000       17650 ns/op        5544 B/op        204 allocs/op
BenchmarkGridInsertion-4    2000000000        0.21 ns/op           0 B/op          0 allocs/op

BenchmarkTreeNeighbor-4       50000000        29.5 ns/op           0 B/op          0 allocs/op
BenchmarkGridNeighbor-4         100000       14484 ns/op         401 B/op          0 allocs/op
```
