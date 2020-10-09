package utils

// PortSelector returns a func that yields a new port every time
func PortSelector(beg int) func() int {
	port := beg
	return func() int {
		if port == 0 {
			return 0
		}

		out := port
		port++
		return out
	}
}
