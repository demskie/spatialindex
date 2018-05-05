package spatialindex

import (
	"errors"
	"math"
	"sort"
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
			if g.buckets[x][y] == nil {
				continue
			} else if len(g.buckets[x][y]) != 0 {
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

func (g *Grid) getClosestPoint(originPoint *Point) *Point {
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
			if !valid {
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

func (g *Grid) ClosestPoint(x, y int64) (Point, error) {
	g.mtx.RLock()
	p := g.getClosestPoint(&Point{-1, x, y})
	g.mtx.RUnlock()
	if p != nil {
		return *p, nil
	}
	return Point{ID: -1}, errors.New("nothing found")
}

func (g *Grid) NearestNeighbor(id int64) (Point, error) {
	g.mtx.RLock()
	p, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return Point{ID: -1}, errors.New("id does not exist")
	}
	p = g.getClosestPoint(p)
	g.mtx.RUnlock()
	if p != nil {
		return *p, nil
	}
	return Point{ID: -1}, errors.New("nothing found")
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

func (g *Grid) NearestNeighbors(id, num int64) ([]Point, error) {
	g.mtx.RLock()
	origin, exists := g.allPoints[id]
	if !exists {
		g.mtx.RUnlock()
		return []Point{}, errors.New("id does not exist")
	}
	xbStart, ybStart := calculateBucket(origin.X, origin.Y, int64(len(g.buckets)))
	var points []Point
	if len(g.buckets[xbStart][ybStart]) > 1 {
		points = make([]Point, len(g.buckets[xbStart][ybStart])-1)
		for i := range g.buckets[xbStart][ybStart] {
			if g.buckets[xbStart][ybStart][i].ID != id {
				points[i] = g.buckets[xbStart][ybStart][i]
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
		return points, errors.New("ran out of data - not enough neighbors")
	}
	return points, errors.New("nothing found")
}
