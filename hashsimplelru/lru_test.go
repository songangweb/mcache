package hashSimplelru

import (
	"fmt"
	"testing"
	"time"
)

func TestLRU(t *testing.T) {

	initTime := initTime()
	evictCounter := 0
	onEvicted := func(k interface{}, v *interface{}, expirationTime int64) {
		if k != *v {
			t.Fatalf("Evict values not equal (%v!=%v) , time = %v", k, *v, expirationTime)
		}
		evictCounter++
	}
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		var v interface{}
		v = i
		l.Add(i, &v, initTime)
	}

	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, expirationTime, ok := l.Get(k); !ok || *v != k || *v != i+128 {
			t.Fatalf("bad i: %v, key: %v, val: %v, time: %v", i, k, *v, expirationTime)
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

	l.Get(192) // expect 192 to be last key in l.Keys()

	for i, k := range l.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
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

func TestLRU_GetOldest_RemoveOldest(t *testing.T) {
	initTime := initTime()

	l, err := NewLRU(128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		var v interface{}
		v = i
		l.Add(i, &v, initTime)
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
func TestLRU_Add(t *testing.T) {
	initTime := initTime()

	evictCounter := 0
	onEvicted := func(k interface{}, v *interface{}, expirationTime int64) {
		evictCounter++
	}

	l, err := NewLRU(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var v1 interface{}
	v1 = 1
	if l.Add(1, &v1, initTime) == false || evictCounter != 0 {
		t.Errorf(fmt.Sprint(evictCounter))
		t.Errorf("should not have an eviction")
	}

	var v2 interface{}
	v2 = 2
	if l.Add(2, &v2, initTime) == false || evictCounter != 1 {
		t.Errorf(fmt.Sprint(evictCounter))
		t.Errorf("should have an eviction")
	}
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	initTime := initTime()

	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var v1 interface{}
	v1 = 1
	l.Add(1, &v1, initTime)
	var v2 interface{}
	v2 = 2
	l.Add(2, &v2, initTime)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}
	var v3 interface{}
	v3 = 3
	l.Add(3, &v3, initTime)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	initTime := initTime()

	l, err := NewLRU(2, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var v1 interface{}
	v1 = 1
	l.Add(1, &v1, initTime)
	var v2 interface{}
	v2 = 2
	l.Add(2, &v2, initTime)
	if v, _, ok := l.Peek(1); !ok || *v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", *v, ok)
	}
	var v3 interface{}
	v3 = 1
	l.Add(3, &v3, initTime)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

// Test that Resize can upsize and downsize
func TestLRU_Resize(t *testing.T) {
	initTime := initTime()

	onEvictCounter := 0
	onEvicted := func(k interface{}, v *interface{}, expirationTime int64) {
		onEvictCounter++
	}
	l, err := NewLRU(2, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Downsize
	var v1 interface{}
	v1 = 1
	l.Add(1, &v1, initTime)
	var v2 interface{}
	v2 = 2
	l.Add(2, &v2, initTime)
	evicted := l.Resize(1)
	if evicted != 1 {
		t.Errorf("1 element should have been evicted: %v", evicted)
	}
	if onEvictCounter != 1 {
		t.Errorf("onEvicted should have been called 1 time: %v", onEvictCounter)
	}
	var v3 interface{}
	v3 = 3
	l.Add(3, &v3, initTime)
	if l.Contains(1) {
		t.Errorf("Element 1 should have been evicted")
	}

	// Upsize
	evicted = l.Resize(2)
	if evicted != 0 {
		t.Errorf("0 elements should have been evicted: %v", evicted)
	}

	var v4 interface{}
	v4 = 4
	l.Add(4, &v4, initTime)
	if !l.Contains(3) || !l.Contains(4) {
		t.Errorf("Cache should have contained 2 elements")
	}
}

// 生成当前时间
func initTime() int64 {
	return time.Now().UnixNano()/1e6 + 2000
}
