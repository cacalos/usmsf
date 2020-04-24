package utils

import (
	"container/heap"
	"time"
)

// TimerHeapNode Timer를 위한 Heap Node
type TimerHeapNode struct {
	idx      int         // Heap index
	UnixNano int64       // UnixNano Time Value
	Value    interface{} // 관련 Value
}

//TimerHeap Timer를 위한 Heap
type TimerHeap struct {
	nodes []*TimerHeapNode
}

////////////////////////////////////////////////////////////////////////////////
// TimerHeapNode functions
////////////////////////////////////////////////////////////////////////////////

// NewTimerHeapNode 새로운 Timer Heap Node를 생성한다.
func NewTimerHeapNode() *TimerHeapNode {
	return &TimerHeapNode{}
}

// NewTimerHeapNodeV 새로운 Timer Heap Node를 생성한다.
func NewTimerHeapNodeV(t time.Time, v interface{}) *TimerHeapNode {
	return &TimerHeapNode{
		UnixNano: t.UnixNano(),
		Value:    v,
	}
}

// GetIndex 전달된 TimerHeapNode의 Heap 내 index 값을 반환한다.
func (n *TimerHeapNode) GetIndex() int {
	return n.idx
}

// SetTime TimerHeapNode에 키가 될 시간을 설정한다.
func (n *TimerHeapNode) SetTime(t time.Time) {
	n.UnixNano = t.UnixNano()
}

// GetTime TimerHeapNode에 설정된 시간을 가져온다.
func (n *TimerHeapNode) GetTime() time.Time {
	return time.Unix(0, n.UnixNano)
}

////////////////////////////////////////////////////////////////////////////////
// TimerHeap functions
////////////////////////////////////////////////////////////////////////////////

// NewTimerHeap 새로운 Timer Heap을 생성해 반환한다.
func NewTimerHeap() *TimerHeap {
	r := &TimerHeap{
		nodes: []*TimerHeapNode{},
	}
	heap.Init(r)
	return r
}

// Head 전달된 Heap의 첫번째 Node를 반환한다.
func (h *TimerHeap) Head() *TimerHeapNode {
	if len(h.nodes) <= 0 {
		return nil
	}
	return h.nodes[0]
}

// Tail 전달된 Heap의 마지막 Node를 반환한다.
func (h *TimerHeap) Tail() *TimerHeapNode {
	if len(h.nodes) <= 0 {
		return nil
	}
	return h.nodes[len(h.nodes)-1]
}

//FixIndex 전달된 위치의 시간이 바뀌었으니 Heap을 다시 정렬한다.
func (h *TimerHeap) FixIndex(i int) {
	heap.Fix(h, i)
}

//FixNode 전달된 Node가 변경 되었으니 Heap을 다시 정렬
func (h *TimerHeap) FixNode(n *TimerHeapNode) {
	heap.Fix(h, n.idx)
}

//PopNode 최상위 Node를 꺼낸다.
func (h *TimerHeap) PopNode() *TimerHeapNode {
	return heap.Pop(h).(*TimerHeapNode)
}

//PushNode Node를 새로 추가한다.
func (h *TimerHeap) PushNode(v *TimerHeapNode) {
	heap.Push(h, v)
}

//RemoveIndex 전달된 위치의 Node를 삭제한다.
func (h *TimerHeap) RemoveIndex(idx int) *TimerHeapNode {
	return heap.Remove(h, idx).(*TimerHeapNode)
}

//RemoveNode 전달된 Node를 삭제한다.
func (h *TimerHeap) RemoveNode(n *TimerHeapNode) {
	heap.Remove(h, n.idx)
}

// Len Heap에 등록된 Node의 개수를 반환한다.
func (h *TimerHeap) Len() int {
	return len(h.nodes)
}

// Less Heap 내 두 위치의 값을 비교한다. - heap.Interface 구현
func (h *TimerHeap) Less(i, j int) bool {
	return h.nodes[i].UnixNano > h.nodes[j].UnixNano
}

// Swap Heap 내 두 위치의 값을 바꾼다. - heap.Interface 구현
func (h *TimerHeap) Swap(i, j int) {
	h.nodes[i], h.nodes[j] = h.nodes[j], h.nodes[i]
	h.nodes[i].idx = i
	h.nodes[j].idx = j
}

// Push 전달된 값을 Heap에 추가한다. - heap.Interface 구현
func (h *TimerHeap) Push(x interface{}) {
	n := x.(*TimerHeapNode)
	n.idx = len(h.nodes)
	h.nodes = append(h.nodes, n)
}

// Pop 최상위의 값을 꺼낸다. - heap.Interface 구현
func (h *TimerHeap) Pop() interface{} {
	old := h.nodes
	n := len(old)
	x := old[n-1]
	x.idx = -1
	h.nodes = old[0 : n-1]
	return x
}
