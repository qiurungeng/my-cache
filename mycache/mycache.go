package mycache

import (
	"fmt"
	"log"
	"mycache/singleflight"
	"sync"
)

// Getter: 用来从数据源获取 kv
type Getter interface {
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
// 函数类型实现某一个接口，称之为接口型函数，
// 方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
type GetterFunc func(key string)([]byte, error)
func (gf GetterFunc) Get(key string) ([]byte, error) {
	return gf(key)
}

// Group 是 GeeCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程。
// Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name
type Group struct {
	name      string
	getter    Getter	// 回调 Getter，在缓存不存在时，调用这个函数，得到源数据。
	mainCache cache
	nodes	  NodePicker
	loader	  *singleflight.Group
}

var (
	mu sync.Mutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}

	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.Lock()
	g := groups[name]
	defer mu.Unlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 从主存中获取
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[MyCache] hit")
		return v, nil
	}
	// 从数据源载入
	return g.load(key)
}

// 从数据源载入
func (g *Group) load(key string) (value ByteView, err error) {
	result, err := g.loader.Do(key, func() (interface{}, error) {
		if g.nodes != nil {
			if node, ok := g.nodes.PickNode(key); ok{
				if value, err := g.getFromNode(node, key); err == nil {
					return value, err
				}
				log.Println("[MyCache Fail to Get From Node]", err)
			}
		}

		// 本地数据源
		return g.getLocally(key)
	})
	if err == nil {
		return result.(ByteView), err
	}
	return
}

// 从本地数据源获取
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 添加到缓存中
	g.populateCache(key, value)
	return value, nil
}

// 数据源中获取的数据添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 注册一个远程节点选择器
func (g *Group) RegisterNodePicker(nodes NodePicker) {
	if g.nodes != nil {
		panic("RegisterNodePicker caller more than once")
	}
	g.nodes = nodes
}

// 访问远程节点，获取包装好的缓存值。
func (g *Group) getFromNode(node NodeGetter, key string) (ByteView, error) {
	bytes, err := node.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, err
}