package spatialindex

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Point struct {
	ID   int
	X, Y int64
}

type Grid struct {
	mtx       *sync.RWMutex
	rows      int
	columns   int
	buckets   [][][]Point
	allPoints map[int]*Point
	rnum      *rand.Rand
}

func NewGrid(rows, columns int) *Grid {
	g := &Grid{
		mtx:       &sync.RWMutex{},
		rows:      rows,
		columns:   columns,
		buckets:   make([][][]Point, rows),
		allPoints: make(map[int]*Point, rows*columns),
		rnum:      nil,
	}
	for x := range g.buckets {
		g.buckets[x] = make([][]Point, rows)
		for y := range g.buckets[x] {
			g.buckets[x][y] = []Point{}
		}
	}
	return g
}

func calculateBucket(x int64, y int64, rows int, columns int) (xb, yb int) {
	xb = columns / 2
	if x > 0 {
		xb += int(x / (math.MaxInt64 / int64(columns)))
	} else if x < 0 {
		xb += int(x / (math.MaxInt64 / int64(columns)))
		xb--
	}
	yb = rows / 2
	if y > 0 {
		yb += int(y / (math.MaxInt64 / int64(rows)))
	} else if x < 0 {
		yb += int(y / (math.MaxInt64 / int64(rows)))
		yb--
	}
	return xb, yb
}

func (g *Grid) Add(id int, x int64, y int64) error {
	g.mtx.Lock()
	_, exists := g.allPoints[id]
	if exists {
		g.mtx.Unlock()
		return errors.New("id parameter is not valid as it already exists")
	}
	xb, yb := calculateBucket(x, y, g.rows, g.columns)
	newPoint := Point{id, x, y}
	g.buckets[xb][yb] = append(g.buckets[xb][yb], newPoint)
	g.allPoints[id] = &newPoint
	g.mtx.Unlock()
	return nil
}

// AddWithoutID should only be used in extreme circumstances. Use the Add() func
// instead whilst keeping track of IDs externally to avoid the performance penalty.
func (g *Grid) AddWithoutID(x, y int64) (id int, err error) {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if g.rnum == nil {
		timeInt := time.Now().UnixNano()
		g.rnum = rand.New(rand.NewSource(timeInt))
	}
	for i := 0; i < math.MaxInt16; i++ {
		id = g.rnum.Int()
		_, exists := g.allPoints[id]
		if exists {
			continue
		}
		xb, yb := calculateBucket(x, y, g.rows, g.columns)
		newPoint := Point{id, x, y}
		g.buckets[xb][yb] = append(g.buckets[xb][yb], newPoint)
		g.allPoints[id] = &newPoint
		return id, nil
	}
	return -1, errors.New("could not find an id")
}

func (g *Grid) Move(id int, x int64, y int64) error {
	return nil
}

func (g *Grid) Delete(id int) error {
	return nil
}

const (
	center = iota
	bottom
	bottomLeft
	left
	topLeft
	top
	topRight
	right
	bottomRight
)

const (
	valid = iota
	tooLow
	tooHigh
)

func adjustBucket(side, xb, yb, distance, rows, columns int) (int, int, int) {
	switch side {
	case center:
		// do nothing
	case bottom:
		yb -= distance
	case bottomLeft:
		xb -= distance
		yb -= distance
	case left:
		xb -= distance
	case topLeft:
		xb -= distance
		yb += distance
	case top:
		yb += distance
	case topRight:
		xb += distance
		yb += distance
	case right:
		xb += distance
	case bottomRight:
		xb += distance
		yb -= distance
	default:
		panic("InvalidParameter")
	}
	if xb < 0 || yb < 0 {
		return math.MinInt32, math.MinInt32, tooLow
	}
	if xb > columns || yb > rows {
		return math.MaxInt32, math.MaxInt32, tooHigh
	}
	return xb, yb, valid
}

func (g *Grid) ClosestNeighbor(x int64, y int64) (Point, error) {
	var (
		xb, yb, side, state        int
		point, bestPoint           *Point
		hypotenuse, bestHypotenuse float64
	)
	xbStart, ybStart := calculateBucket(x, y, g.rows, g.columns)
	for distance := 1; distance < math.MaxInt32; distance++ {
		for side = 0; side < 9; side++ {
			if side == 0 && distance != 1 {
				continue
			}
			xb, yb, state = adjustBucket(side, xbStart, ybStart, distance, g.rows, g.columns)
			if state != valid {
				continue
			}
			for i := range g.buckets[xb][yb] {
				point = &g.buckets[xb][yb][i]
				hypotenuse = math.Hypot(float64(x-point.X), float64(y-point.Y))
				if hypotenuse < bestHypotenuse || bestHypotenuse == 0 {
					bestHypotenuse = hypotenuse
					bestPoint = point
				}
			}
		}
		if bestPoint != nil {
			return *bestPoint, nil
		}
	}
	return Point{}, errors.New("nothing found")
}
