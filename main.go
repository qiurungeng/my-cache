package main

import (
	"flag"
	"fmt"
	"log"
	"mycache"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "哈哈",
}

func main() {
	testDistributedNodes()
}

func testHTTPPool() {
	mycache.NewGroup("scores", 2 << 10, mycache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Printf("[Slow DB search key]: %s", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:9999"
	pool := mycache.NewHTTPPool(addr)
	log.Println("MyCache is running...")
	log.Fatal(http.ListenAndServe(addr, pool))
}
func testDistributedNodes() {
	//测试, 开启三个实例，分别在三个端口上
	//arguments: -port=8001
	//arguments: -port=8002
	//arguments: -port=8003 -api=1

	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "MyCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	myCache := createGroup()
	if api {
		go startAPIServer(apiAddr, myCache)
	}
	startCacheServer(addrMap[port], addrs, myCache)
}



func createGroup() *mycache.Group {
	return mycache.NewGroup("scores", 2<<10, mycache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, myCache *mycache.Group) {
	// 本机cache服务端
	nodes := mycache.NewHTTPPool(addr)
	// 其他cache服务节点
	nodes.SetAllNodes(addrs...)

	myCache.RegisterNodePicker(nodes)
	log.Println("MyCache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], nodes))
}

// 提供API查询服务
func startAPIServer(apiAddr string, myCache *mycache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := myCache.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("frontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}
