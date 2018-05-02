package spatialindex

import (
	"errors"
	"math"
	"sync"
	//"fmt"
)

type Point struct {
	ID, X, Y   int64
}

type Grid struct {
	mtx       *sync.RWMutex
	rows      int64
	columns   int64
	buckets   [][][]Point
	allPoints map[int64]*Point
}

func NewGrid(rows, columns int64) *Grid {
	if rows < 2 || columns < 2 {
		return nil
	}
	g := &Grid{
		mtx:       &sync.RWMutex{},
		rows:      rows,
		columns:   columns,
		buckets:   make([][][]Point, rows),
		allPoints: make(map[int64]*Point, rows*columns),
	}
	for x := range g.buckets {
		g.buckets[x] = make([][]Point, rows)
		for y := range g.buckets[x] {
			g.buckets[x][y] = []Point{}
		}
	}
	return g
}

func calculateBucket(x, y, rows, columns int64) (xb, yb int64) {
	xb, yb = columns/2, rows/2
	xb += x / (2 * (1+(math.MaxInt64 / columns)))
	yb += y / (2 * (1+(math.MaxInt64 / rows)))
	return xb, yb
}

func (g *Grid) Add(id, x, y int64) error {
	g.mtx.Lock()
	_, exists := g.allPoints[id]
	if exists {
		g.mtx.Unlock()
		return errors.New("id already exists")
	}
	xb, yb := calculateBucket(x, y, g.rows, g.columns)
	newPoint := Point{id, x, y}
	g.buckets[xb][yb] = append(g.buckets[xb][yb], newPoint)
	g.allPoints[id] = &newPoint
	g.mtx.Unlock()
	return nil
}

func (g *Grid) Move(id, x, y int64) error {
	g.mtx.Lock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.Unlock()
		return errors.New("id does not exist")
	}
	if point.X == x && point.Y == y {
		g.mtx.Unlock()
		return nil
	}
	xb1, yb1 := calculateBucket(point.X, point.Y, g.rows, g.columns)
	xb2, yb2 := calculateBucket(x, y, g.rows, g.columns)
	if xb1 != xb2 || yb1 != yb2 {
		for i := range g.buckets[xb1][yb1] {
			if g.buckets[xb1][yb1][i].ID == point.ID {
				g.buckets[xb1][yb1] = append(g.buckets[xb1][yb1][:i],
					g.buckets[xb1][yb1][i+1:]...)
				break
			}
		}
		newPoint := Point{id, x, y}
		g.buckets[xb2][yb2] = append(g.buckets[xb2][yb2], newPoint)
		g.allPoints[id] = &newPoint
	}
	g.mtx.Unlock()
	return nil
}

func (g *Grid) Delete(id int64) error {
	g.mtx.Lock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return errors.New("id does not exist")
	}
	xb, yb := calculateBucket(point.X, point.Y, g.rows, g.columns)
	for i := range g.buckets[xb][yb] {
		if g.buckets[xb][yb][i].ID == id {
			g.buckets[xb][yb] = append(g.buckets[xb][yb][:i],
				g.buckets[xb][yb][i+1:]...)
			break
		}
	}
	delete(g.allPoints, id)
	g.mtx.Unlock()
	return nil
}

func (g *Grid) Reset() {
	g.mtx.Lock()
	for x := range g.buckets {
		for y := range g.buckets[x] {
			if len(g.buckets[x][y]) != 0 {
				g.buckets[x][y] = g.buckets[x][y][:0]
			}
		}
	}
	g.allPoints = map[int64]*Point{}
	g.mtx.Unlock()
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

func adjustBucket(side, xb, yb, distance, rows, columns int64) (int64, int64, int64) {
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
		return math.MinInt64, math.MinInt64, tooLow
	}
	if xb >= columns || yb >= rows {
		return math.MaxInt64, math.MaxInt64, tooHigh
	}
	return xb, yb, valid
}

func (g *Grid) ClosestNeighbor(x, y int64) (Point, error) {
	g.mtx.RLock()
	var (
		xb, yb, side, state        int64
		point, bestPoint           *Point
		hypotenuse, bestHypotenuse float64
	)
	xbStart, ybStart := calculateBucket(x, y, g.rows, g.columns)
	for distance := int64(1); distance < math.MaxInt64; distance++ {
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
			g.mtx.RUnlock()
			return *bestPoint, nil
		}
	}
	g.mtx.RUnlock()
	return Point{}, errors.New("nothing found")
}
