package main

import (
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


