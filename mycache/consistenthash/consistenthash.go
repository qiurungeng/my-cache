package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash map []byte to uint32
type Hash func(data []byte) uint32

// 一致性HashMap, 提供一种为每个 key 获取它所属的特定 节点key 的服务
// 比如我们以集群中所有机器地址Addr为节点key, 然后AddNodes(All Addr), 那现在
// 我们想查询一个值k的value, 通过Get(k)就可获得我们需要查询的具体机器的地址Addr
// AddNodes(nodeKeys ...string) 批量添加节点key, 按照一定倍数散列到哈希环中
// GetNode(key string) string	由 key 获取它在哈希环中顺时针最近的节点key
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
func (m *Map) AddNodes(nodeKeys ...string) {
	// 对每一个真实节点 nodeKey，对应创建 m.replicas 个虚拟节点
	for _, nodeKey := range nodeKeys {
		for i := 0 ; i < m.replicas ; i++ {
			// 虚拟节点的名称: "i"+nodeKey
			hash := int(m.hash([]byte(strconv.Itoa(i) + nodeKey)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = nodeKey
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

