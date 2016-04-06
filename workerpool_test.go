package workerpool

import (
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestNegWorkers(t *testing.T) {
	jobChannel := make(chan Job)
	InitNewPool(-1, jobChannel)

	n := int64(runtime.NumCPU())
	var backSlot int64
	var job = JobFunc(func() {
		atomic.AddInt64(&backSlot, 1)
		select {}
	})
OUT1:
	for {
		select {
		case jobChannel <- job:
		case <-time.After(time.Millisecond * 300):
			break OUT1
		}
	}

	actual := atomic.LoadInt64(&backSlot)
	if actual != n {
		t.Log(actual)
		t.Fail()
	}
}

func TestZeroWorkers(t *testing.T) {
	jobChannel := make(chan Job)
	pool := InitNewPool(0, jobChannel)

	var backSlot int64 = 10
	var job = JobFunc(func() {
		atomic.StoreInt64(&backSlot, 110)
	})

	select {
	case jobChannel <- job:
	case <-time.After(time.Millisecond * 300):
	}
	if atomic.LoadInt64(&backSlot) != 10 {
		t.Fail()
	}

	pool.Expand(1, 0, make(chan bool))

	done := make(chan bool)
	job = func() {
		defer close(done)
		atomic.StoreInt64(&backSlot, 73)
	}
	select {
	case jobChannel <- job:
	case <-time.After(time.Millisecond * 300):
		t.Fail()
	}
	<-done

	if atomic.LoadInt64(&backSlot) != 73 {
		t.Fail()
	}
}

func TestAbsoluteTimeout(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := InitNewPool(initialWorkers, jobChannel)

	quit1 := make(chan bool)
	pool.Expand(extraWorkers, 0, quit1)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + extraWorkers + initialWorkers + dispatcherGoroutine
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}

	done := make(chan bool)
	absoluteTimeout := func() {
		defer close(done)
		<-time.After(time.Millisecond * 100)
		close(quit1)
	}

	go absoluteTimeout()
	<-done
	<-time.After(time.Millisecond * 400)
	runtime.GC()

	afterGoroutines = runtime.NumGoroutine()
	thenGoroutines = startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func TestTimeout(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := InitNewPool(initialWorkers, jobChannel)

	pool.Expand(extraWorkers, time.Millisecond*10, nil)

	<-time.After(time.Millisecond * 100)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func TestQuit(t *testing.T) {
	dispatcherGoroutine := 1
	initialWorkers := 1
	extraWorkers := 10
	startedWith := runtime.NumGoroutine()

	jobChannel := make(chan Job, 2)

	pool := InitNewPool(initialWorkers, jobChannel)

	quit1 := make(chan bool)
	pool.Expand(extraWorkers, 0, quit1)

	afterGoroutines := runtime.NumGoroutine()
	thenGoroutines := startedWith + extraWorkers + initialWorkers + dispatcherGoroutine
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}

	close(quit1)
	<-time.After(time.Millisecond * 100)

	afterGoroutines = runtime.NumGoroutine()
	thenGoroutines = startedWith + initialWorkers + dispatcherGoroutine // no extraWorkers
	if maxDiff(afterGoroutines, thenGoroutines, 1) {
		t.Log(afterGoroutines, thenGoroutines)
		t.Fail()
	}
}

func maxDiff(fst, snd, diff int) bool {
	return math.Abs(float64(fst)-float64(snd)) > float64(diff)
}
