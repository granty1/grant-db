package customrand

//go:linkname Uint32 runtime.fastrand
func Uint32() uint32

// Uint32N returns, as an uint32, a pseudo-random number in [0,n).
func Uint32N(n uint32) uint32 {
	if n&(n-1) == 0 { // n is power of two, can mask
		return Uint32() & (n - 1)
	}
	return Uint32() % n
}

func Buf(size int) []byte {
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = byte(Uint32N(127))
		if buf[i] == 0 || buf[i] == byte('$') {
			buf[i]++
		}
	}
	return buf
}
