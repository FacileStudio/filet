package fixture

import (
	"context"
	"net/http"
	"sync"
)

func notify(string) {}

func work(context.Context) {}

func detached(w http.ResponseWriter, r *http.Request) {
	go notify(r.URL.Path) // want
	go func() {           // want
		notify("done")
	}()
	go work(context.Background()) // want
	w.WriteHeader(http.StatusAccepted)
}

func owned(w http.ResponseWriter, r *http.Request) {
	go work(r.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		notify("x")
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		notify("y")
	}()
	wg.Wait()
	<-done
	w.WriteHeader(http.StatusOK)
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		go notify(r.Method) // want
		next.ServeHTTP(w, r)
	})
}

func notAHandler() {
	go notify("background job")
}
