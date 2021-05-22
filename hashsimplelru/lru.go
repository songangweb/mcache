package hashSimplelru

import (
	"container/list"
	"errors"
	"time"
)
// EvictCallback is used to get a callback when a cache entry is evicted
// EvictCallback 用于在缓存条目被淘汰时的回调函数
type EvictCallback func(key interface{}, value *interface{}, expirationTime int64)

// LRU implements a non-thread safe fixed size LRU cache
// LRU 实现一个非线程安全的固定大小的LRU缓存
type LRU struct {
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
	onEvict   EvictCallback
}

// entry is used to hold a value in the evictList
// 缓存详细信息
type entry struct {
	key            interface{}
	value          *interface{}
	reference      int64
	expirationTime int64
}

// NewLRU constructs an LRU of the given size
// NewLRU 构造一个给定大小的LRU
func NewLRU(size int, onEvict EvictCallback) (*LRU, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
		onEvict:   onEvict,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
// Purge 用于完全清除缓存
func (c *LRU) Purge() {
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.(*entry).value, v.Value.(*entry).expirationTime)
		}
		delete(c.items, k)
	}
	c.evictList.Init()
}

// PurgeOverdue is used to completely clear the overdue cache.
// PurgeOverdue 清除缓存
func (c *LRU) PurgeOverdue() {
	for _, ent := range c.items {
		c.removeElement(ent)
	}
	c.evictList.Init()
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
// Add 向缓存添加一个值。如果已经存在,则更新信息
func (c *LRU) Add(key interface{}, value *interface{}, expirationTime int64) (ok bool) {
	// 判断缓存中是否已经存在数据,如果已经存在则更新数据
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*entry).value = value
		ent.Value.(*entry).expirationTime = expirationTime
		return true
	}
	// 判断缓存条数是否已经达到限制
	if c.evictList.Len() >= c.size {
		// 判断是否删除成功
		if c.removeOldest() {
			// 创建数据
			ent := &entry{key, value, 0, expirationTime}
			entry := c.evictList.PushFront(ent)
			c.items[key] = entry
			return true
		}
		return false
	}

	// 创建数据
	ent := &entry{key, value, 0, expirationTime}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry
	return true
}

// Get looks up a key's value from the cache.
// Get 从缓存中查找一个键的值。
func (c *LRU) Get(key interface{}) (value *interface{}, expirationTime int64, ok bool) {
	// 判断缓存是否存在
	if ent, ok := c.items[key]; ok {
		// 判断此值是否已经超时,如果超时则进行删除
		if checkExpirationTime(ent.Value.(*entry).expirationTime) {
			c.removeElement(ent)
			return nil, 0, false
		}
		// 数据移到头部
		c.evictList.MoveToFront(ent)
		if ent.Value.(*entry) == nil {
			return nil, 0, false
		}

		// 引用+1
		ent.Value.(*entry).reference++

		//fmt.Println("ent.Value.(*entry): ", ent.Value.(*entry))
		return ent.Value.(*entry).value, ent.Value.(*entry).expirationTime, true
	}
	return nil, 0, false
}

