package common

import (
	"sync"
	"time"
)

// emNode Expire Map Node
type emNode struct {
	tmnode TimerHeapNode // timer node
	key    string        // data key
	value  interface{}   // data value
}

//ExpireMapShard A "thread" safe string to emNode
type ExpireMapShard struct {
	items        map[string]*emNode
	timerheap    *TimerHeap
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// ExpireMap Data Expire 기능이 있는 Map
type ExpireMap struct {
	shardCount int
	shards     []*ExpireMapShard
	ticker     *time.Ticker
	Expire     func(key string, value interface{}, now *time.Time)
}

// NewExpireMap Data Time Expire 기능이 있는 Map을 생성한다.
// chcktime은 expire 되었는지 확인하는 주기 값이다.
func NewExpireMap(shardCount int, chcktime time.Duration) *ExpireMap {
	if shardCount <= 0 {
		shardCount = 32
	}
	if chcktime <= 0 {
		chcktime = 1 * time.Second
	}
	r := &ExpireMap{
		shardCount: shardCount,
		shards:     make([]*ExpireMapShard, shardCount),
		ticker:     time.NewTicker(chcktime),
	}
	for i := 0; i < shardCount; i++ {
		r.shards[i] = &ExpireMapShard{
			items:     make(map[string]*emNode),
			timerheap: NewTimerHeap(),
		}
	}

	go r.handleTimer()
	return r
}

// Close ExpireMap 사용을 종료한다.
func (m *ExpireMap) Close() {
	m.ticker.Stop()
}

// GetShard Returns shard under given key
func (m *ExpireMap) GetShard(key string) *ExpireMapShard {
	return m.shards[uint(m.getHash(key))%uint(m.shardCount)]
}

// Count 전달된 Expire Map내 등록된 Item의 개수를 반환한다.
func (m *ExpireMap) Count() int {
	count := 0
	for _, shard := range m.shards {
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// GetAndUpdate 전달된 Key의 value를 반환하면, 만료시간도 전달된 duration 만큼 update 한다.
func (m *ExpireMap) GetAndUpdate(key string, duration time.Duration) (interface{}, bool) {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if !ok {
		return nil, false
	}
	node.tmnode.UnixNano = time.Now().Add(duration).UnixNano()
	shard.timerheap.FixNode(&node.tmnode)
	return node.value, true
}

// Get 전달된 Key의 value를 반환한다.
func (m *ExpireMap) Get(key string) (interface{}, bool) {
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()
	node, ok := shard.items[key]
	if !ok {
		return nil, false
	}
	return node.value, true
}

// Set 전달된 값을 저장 하고 지정된 시간 후 expire 되도록 설정한다.
func (m *ExpireMap) Set(key string, value interface{}, expire time.Time) bool {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if ok {
		node.value = value
		node.tmnode.UnixNano = expire.UnixNano()
		shard.timerheap.FixNode(&node.tmnode)
		return true
	}
	node = &emNode{
		key:   key,
		value: value,
	}
	shard.items[key] = node
	node.tmnode.UnixNano = expire.UnixNano()
	node.tmnode.Value = node
	shard.timerheap.PushNode(&node.tmnode)
	return true
}

// SetIfAbsent 기존에 값이 없을 경우에 전달된 값을 저장 하고 지정된 시간 후 expire 되도록 설정한다.
func (m *ExpireMap) SetIfAbsent(key string, value interface{}, expire time.Time) bool {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if ok {
		return false
	}
	node = &emNode{
		key:   key,
		value: value,
	}
	shard.items[key] = node
	node.tmnode.UnixNano = expire.UnixNano()
	node.tmnode.Value = node
	shard.timerheap.PushNode(&node.tmnode)
	return true
}

//MSet 전달된 Map을 모두 등록한다.
func (m *ExpireMap) MSet(data map[string]interface{}, expire time.Time) {
	for key, value := range data {
		m.Set(key, value, expire)
	}
}

// Remove 전달된 Key의 값을 삭제한다.
func (m *ExpireMap) Remove(key string) interface{} {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if !ok {
		return nil
	}
	delete(shard.items, key)
	shard.timerheap.RemoveNode(&node.tmnode)
	return node.value
}

// RemoveAll 등록된 전체 요소들을 삭제한다.
func (m *ExpireMap) RemoveAll() {
	for _, shard := range m.shards {
		shard.Lock()
		for i := 0; i < shard.timerheap.Len(); i++ {
			head := shard.timerheap.Head()
			if head == nil {
				break
			}
			shard.timerheap.PopNode()
			node := head.Value.(*emNode)
			head.Value = nil
			delete(shard.items, node.key)
		}
		shard.Unlock()
	}
}

func (m *ExpireMap) expire(t *time.Time) {
	now := t.UnixNano()
	for _, shard := range m.shards {
		shard.Lock()
		for i := shard.timerheap.Len(); i > 0; i-- {
			head := shard.timerheap.Head()
			if head == nil || head.UnixNano >= now {
				break
			}
			shard.timerheap.PopNode()
			node := head.Value.(*emNode)
			head.Value = nil
			delete(shard.items, node.key)
			shard.Unlock()
			if m.Expire != nil {
				m.Expire(node.key, node.value, t)
			}
			shard.Lock()
		}
		shard.Unlock()
	}
}

// handleTimer timer에 node를 등록하거나 expire 시킨다.
func (m *ExpireMap) handleTimer() {
	for t := range m.ticker.C {
		m.expire(&t)
	}
	m.RemoveAll()
}

func (m *ExpireMap) getHash(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
