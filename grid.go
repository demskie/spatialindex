package spatialindex

import (
	"errors"
	"math"
	"sort"
	"sync"
)

// Point represents an object in 2D space
type Point struct {
	ID   uint64
	X, Y int64
}

// Grid is a statically set series of slices that Points get put into
type Grid struct {
	mtx        *sync.RWMutex
	minX, minY int64
	maxX, maxY int64
	precision  int64
	buckets    [][]Point
	allPoints  map[uint64]*Point
}

// NewGrid returns a Grid without preallocating nested slices
func NewGrid(precision, minX, minY, maxX, maxY int64) *Grid {
	if precision < 1 {
		return nil
	}
	return &Grid{
		mtx:  &sync.RWMutex{},
		minX: minX, minY: minY,
		maxX: maxX, maxY: maxY,
		precision: precision,
		buckets:   make([][]Point, precision*precision),
		allPoints: make(map[uint64]*Point, 0),
	}
}

// Package level errors
var (
	ErrDuplicateID        = errors.New("id already exists")
	ErrInvalidID          = errors.New("id does not exist")
	ErrNotEnoughNeighbors = errors.New("not enough neighbors")
)

func (g *Grid) calculateBucket(x, y int64) (xb, yb int64) {
	xb = g.precision / 2
	xb += int64(float64(x) / ((float64(g.maxX) - float64(g.minX)) / float64(g.precision)))
	if x >= g.maxX {
		xb = g.precision - 1
	} else if x <= g.minX {
		xb = 0
	}
	yb = g.precision / 2
	yb += int64(float64(y) / ((float64(g.maxY) - float64(g.minY)) / float64(g.precision)))
	if y >= g.maxY {
		yb = g.precision - 1
	} else if y <= g.minY {
		yb = 0
	}
	return xb, yb
}

func (g *Grid) getRealBucket(xb, yb int64) int64 {
	return yb + (xb * g.precision)
}

// Add inserts a new Point into the appropriate bucket if it doesn't already exist
func (g *Grid) Add(id uint64, x, y int64) error {
	g.mtx.Lock()
	_, exists := g.allPoints[id]
	if exists {
		g.mtx.Unlock()
		return ErrDuplicateID
	}
	xb, yb := g.calculateBucket(x, y)
	b := g.getRealBucket(xb, yb)
	newPoint := Point{id, x, y}
	if g.buckets[b] == nil {
		g.buckets[b] = make([]Point, 0, 1)
	}
	g.buckets[b] = append(g.buckets[b], newPoint)
	g.allPoints[id] = &newPoint
	g.mtx.Unlock()
	return nil
}

// Move will remove an existing Point and insert a new one into the appropriate bucket
func (g *Grid) Move(id uint64, x, y int64) error {
	g.mtx.Lock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.Unlock()
		return ErrInvalidID
	}
	if point.X == x && point.Y == y {
		g.mtx.Unlock()
		return nil
	}
	xb1, yb1 := g.calculateBucket(point.X, point.Y)
	b1 := g.getRealBucket(xb1, yb1)
	xb2, yb2 := g.calculateBucket(x, y)
	b2 := g.getRealBucket(xb2, yb2)
	if g.buckets[b2] == nil {
		g.buckets[b2] = make([]Point, 0, 1)
	}
	if xb1 != xb2 || yb1 != yb2 {
		for i := range g.buckets[b1] {
			if g.buckets[b1][i].ID == point.ID {
				g.buckets[b1] = append(g.buckets[b1][:i], g.buckets[b1][i+1:]...)
				break
			}
		}
		newPoint := Point{id, x, y}
		g.buckets[b2] = append(g.buckets[b2], newPoint)
		g.allPoints[id] = &newPoint
	}
	g.mtx.Unlock()
	return nil
}

// Delete removes the existing Point
func (g *Grid) Delete(id uint64) error {
	g.mtx.Lock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.Unlock()
		return ErrInvalidID
	}
	b := g.getRealBucket(g.calculateBucket(point.X, point.Y))
	for i := range g.buckets[b] {
		if g.buckets[b][i].ID == id {
			g.buckets[b] = append(g.buckets[b][:i], g.buckets[b][i+1:]...)
			break
		}
	}
	delete(g.allPoints, id)
	g.mtx.Unlock()
	return nil
}

// Reset will empty all buckets
func (g *Grid) Reset() {
	g.mtx.Lock()
	for i := range g.buckets {
		if g.buckets[i] != nil {
			g.buckets[i] = g.buckets[i][:0]
		}
	}
	g.allPoints = map[uint64]*Point{}
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

func (g *Grid) adjustBucket(side, xb, yb, distance int64) (int64, int64, bool) {
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
		return math.MinInt64, math.MinInt64, false
	}
	if xb >= g.precision || yb >= g.precision {
		return math.MaxInt64, math.MaxInt64, false
	}
	return xb, yb, true
}

