package singleflight

import "sync"

type call struct {
	waitGroup sync.WaitGroup
	val       interface{}
	err       error
}

type Group struct {
	mu      sync.Mutex // 保护 Map callMap
	callMap map[string]*call
}


//Do 的作用就是，针对相同的 key，无论 Do 被并发调用多少次，
//函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值
//或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.callMap == nil {
		g.callMap = make(map[string]*call)
	}
	if c, ok := g.callMap[key]; ok{
		g.mu.Unlock()
		c.waitGroup.Wait()   // 如果请求正在进行中，则阻塞等待直到锁被释放
		return c.val, c.err  // 请求结束，返回结果
	}

	c := new(call)
	c.waitGroup.Add(1) // 请求前加锁, 锁:+1
	g.callMap[key] = c 		 // 添加到callMap, 表明key已有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn()		 // 调用请求
	c.waitGroup.Done()		 // 请求结束, 锁:-1

	//缓存值的存储都放在 LRU 中，其他地方不保存数据。如果不删除，占用内存，且不会淘汰。
	g.mu.Lock()
	delete(g.callMap, key)
	g.mu.Unlock()

	return c.val, c.err
}