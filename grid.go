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
	mtx       *sync.RWMutex
	buckets   [][][]Point
	allPoints map[uint64]*Point
}

// NewGrid returns a Grid without preallocating nested slices
func NewGrid(numberOfSquares int) *Grid {
	if numberOfSquares < 4 {
		return nil
	}
	return &Grid{
		mtx:       &sync.RWMutex{},
		buckets:   make([][][]Point, int(math.Sqrt(float64(numberOfSquares)))),
		allPoints: make(map[uint64]*Point, numberOfSquares),
	}
}

// Package level errors
var (
	ErrDuplicateID        = errors.New("id already exists")
	ErrInvalidID          = errors.New("id does not exist")
	ErrNotEnoughNeighbors = errors.New("not enough neighbors")
	ErrNothingFound       = errors.New("nothing found")
)

func calculateBucket(x, y, diameter int64) (xb, yb int64) {
	xb, yb = diameter/2, diameter/2
	xb += x / (2 * (1 + (math.MaxInt64 / diameter)))
	yb += y / (2 * (1 + (math.MaxInt64 / diameter)))
	return xb, yb
}

// Add inserts a new Point into the appropriate bucket if it doesn't already exist
func (g *Grid) Add(id uint64, x, y int64) error {
	g.mtx.Lock()
	_, exists := g.allPoints[id]
	if exists {
		g.mtx.Unlock()
		return ErrDuplicateID
	}
	xb, yb := calculateBucket(x, y, int64(len(g.buckets)))
	newPoint := Point{id, x, y}
	if g.buckets[xb] == nil {
		g.buckets[xb] = make([][]Point, len(g.buckets))
	}
	g.buckets[xb][yb] = append(g.buckets[xb][yb], newPoint)
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
	xb1, yb1 := calculateBucket(point.X, point.Y, int64(len(g.buckets)))
	xb2, yb2 := calculateBucket(x, y, int64(len(g.buckets)))
	if g.buckets[xb2] == nil {
		g.buckets[xb2] = make([][]Point, len(g.buckets))
	}
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

// Delete removes the existing Point
func (g *Grid) Delete(id uint64) error {
	g.mtx.Lock()
	point, exists := g.allPoints[id]
	if !exists {
		g.mtx.Unlock()
		return ErrInvalidID
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

// Reset will empty all buckets
func (g *Grid) Reset() {
	g.mtx.Lock()
	for x := range g.buckets {
		for y := range g.buckets[x] {
			if g.buckets[x][y] == nil {
				continue
			} else if len(g.buckets[x][y]) != 0 {
				g.buckets[x][y] = g.buckets[x][y][:0]
			}
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

func adjustBucket(side, xb, yb, distance, diameter int64) (int64, int64, bool) {
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
	if xb >= diameter || yb >= diameter {
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
	xbStart, ybStart := calculateBucket(originPoint.X, originPoint.Y, int64(len(g.buckets)))
	for distance := int64(1); distance < int64(len(g.buckets)); distance++ {
		for side = 0; side < 9; side++ {
			if side == 0 && distance != 1 {
				continue
			}
			xb, yb, valid = adjustBucket(side, xbStart, ybStart, distance, int64(len(g.buckets)))
			if !valid || g.buckets[xb] == nil || g.buckets[xb][yb] == nil {
				continue
			}
			for i := range g.buckets[xb][yb] {
				otherPoint = &g.buckets[xb][yb][i]
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
		if bestPoint != nil {
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
	return Point{}, ErrNothingFound
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
	return Point{}, ErrNothingFound
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
	xbStart, ybStart := calculateBucket(origin.X, origin.Y, int64(len(g.buckets)))
	var points []Point
	if len(g.buckets[xbStart][ybStart]) > 1 {
		points = make([]Point, 0, len(g.buckets[xbStart][ybStart])-1)
		for _, obj := range g.buckets[xbStart][ybStart] {
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
	for distance := int64(1); distance < int64(len(g.buckets)); distance++ {
		otherPoints = []Point{}
		for side = 1; side < 9; side++ {
			xb, yb, valid = adjustBucket(side, xbStart, ybStart, distance, int64(len(g.buckets)))
			if !valid {
				continue
			}
			if len(g.buckets[xb][yb]) > 0 {
				otherPoints = append(otherPoints, g.buckets[xb][yb]...)
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
	return points, ErrNothingFound
}

// GetUnderlyingBucket is used to return Points given a bucket number
func (g *Grid) GetUnderlyingBucket(bucket int) []Point {
	g.mtx.RLock()
	var i, j int
	if bucket != 0 {
		i = bucket / len(g.buckets)
		j = bucket % len(g.buckets)
	}
	if g.buckets[i] == nil {
		g.mtx.RUnlock()
		return nil
	}
	if g.buckets[i][j] == nil {
		g.mtx.RUnlock()
		return nil
	}
	newSlice := make([]Point, len(g.buckets[i][j]))
	copy(newSlice, g.buckets[i][j])
	g.mtx.RUnlock()
	return newSlice
}
