package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var ACTIVEFUTURES int32

type Future[T any] struct {
	function func() T

	finished bool

	result *T

	done chan struct{}
}

func (future *Future[T]) await() T {
	// <-future.done
	for !future.finished {
		time.Sleep(time.Microsecond * 10)
	}

	return *future.result
}

func (future *Future[T]) run() {
	var max int32
	max = 1250
	if ACTIVEFUTURES >= max {
		for ACTIVEFUTURES >= max {
			time.Sleep(time.Millisecond * 10)
			runtime.NumGoroutine()
		}
	}

	atomic.AddInt32(&ACTIVEFUTURES, 1)
	go func() {
		defer atomic.AddInt32(&ACTIVEFUTURES, -1)
		value := future.function()
		future.result = &value
		future.finished = true
		close(future.done)
	}()
}

func (future *Future[T]) then(fn func(T)) {

	// <-future.done
	for !future.finished {
		time.Sleep(time.Microsecond * 10)
	}
	go func() {
		fn(*future.result)
	}()
}

// Creates a pointer to a new future instance and runs its defined function
// Variables used within the function are defined in the closure, make sure to pass pointers
// instead of values if you want to perform an operation on the passed variable
func FUTURE[T any](fn func() T) *Future[T] {
	future := new(Future[T])
	future.finished = false
	future.result = nil
	future.function = fn
	future.done = make(chan struct{})
	future.run()

	return future
}

func async_wait(ms int) *Future[int] {
	return FUTURE(func() int {
		time.Sleep(time.Duration(ms * 1000000))
		return 0
	})
}

func async_load(file_name string) *Future[string] {
	return FUTURE[string](func() string {
		file, err := os.Open(file_name)
		if err != nil {
			return fmt.Sprintf("%+v", err)
		}
		bytes, err := io.ReadAll(file)
		if err != nil {
			return fmt.Sprintf("%+v", err)
		}
		return string(bytes)
	})
}

func async_test(num int) *Future[int] {
	return FUTURE[int](func() int {
		return num
	})
}

func async_GET(URL string) *Future[string] {
	return FUTURE(func() string {
		resp, err := http.Get(URL)
		if err != nil {
			return fmt.Sprintf("%+v", err)
		}
		defer resp.Body.Close()

		buf := make([]byte, 1024)

		_, err = resp.Body.Read(buf)
		if err != nil {
			return fmt.Sprintf("%+v", err)
		}

		return string(buf)
	})
}

func async_fib(n int) *Future[int] {
	return FUTURE(func() int {
		if n <= 1 {
			return n
		}

		return async_fib(n-1).await() + async_fib(n-2).await()
	})
}

func main() {

	var list []*Future[string]

	total_init := time.Now()

	init := time.Now()
	for range 1000 {
		list = append(list, async_GET("https://jsonplaceholder.typicode.com/posts"))
	}
	fin := time.Now()
	fmt.Printf("initializating futures took: %+v\n", fin.Sub(init))

	init = time.Now()
	for _, future := range list {
		future.then(func(e string) {
			fmt.Printf(e)
		})
	}
	fin = time.Now()
	fmt.Printf("awaiting futures took: %+v\n", fin.Sub(init))

	total_fin := time.Now()
	fmt.Printf("total program time: %+v\n", total_fin.Sub(total_init))
}