// Release 缓存reference - 1 与 获取数据的方法 对应使用,  当reference为0时,数据才可以被真删
func (c *LRU) Release(key interface{}) {
	// 判断缓存是否存在
	ent, ok := c.items[key]
	if ok {
		if ent.Value.(*entry).reference > 0 {
			ent.Value.(*entry).reference--
		}
	}
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
// Contains 检查某个键是否在缓存中，但不更新缓存的状态
func (c *LRU) Contains(key interface{}) (ok bool) {
	ent, ok := c.items[key]
	if ok {
		// 判断此值是否已经超时,如果超时则进行删除
		if checkExpirationTime(ent.Value.(*entry).expirationTime) {
			c.removeElement(ent)
			return !ok
		}
	}
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
// Peek 在不更新的情况下返回键值(如果没有找到则返回false),不更新缓存的状态
func (c *LRU) Peek(key interface{}) (value *interface{}, expirationTime int64, ok bool) {
	var ent *list.Element
	if ent, ok = c.items[key]; ok {
		// 判断是否已经超时
		if checkExpirationTime(ent.Value.(*entry).expirationTime) {
			c.removeElement(ent)
			return nil, 0, ok
		}
		return ent.Value.(*entry).value, ent.Value.(*entry).expirationTime, true
	}
	return nil, 0, ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
// Remove 从缓存中移除提供的键
func (c *LRU) Remove(key interface{}) (ok bool) {
	if ent, ok := c.items[key]; ok {
		return c.removeElement(ent)
	}
	return ok
}

// RemoveOldest removes the oldest item from the cache.
// RemoveOldest 从缓存中移除最老的项
func (c *LRU) RemoveOldest() (key interface{}, value *interface{}, expirationTime int64, ok bool) {
	ent := c.evictList.Back()
	return c.removeEnt(ent)
}

// GetOldest returns the oldest entry
// GetOldest 返回最老的条目
func (c *LRU) GetOldest() (key interface{}, value *interface{}, expirationTime int64, ok bool) {
	ent := c.evictList.Back()
	return c.getRecursionEnt(ent)
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
// Keys 返回缓存的切片，从最老的到最新的。
func (c *LRU) Keys() []interface{} {
	keys := make([]interface{}, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*entry).key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
// Len 返回缓存中的条数
func (c *LRU) Len() int {
	return c.evictList.Len()
}

// Resize changes the cache size.
// Resize 改变缓存大小。
func (c *LRU) Resize(size int) (evicted int) {
	diff := c.Len() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.size = size
	return diff
}

// removeOldest removes the oldest item from the cache.
// removeOldest 从缓存中移除最老的项。
func (c *LRU) removeOldest() bool {
	// 循环判断是否元素被引用,如果未被引用才可以删除
	ent := c.evictList.Back()
	_, _, _, ok := c.removeEnt(ent)
	return ok
}

// 递归判断删除reference为0 ent
func (c *LRU) removeEnt(e *list.Element) (key interface{}, value *interface{}, expirationTime int64, ok bool) {
	if e.Value.(*entry).reference == 0 {
		// 判断数据是否过期了, 如果过期了,则这种数据不应该属于 删除最老的项
		if checkExpirationTime(e.Value.(*entry).expirationTime) {
			c.removeElement(e)
			return c.removeEnt(e.Prev())
		}
		// 删除此节点数据
		c.evictList.Remove(e)
		delete(c.items, e.Value.(*entry).key)
		if c.onEvict != nil {
			c.onEvict(e.Value.(*entry).key, e.Value.(*entry).value, e.Value.(*entry).expirationTime)
		}
		return e.Value.(*entry).key, e.Value.(*entry).value, e.Value.(*entry).expirationTime, true
	}
	if e.Prev() == nil {
		return nil, nil, 0, false
	}
	return c.removeEnt(e.Prev())
}

// removeElement is used to remove a given list element from the cache
// removeElement 从缓存中移除一个列表元素
func (c *LRU) removeElement(e *list.Element) bool {
	if e.Value.(*entry).reference == 0 {
		c.evictList.Remove(e)
		delete(c.items, e.Value.(*entry).key)
		if c.onEvict != nil {
			c.onEvict(e.Value.(*entry).key, e.Value.(*entry).value, e.Value.(*entry).expirationTime)
		}
		return true
	}
	return false
}


// getOldest 从缓存中获取最老的项。
func (c *LRU) getOldest() (key interface{}, value *interface{}, expirationTime int64, ok bool) {
	// 循环判断是否元素被引用,如果未被引用才可以删除
	ent := c.evictList.Back()
	return c.getRecursionEnt(ent)
}

// 递归判断获取ent
func (c *LRU) getRecursionEnt(e *list.Element) (key interface{}, value *interface{}, expirationTime int64, ok bool) {
	// 判断数据是否过期了, 如果过期了,则清理垃圾数据
	if checkExpirationTime(e.Value.(*entry).expirationTime) {
		c.removeElement(e)

		if e.Prev() == nil {
			return nil, nil, 0, false
		}
		return c.getRecursionEnt(e.Prev())
	}

	e.Value.(*entry).reference++
	return e.Value.(*entry).key, e.Value.(*entry).value, e.Value.(*entry).expirationTime, true
}

// checkExpirationTime is Determine if the cache has expired
// checkExpirationTime 判断缓存是否已经过期
func checkExpirationTime(expirationTime int64) (ok bool) {
	if 0 != expirationTime && expirationTime <= time.Now().UnixNano()/1e6 {
		return true
	}
	return false
}
