package filet

import (
	"sync/atomic"
	"testing"
)

func TestParallelForRunsEveryIndex(t *testing.T) {
	var seen [7]atomic.Int32
	parallelFor(len(seen), 3, func(i int) { seen[i].Add(1) })
	for i := range seen {
		if n := seen[i].Load(); n != 1 {
			t.Fatalf("index %d ran %d times, want once", i, n)
		}
	}
}
