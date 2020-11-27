package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access
type Cache struct {
	maxBytes 	int64
	nBytes 		int64
	list 		*list.List
	cache 		map[string]*list.Element
	// 某条记录被移除时的回调函数
	OnEvicted 	func(key string, value Value)
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

type entry struct {
	key string
	value Value
}


// Constructor
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		nBytes: 0,
		list: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查询
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		c.list.MoveToFront(element)
		kv := element.Value.(*entry)
		return kv.value, ok
	}
	return
}

// 淘汰最老的
func (c *Cache) RemoveOldest() {
	element := c.list.Back()
	if element != nil {
		c.list.Remove(element)
		kv := element.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增 & 修改
func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		c.list.MoveToFront(element)
		kv := element.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		element = c.list.PushFront(&entry{key, value})
		c.cache[key] = element
		c.nBytes += int64(len(key)) + int64(value.Len())
	}

	// 超出了缓存容量限制，则要淘汰掉最老的
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.list.Len()
}