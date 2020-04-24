package common

import (
	"container/heap"
	"time"
)

// TimerItem Timer Item
type TimerItem struct {
	Value  interface{}
	Expire time.Time
	Index  int // The index of the item in the heap.
}

//TimerQueue implements heap.Interface and holds Items.
type TimerQueue []*TimerItem

// Len TimeQueue의 개수를 반환한다.
func (tq TimerQueue) Len() int { return len(tq) }

// Less 두 값을 비교한다.
func (tq TimerQueue) Less(i, j int) bool {
	return tq[i].Expire.After(tq[j].Expire)
}

// Swap 두 값의 위치를 바꾼다.
func (tq TimerQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].Index = i
	tq[j].Index = j
}

// Push 두 값의 위치를 바꾼다.
func (tq *TimerQueue) Push(x interface{}) {
	n := len(*tq)
	item := x.(*TimerItem)
	item.Index = n
	*tq = append(*tq, item)
}

// Pop Pop
func (tq *TimerQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*tq = old[0 : n-1]
	return item
}

// Update modifies the priority and value of an Item in the queue.
func (tq *TimerQueue) Update(item *TimerItem, value interface{}, expire time.Time) {
	item.Value = value
	item.Expire = expire
	heap.Fix(tq, item.Index)
}
