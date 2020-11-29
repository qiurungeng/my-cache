package mycache

import (
	"fmt"
	"io/ioutil"
	"log"
	"mycache/consistenthash"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_myCache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex             // guards nodeMap and httpGetters
	nodeMap     *consistenthash.Map    // 一致性哈希Map，包含远程 Group 的名称, 用来根据具体的 key 选择对应节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的 httpGetter, key 为地址如: "http://10.0.0.2:8008"
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}


func (p *HTTPPool) Log(format string, value ...interface{}) {
	log.Printf("[Server %s]: %s", p.self, fmt.Sprintf(format, value...))
}


func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serve unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(view.ByteSlice())
}


// SetAllNodes updates the pool's list of nodeMap
func (p *HTTPPool) SetAllNodes(nodeAddrs ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodeMap = consistenthash.New(defaultReplicas, nil)
	p.nodeMap.AddNodes(nodeAddrs...)
	p.httpGetters = make(map[string]*httpGetter, len(nodeAddrs))
	for _, nodeAddr := range nodeAddrs {
		p.httpGetters[nodeAddr] = &httpGetter{baseURL: nodeAddr + p.basePath}
	}
}


// 由 key 通过一致性哈希获取 对应的缓存节点
func (p *HTTPPool) PickNode(key string) (NodeGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	node := p.nodeMap.GetNode(key)
	if node != "" && node != p.self {
		p.Log("Pick Node %s", node)
		return p.httpGetters[node], true
	}
	return nil, false
}


var _ NodePicker = (*HTTPPool)(nil)

// 远程节点对应的 Getter, 相当于远程客户端实体
type httpGetter struct {
	baseURL string
}

// 从远程 httpGetter(客户端) 通过 http 查询相应 group 的 kv
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	queryPath := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)

	res, err := http.Get(queryPath)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ NodeGetter = (*httpGetter)(nil)