func (g *Grid) getClosestPoint(originPoint *Point, checkID bool) *Point {
	var (
		valid                      bool
		bestPoint, otherPoint      *Point
		xb, yb, side               int64
		hypotenuse, bestHypotenuse float64
	)
	xbStart, ybStart := g.calculateBucket(originPoint.X, originPoint.Y)
	for distance := int64(1); distance < g.precision; distance++ {
		for side = 0; side < 9; side++ {
			if side == 0 && distance != 1 {
				continue
			}
			xb, yb, valid = g.adjustBucket(side, xbStart, ybStart, distance)
			if valid {
				b := g.getRealBucket(xb, yb)
				for i := range g.buckets[b] {
					otherPoint = &g.buckets[b][i]
					if checkID && otherPoint.ID == originPoint.ID {
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
		}
		if bestPoint != nil {
			if distance > g.precision {
				return nil
			}
			break
		}
	}
	return bestPoint
}

// ClosestPoint will return the closest Point regardless of proximity
// Please note that the returned Point could be in the same position
func (g *Grid) ClosestPoint(x, y int64) (Point, error) {
	g.mtx.RLock()
	p := g.getClosestPoint(&Point{0, x, y}, false)
	g.mtx.RUnlock()
	if p != nil {
		return *p, nil
	}
	return Point{}, ErrNotEnoughNeighbors
}

// NearestNeighbor will return the first adjacent Point
func (g *Grid) NearestNeighbor(id uint64) (Point, error) {
	g.mtx.RLock()
	p, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return Point{}, ErrInvalidID
	}
	p = g.getClosestPoint(p, true)
	g.mtx.RUnlock()
	if p != nil {
		return *p, nil
	}
	return Point{}, ErrNotEnoughNeighbors
}

type distanceVectors struct {
	points    []Point
	distances []float64
}

func createDistanceVectors(origin *Point, queryPoints []Point) distanceVectors {
	dv := distanceVectors{
		points:    queryPoints,
		distances: make([]float64, len(queryPoints)),
	}
	for i := range queryPoints {
		dv.distances[i] = math.Hypot(float64(origin.X-queryPoints[i].X),
			float64(origin.Y-queryPoints[i].Y))
	}
	return dv
}

func (dv distanceVectors) Len() int {
	return len(dv.distances)
}

func (dv distanceVectors) Swap(i, j int) {
	dv.points[i], dv.points[j] = dv.points[j], dv.points[i]
	dv.distances[i], dv.distances[j] = dv.distances[j], dv.distances[i]
}

func (dv distanceVectors) Less(i, j int) bool {
	return dv.distances[i] < dv.distances[j]
}

// NearestNeighbors returns multiple adjacent Points in order of proximity.
// If unable to fulfill the requested number it will return a slice containing
// an unspecified number of Points and a non-nill error value.
func (g *Grid) NearestNeighbors(id uint64, num int64) ([]Point, error) {
	g.mtx.RLock()
	origin, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return []Point{}, ErrInvalidID
	}
	xbStart, ybStart := g.calculateBucket(origin.X, origin.Y)
	b := g.getRealBucket(xbStart, ybStart)
	var points []Point
	if len(g.buckets[b]) > 1 {
		points = make([]Point, 0, len(g.buckets[b])-1)
		for _, obj := range g.buckets[b] {
			if obj.ID != id {
				points = append(points, obj)
			}
		}
	}
	distanceVectors := createDistanceVectors(origin, points)
	sort.Sort(distanceVectors)
	points = distanceVectors.points
	if int64(len(points)) >= num {
		return points[:num], nil
	}
	var (
		valid        bool
		otherPoints  []Point
		xb, yb, side int64
	)
	for distance := int64(1); distance < g.precision; distance++ {
		otherPoints = []Point{}
		for side = 1; side < 9; side++ {
			xb, yb, valid = g.adjustBucket(side, xbStart, ybStart, distance)
			b := g.getRealBucket(xb, yb)
			if !valid {
				continue
			}
			if len(g.buckets[b]) > 0 {
				otherPoints = append(otherPoints, g.buckets[b]...)
			}
		}
		distanceVectors = createDistanceVectors(origin, otherPoints)
		sort.Sort(distanceVectors)
		points = append(points, distanceVectors.points...)
		if int64(len(points)) >= num {
			g.mtx.RUnlock()
			return points[:num], nil
		}
	}
	g.mtx.RUnlock()
	if len(points) != 0 {
		return points, ErrNotEnoughNeighbors
	}
	return points, ErrNotEnoughNeighbors
}
