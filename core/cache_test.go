package core

import (
	"errors"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	c := NewCache(0)
	defer c.Close()

	c.Set("user:1", "Alice", 100*time.Millisecond)

	val, ok := c.Get("user:1")
	if !ok || val != "Alice" {
		t.Fatalf("expected 'Alice', got %v (found=%v)", val, ok)
	}

	time.Sleep(150 * time.Millisecond)

	_, ok = c.Get("user:1")
	if ok {
		t.Fatal("expected item to be expired")
	}
}

func TestCacheRemember(t *testing.T) {
	c := NewCache(0)
	defer c.Close()

	calls := 0
	compute := func() (interface{}, error) {
		calls++
		return "expensive_result", nil
	}

	val1, err := c.Remember("heavy_query", 1*time.Second, compute)
	if err != nil || val1 != "expensive_result" {
		t.Fatalf("unexpected result %v, err=%v", val1, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}

	// Second call should return cached value without computing
	val2, err := c.Remember("heavy_query", 1*time.Second, compute)
	if err != nil || val2 != "expensive_result" {
		t.Fatalf("unexpected result %v, err=%v", val2, err)
	}
	if calls != 1 {
		t.Fatalf("expected computation to not run again, calls=%d", calls)
	}
}

func TestCacheRememberError(t *testing.T) {
	c := NewCache(0)
	defer c.Close()

	expectedErr := errors.New("computation error")
	_, err := c.Remember("fail_key", 1*time.Second, func() (interface{}, error) {
		return nil, expectedErr
	})

	if err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}

	if _, ok := c.Get("fail_key"); ok {
		t.Fatal("failed computation should not be cached")
	}
}

func TestCacheDeleteAndFlush(t *testing.T) {
	c := NewCache(0)
	defer c.Close()

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)

	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected key 'a' to be deleted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("expected key 'b' to exist")
	}

	c.Flush()
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected cache to be empty after Flush")
	}
}
