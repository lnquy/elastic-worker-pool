package ewp

import (
	"testing"
	"time"
)

func TestAtomicBool_Single(t *testing.T) {
	ab := &AtomicBool{}
	if b := ab.Get(); b {
		t.Fatalf("1. Expected init false. Got %v", b)
	}
	ab.Set(true)
	if b := ab.Get(); !b {
		t.Fatalf("2. Expected set true. Got %v", b)
	}
	ab.Set(false)
	if b := ab.Get(); b {
		t.Fatalf("2. Expected set false. Got %v", b)
	}
}

var _tmpAB bool

func TestAtomicBool_Concurrent(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("1. Expected no race condition, caught: %v", err)
		}
	}()

	ab, workerNum, runNum := &AtomicBool{}, 4, 10000000

	for i := 0; i < workerNum; i++ {
		go func() {
			for j := 0; j < runNum; j++ {
				if j%2 == 0 {
					ab.Set(true)
				} else {
					ab.Set(false)
				}
			}
		}()
	}

	go func() {
		for i := 0; i < runNum; i++ {
			_tmpAB = ab.Get()
		}
	}()

	select {
	case <-time.After(3 * time.Second):
		return
	}
}
