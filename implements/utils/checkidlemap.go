package utils

import (
	"container/list"
	"sync"
	"time"
)

// cmNode Check Map Node
type cmNode struct {
	lnode      *list.Element // timer node
	key        string        // data key
	value      interface{}   //data
	expireTime int64         // expireTime
}

//CheckIdleMapShard A "thread" safe string to emNode
type CheckIdleMapShard struct {
	items        map[string]*cmNode
	list         *list.List
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// CheckIdleMapIsIdle CheckIdleMap에서 해당 데이터가 현재 시간 기준으로 사용되었는지 확인한다.
type CheckIdleMapIsIdle func(key string, value interface{}, expireTime *time.Time, unixNano int64) bool

// CheckIdleMap Data Check 기능이 있는 Map
type CheckIdleMap struct {
	IdleTimeout time.Duration
	shardCount  int
	shards      []*CheckIdleMapShard
	ticker      *time.Ticker
	isIdle      CheckIdleMapIsIdle
}

// NewCheckIdleMap 해당 주기로 데이터를 체크하는 기능이 있는 Map을 생성한다.
// idleTimeout 데이터가 IDLE이 되는 시간을 지정한다.
// checkPeriod 데이터를 확인하는 주기 값이다.
// shardCount map 동시성 접근 효율성을 위한 key sharding 개수 값이다.
// isIdle 전달된 시간 이후 사용된 적이 없는지 여부를 반환하는 함수
func NewCheckIdleMap(idleTimeout, checkPeriod time.Duration, shardCount int, isIdle CheckIdleMapIsIdle) *CheckIdleMap {
	if shardCount <= 0 {
		shardCount = 32
	}
	if idleTimeout <= 0 {
		idleTimeout = 3 * time.Minute
	}
	if checkPeriod <= 0 {
		checkPeriod = 1 * time.Second
	}
	r := &CheckIdleMap{
		IdleTimeout: idleTimeout,
		shardCount:  shardCount,
		shards:      make([]*CheckIdleMapShard, shardCount),
		ticker:      time.NewTicker(checkPeriod),
		isIdle:      isIdle,
	}
	for i := 0; i < shardCount; i++ {
		r.shards[i] = &CheckIdleMapShard{
			items: make(map[string]*cmNode),
			list:  list.New(),
		}
	}

	go r.handleTimer()
	return r
}

// Close IpxMemCache 사용을 종료한다.
func (m *CheckIdleMap) Close() {
	m.ticker.Stop()
}

// GetShard Returns shard under given key
func (m *CheckIdleMap) GetShard(key string) *CheckIdleMapShard {
	return m.shards[uint(Hash32ForMap(key))%uint(m.shardCount)]
}

// Count 전달된 Expire Map내 등록된 Item의 개수를 반환한다.
func (m *CheckIdleMap) Count() int {
	count := 0
	for _, shard := range m.shards {
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Get 전달된 Key의 value를 반환한다.
func (m *CheckIdleMap) Get(key string) (interface{}, bool) {
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
func (m *CheckIdleMap) Set(key string, value interface{}) bool {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if ok {
		node.value = value
		return true
	}
	node = &cmNode{
		key:   key,
		value: value,
	}
	shard.items[key] = node
	node.lnode = shard.list.PushBack(node)
	node.expireTime = time.Now().Add(m.IdleTimeout).UnixNano()
	return true
}

// SetIfAbset 전달된 값을 저장 하고 지정된 시간 후 expire 되도록 설정한다.
func (m *CheckIdleMap) SetIfAbset(key string, value interface{}) bool {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if ok {
		return false
	}
	node = &cmNode{
		key:   key,
		value: value,
	}
	shard.items[key] = node
	node.lnode = shard.list.PushBack(node)
	return true
}

//MSet 전달된 Map을 모두 등록한다.
func (m *CheckIdleMap) MSet(data map[string]interface{}) {
	for key, value := range data {
		m.Set(key, value)
	}
}

// Remove 전달된 Key의 값을 삭제한다.
func (m *CheckIdleMap) Remove(key string) interface{} {
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	node, ok := shard.items[key]
	if !ok {
		return nil
	}
	delete(shard.items, key)
	shard.list.Remove(node.lnode)
	return node.value
}

// RemoveAll 전체 Map을 삭제한다.
func (m *CheckIdleMap) RemoveAll() {
	for _, shard := range m.shards {
		shard.Lock()
		for i := shard.list.Len(); i > 0; i-- {
			e := shard.list.Front()
			if e == nil {
				break
			}
			node := e.Value.(*cmNode)
			shard.list.Remove(e)
			e.Value = nil
			delete(shard.items, node.key)
		}
		shard.Unlock()
	}
}

func (m *CheckIdleMap) expire(timeout *time.Time) {
	var node *cmNode
	tonano := timeout.UnixNano()
	for _, shard := range m.shards {
		shard.RLock()
		len := shard.list.Len()
		shard.RUnlock()
		for i := len; i > 0; i-- {
			shard.RLock()
			e := shard.list.Front()
			if e == nil {
				node = nil
			} else {
				node = e.Value.(*cmNode)
			}
			shard.RUnlock()
			if node == nil || node.expireTime > tonano {
				break
			}
			var isIdle bool
			if m.isIdle != nil {
				isIdle = m.isIdle(node.key, node.value, timeout, tonano)
			} else {
				isIdle = true
			}
			shard.Lock()
			if isIdle {
				shard.list.Remove(node.lnode)
				delete(shard.items, node.key)
			} else {
				node.expireTime = tonano + m.IdleTimeout.Nanoseconds()
				shard.list.MoveToBack(node.lnode)
			}
			shard.Unlock()
		}
	}
}

// handleTimer timer에 node를 등록하거나 expire 시킨다.
func (m *CheckIdleMap) handleTimer() {
	for t := range m.ticker.C {
		timeout := t.Add(-m.IdleTimeout)
		m.expire(&timeout)
	}
	m.RemoveAll()
}
