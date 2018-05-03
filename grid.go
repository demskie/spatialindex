package spatialindex

import (
	"errors"
	"math"
	"sync"
	//"fmt"
)

type Point struct {
	ID, X, Y int64
}

type Grid struct {
	mtx       *sync.RWMutex
	buckets   [][][]Point
	allPoints map[int64]*Point
}

func NewGrid(numberOfSquares int32) *Grid {
	if numberOfSquares < 4 {
		return nil
	}
	g := &Grid{
		mtx:       &sync.RWMutex{},
		buckets:   make([][][]Point, int32(math.Sqrt(float64(numberOfSquares)))),
		allPoints: make(map[int64]*Point, numberOfSquares),
	}
	for x := range g.buckets {
		g.buckets[x] = make([][]Point, len(g.buckets))
		for y := range g.buckets[x] {
			g.buckets[x][y] = []Point{}
		}
	}
	return g
}

func calculateBucket(x, y, diameter int64) (xb, yb int64) {
	xb, yb = diameter/2, diameter/2
	xb += x / (2 * (1 + (math.MaxInt64 / diameter)))
	yb += y / (2 * (1 + (math.MaxInt64 / diameter)))
	return xb, yb
}

func (g *Grid) Add(id, x, y int64) error {
	if id < 0 {
		return errors.New("id is a negative number")
	}
	g.mtx.Lock()
	_, exists := g.allPoints[id]
	if exists {
		g.mtx.Unlock()
		return errors.New("id already exists")
	}
	xb, yb := calculateBucket(x, y, int64(len(g.buckets)))
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
	xb1, yb1 := calculateBucket(point.X, point.Y, int64(len(g.buckets)))
	xb2, yb2 := calculateBucket(x, y, int64(len(g.buckets)))
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
	xb, yb := calculateBucket(point.X, point.Y, int64(len(g.buckets)))
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
	isValid = iota
	tooLow
	tooHigh
)

func adjustBucket(side, xb, yb, distance, diameter int64) (int64, int64, int64) {
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
	if xb >= diameter || yb >= diameter {
		return math.MaxInt64, math.MaxInt64, tooHigh
	}
	return xb, yb, isValid
}

func (g *Grid) getClosestPoint(originPoint *Point) *Point {
	var (
		bestPoint, otherPoint      *Point
		xb, yb, side, state        int64
		hypotenuse, bestHypotenuse float64
	)
	xbStart, ybStart := calculateBucket(originPoint.X, originPoint.Y, int64(len(g.buckets)))
	for distance := int64(1); distance < int64(len(g.buckets)); distance++ {
		for side = 0; side < 9; side++ {
			if side == 0 && distance != 1 {
				continue
			}
			xb, yb, state = adjustBucket(side, xbStart, ybStart, distance, int64(len(g.buckets)))
			if state != isValid {
				continue
			}
			for i := range g.buckets[xb][yb] {
				otherPoint = &g.buckets[xb][yb][i]
				if otherPoint.ID == originPoint.ID {
					continue
				}
				hypotenuse = math.Hypot(float64(originPoint.X-otherPoint.X),
					float64(originPoint.Y-otherPoint.Y))
				if hypotenuse < bestHypotenuse || bestHypotenuse == 0 {
					bestHypotenuse = hypotenuse
					bestPoint = otherPoint
				}
			}
		}
		if bestPoint != nil {
			break
		}
	}
	return bestPoint
}

func insertPointIntoOrderedList(p *Point, h float64, bestPoints []Point, bestHypotenuses []float64) ([]Point, []float64) {
	for i, bestPoint := range bestPoints {
		if bestPoint.ID == -1 {
			bestPoints[i] = *p
			bestHypotenuses[i] = h
			return bestPoints, bestHypotenuses
		}
		if h < bestHypotenuses[i] {
			if bestPoints[len(bestPoints)-1].ID != -1 {
				bestPoints = append(bestPoints, Point{ID: -1})
				bestHypotenuses = append(bestHypotenuses, 0)
			}
			copy(bestPoints[i+1:], bestPoints[i:])
			bestPoints[i] = *p
			copy(bestHypotenuses[i+1:], bestHypotenuses[i:])
			bestHypotenuses[i] = h
			return bestPoints, bestHypotenuses
		}
	}
	return bestPoints, bestHypotenuses
}

func (g *Grid) getClosestPoints(originPoint *Point, number int64) []Point {
	var (
		otherPoint          *Point
		xb, yb, side, state int64
		hypotenuse          float64
	)
	bestPoints := make([]Point, number)
	for _, val := range bestPoints {
		val.ID = -1
	}
	bestHypotenuses := make([]float64, number)
	xbStart, ybStart := calculateBucket(originPoint.X, originPoint.Y, int64(len(g.buckets)))
	for distance := int64(1); distance < int64(len(g.buckets)); distance++ {
		for side = 0; side < 9; side++ {
			if side == 0 && distance != 1 {
				continue
			}
			xb, yb, state = adjustBucket(side, xbStart, ybStart, distance, int64(len(g.buckets)))
			if state != isValid {
				continue
			}
			for i := range g.buckets[xb][yb] {
				otherPoint = &g.buckets[xb][yb][i]
				if otherPoint.ID == originPoint.ID {
					continue
				}
				hypotenuse = math.Hypot(float64(originPoint.X-otherPoint.X),
					float64(originPoint.Y-otherPoint.Y))
				insertPointIntoOrderedList(otherPoint, hypotenuse, bestPoints, bestHypotenuses)
			}
		}
		if bestPoints[number-1].ID != -1 {
			break
		}
	}
	return bestPoints
}

func (g *Grid) ClosestPoint(x, y int64) (Point, error) {
	g.mtx.RLock()
	point := g.getClosestPoint(&Point{-1, x, y})
	g.mtx.RUnlock()
	if point != nil {
		return *point, nil
	}
	return Point{ID: -1}, errors.New("nothing found")
}

func (g *Grid) NearestNeighbor(id int64) (Point, error) {
	g.mtx.RLock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return Point{ID: -1}, errors.New("id does not exist")
	}
	point = g.getClosestPoint(point)
	g.mtx.RUnlock()
	if point != nil {
		return *point, nil
	}
	return Point{ID: -1}, errors.New("nothing found")
}

func (g *Grid) NearestNeighbors(id, num int64) ([]Point, error) {
	g.mtx.RLock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return []Point{}, errors.New("id does not exist")
	}
	points := g.getClosestPoints(point, num)
	g.mtx.RUnlock()
	if len(points) != 0 {
		return points, nil
	}
	return points, errors.New("nothing found")
}
