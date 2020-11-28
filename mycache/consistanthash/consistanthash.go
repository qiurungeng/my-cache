package consistanthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash map []byte to uint32
type Hash func(data []byte) uint32

type Map struct {
	// Hash 函数
	hash     Hash
	// 虚拟节点倍数
	replicas int
	// 哈希环
	keys     []int
	// 虚拟节点与真实节点的映射表
	hashMap  map[int]string
}

// Constructor
func New(replicas int, hashFunc Hash) *Map {
	if hashFunc == nil {
		hashFunc = crc32.ChecksumIEEE
	}
	return &Map{
		replicas: replicas,
		hashMap: make(map[int]string),
		hash: hashFunc,
	}
}

// 添加真实的 节点 or 机器
func (m *Map) AddNode(keys ...string) {
	// 对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
	for _, key := range keys {
		for i := 0 ; i < m.replicas ; i++ {
			// 虚拟节点的名称: "i"+key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 获取该 key 所映射到的 节点 or 机器
func (m *Map) GetNode(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx % len(m.keys)]]
}

