package simplelfu

import (
	"fmt"
	"testing"
	"time"
)

func TestLFU(t *testing.T) {

	initTime := initTime()
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}, expirationTime int64) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v) , time = %v", k, v, expirationTime)
		}
		evictCounter++
	}
	l, err := NewLFU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i, initTime)
		for c := 0; c < i; c++ {
			l.Get(i)
		}
	}

	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, expirationTime, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad i: %v, key: %v, v: %v, time: %v", i, k, v, expirationTime)
		}
	}
	for i := 0; i < 128; i++ {
		_, expirationTime, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted , time: %v", expirationTime)
		}
	}
	for i := 128; i < 256; i++ {
		_, expirationTime, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted, time: %v", expirationTime)
		}
	}

	for i := 128; i < 192; i++ {
		ok := l.Remove(i)
		if !ok {
			t.Fatalf("should be contained")
		}
		ok = l.Remove(i)
		if ok {
			t.Fatalf("should not be contained")
		}
		_, expirationTime, ok := l.Get(i)
		if ok {
			t.Fatalf("should be deleted, time: %v", expirationTime)
		}
	}

	l.Get(192) // expect 192 to be first key in l.Keys()
	for i, k := range l.Keys() {
		if i < 63 && k != 192+i {
			t.Fatalf("out of order i:% v ,key: %v", i, k)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, expirationTime, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing, time: %v", expirationTime)
	}
}

func TestLFU_GetOldest_RemoveOldest(t *testing.T) {
	initTime := initTime()

	l, err := NewLFU(128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		l.Add(i, i, initTime)
		for c := 0; c < i; c++ {
			l.Get(i)
		}
	}
	k, _, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != 129 {
		t.Fatalf("bad: %v", k)
	}
}

// Test that Add returns true/false if an eviction occurred
func TestLFU_Add(t *testing.T) {
	initTime := initTime()

	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}, expirationTime int64) {
		evictCounter++
	}

	l, err := NewLFU(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if l.Add(1, 1, initTime) == false || evictCounter != 0 {
		t.Errorf(fmt.Sprint(evictCounter))
		t.Errorf("should not have an eviction")
	}

	if l.Add(2, 2, initTime) == false || evictCounter != 1 {
		t.Errorf(fmt.Sprint(evictCounter))
		t.Errorf("should have an eviction")
	}
}

// Test that Contains doesn't update recent-ness
func TestLFU_Contains(t *testing.T) {
	initTime := initTime()

	l, err := NewLFU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1, initTime)
	l.Get(1)
	l.Add(2, 2, initTime)
	l.Get(2)
	l.Get(2)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	l.Add(3, 3, initTime)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLFU_Peek(t *testing.T) {
	initTime := initTime()

	l, err := NewLFU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1, initTime)
	l.Get(1)
	l.Add(2, 2, initTime)
	l.Get(2)
	l.Get(2)
	if v, _, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, 3, initTime)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

// Test that Resize can upsize and downsize
func TestLFU_Resize(t *testing.T) {
	initTime := initTime()

	onEvictCounter := 0
	onEvicted := func(k interface{}, v interface{}, expirationTime int64) {
		onEvictCounter++
	}
	l, err := NewLFU(2, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Downsize
	l.Add(1, 1, initTime)
	l.Add(2, 2, initTime)
	evicted := l.Resize(1)
	if evicted != 1 {
		t.Errorf("1 element should have been evicted: %v", evicted)
	}
	if onEvictCounter != 1 {
		t.Errorf("onEvicted should have been called 1 time: %v", onEvictCounter)
	}

	l.Add(3, 3, initTime)
	if l.Contains(1) {
		t.Errorf("Element 1 should have been evicted")
	}

	// Upsize
	evicted = l.Resize(2)
	if evicted != 0 {
		t.Errorf("0 elements should have been evicted: %v", evicted)
	}

	l.Add(4, 4, initTime)
	if !l.Contains(3) || !l.Contains(4) {
		t.Errorf("Cache should have contained 2 elements")
	}
}

// 生成当前时间 + 2秒
func initTime() int64 {
	return time.Now().UnixNano()/1e6 + 2000
}
