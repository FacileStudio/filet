package filet

import (
	"runtime"
	"sync"
)

// checkFilesParallel runs checkFileCached over every file with a bounded worker
// pool, bucketing each result to its own index so the order matches Scan and
// the cache store stays coherent under lock.
func checkFilesParallel(w *cacheWorker, cfg *Config, files []SourceFile) []Finding {
	buckets := make([][]Finding, len(files))
	parallelFor(len(files), workerCount(len(files)), func(i int) {
		buckets[i] = checkFileCached(w, cfg, files[i])
	})
	return joinFindings(buckets)
}

// joinFindings concatenates index-bucketed findings in order.
func joinFindings(buckets [][]Finding) []Finding {
	var out []Finding
	for _, b := range buckets {
		out = append(out, b...)
	}
	return out
}

// workerCount bounds a worker pool to the number of logical CPUs, and to at
// most one goroutine per job so tiny runs do not spin up idle workers.
func workerCount(n int) int {
	c := runtime.NumCPU()
	if c < 1 {
		return 1
	}
	if c > n {
		return n
	}
	return c
}

// parallelFor runs fn(i) for every i in [0, n) with a bounded worker pool and
// waits for all of them before returning.
func parallelFor(n, workers int, fn func(int)) {
	if n <= 0 || workers <= 0 {
		return
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				fn(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}
