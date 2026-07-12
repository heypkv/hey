package svc

import (
	"strconv"
	"strings"
)

// byVersion sorts dotted numeric version strings ascending, comparing segment
// by segment numerically (so "16.4.0" > "9.6.0"), falling back to lexical for
// non-numeric segments.
type byVersion []string

func (b byVersion) Len() int      { return len(b) }
func (b byVersion) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byVersion) Less(i, j int) bool {
	ai := strings.Split(b[i], ".")
	aj := strings.Split(b[j], ".")
	for k := 0; k < len(ai) && k < len(aj); k++ {
		ni, ei := strconv.Atoi(ai[k])
		nj, ej := strconv.Atoi(aj[k])
		if ei == nil && ej == nil {
			if ni != nj {
				return ni < nj
			}
			continue
		}
		if ai[k] != aj[k] {
			return ai[k] < aj[k]
		}
	}
	return len(ai) < len(aj)
}
