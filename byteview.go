package my_cache

// 缓存值的抽象与封装
type ByteView struct {
	b []byte
}

func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice returns a copy of the data as a byte slice.
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}


func cloneBytes(b []byte) []byte{
	bytes := make([]byte, len(b))
	copy(bytes, b)
	return bytes
}