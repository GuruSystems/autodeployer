package main

type LineReader struct {
	buf []byte
}

// returns a string or nil of no full line atm
func (lr *LineReader) gotBytes(a []byte, lenctr int) string {
	lr.buf = append(lr.buf, a[:lenctr]...)
	for idx, c := range lr.buf {
		if c == '\n' {
			s := string(lr.buf[:idx])
			nidx := idx + 1
			if nidx >= len(lr.buf) {
				lr.buf = []byte{}
			} else {
				lr.buf = lr.buf[nidx:]
			}
			return s
		}
	}
	return ""

}
