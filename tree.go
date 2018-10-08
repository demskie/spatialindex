package spatialindex

import (
	"sync"

	"github.com/dhconnelly/rtreego"
)

// Tree is a wrapper struct around dhconnelly/rtreego.Rtree
type Tree struct {
	rt     *rtreego.Rtree
	mtx    *sync.RWMutex
	objMap map[uint64]*customSpatial
}

type customSpatial struct {
	rct *rtreego.Rect
	id  uint64
}

func (sp *customSpatial) Bounds() *rtreego.Rect {
	return sp.rct
}

// NewTree returns a new wrapper struct
func NewTree() *Tree {
	return &Tree{
		rt:     rtreego.NewTree(2, 2, 3),
		mtx:    &sync.RWMutex{},
		objMap: make(map[uint64]*customSpatial),
	}
}

// Add creates and inserts a new Point into the RTree
func (t *Tree) Add(id uint64, x, y float64) error {
	rect, err := rtreego.NewRect(rtreego.Point{x, y}, []float64{0.01, 0.01})
	if err != nil {
		return err
	}
	t.mtx.Lock()
	_, exists := t.objMap[id]
	if exists {
		t.mtx.Unlock()
		return ErrInvalidID
	}
	newSpatial := &customSpatial{rect, id}
	t.objMap[id] = newSpatial
	t.rt.Insert(newSpatial)
	t.mtx.Unlock()
	return nil
}

// Delete removes a specific Point from the RTree
func (t *Tree) Delete(id uint64) error {
	t.mtx.Lock()
	obj, exists := t.objMap[id]
	if !exists {
		t.mtx.Unlock()
		return ErrInvalidID
	}
	delete(t.objMap, id)
	t.rt.Delete(obj)
	t.mtx.Unlock()
	return nil
}

// Object is the base type that gets inserted into the RTree
type Object struct {
	ID   uint64
	X, Y float64
}

// NearestNeighbors returns a list of Objects closest to the Point with the specified id
func (t *Tree) NearestNeighbors(id uint64, num int) (results []Object, err error) {
	t.mtx.RLock()
	obj, exists := t.objMap[id]
	if !exists {
		t.mtx.RUnlock()
		return []Object{}, ErrInvalidID
	}
	spatials := t.rt.NearestNeighbors(1+num, rtreego.Point{
		obj.rct.PointCoord(0),
		obj.rct.PointCoord(1),
	})
	for i := 0; i < len(spatials); i++ {
		if spatials[i].(*customSpatial).id == id {
			spatials = append(spatials[:i], spatials[i+1:]...)
			break
		}
	}
	results = make([]Object, len(spatials))
	for i := 0; i < len(spatials); i++ {
		results[i] = Object{
			spatials[i].(*customSpatial).id,
			spatials[i].(*customSpatial).rct.PointCoord(0),
			spatials[i].(*customSpatial).rct.PointCoord(1),
		}
	}
	t.mtx.RUnlock()
	if len(results) != num {
		return results, ErrNotEnoughNeighbors
	}
	return results, nil
}
