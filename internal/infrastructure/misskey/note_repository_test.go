package misskey

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_ImmediateExecution(t *testing.T) {
	limiter := newRateLimiter(3, 10*time.Second)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("unexpected error on request %d: %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("expected immediate execution within 100ms, took %v", elapsed)
	}

	if limiter.permits != 0 {
		t.Errorf("expected 0 permits remaining, got %d", limiter.permits)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	refillInterval := 100 * time.Millisecond
	limiter := newRateLimiter(1, refillInterval)
	ctx := context.Background()

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < refillInterval {
		t.Errorf("expected to wait at least %v, only waited %v", refillInterval, elapsed)
	}

	if elapsed > refillInterval+50*time.Millisecond {
		t.Errorf("waited too long: %v (expected ~%v)", elapsed, refillInterval)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	limiter := newRateLimiter(1, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("cancellation took too long: %v", elapsed)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	maxPermits := 5
	limiter := newRateLimiter(maxPermits, 50*time.Millisecond)
	ctx := context.Background()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := limiter.Wait(ctx); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	elapsed := time.Since(start)

	for err := range errors {
		t.Errorf("unexpected error from goroutine: %v", err)
	}

	expectedMinDuration := 50 * time.Millisecond
	if elapsed < expectedMinDuration {
		t.Errorf("expected at least %v for token refill, got %v", expectedMinDuration, elapsed)
	}
}

func TestRateLimiter_MultipleRefills(t *testing.T) {
	refillInterval := 50 * time.Millisecond
	limiter := newRateLimiter(2, refillInterval)
	ctx := context.Background()

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("request 1 failed: %v", err)
	}
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("request 2 failed: %v", err)
	}

	time.Sleep(refillInterval * 3)

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("request after sleep failed: %v", err)
	}

	limiter.mu.Lock()
	if limiter.permits < 1 {
		t.Errorf("expected at least 1 permit after refill and one use, got %d", limiter.permits)
	}
	if limiter.permits > limiter.maxPermits {
		t.Errorf("permits exceeded max: %d > %d", limiter.permits, limiter.maxPermits)
	}
	limiter.mu.Unlock()
}

func TestRateLimiter_ZeroTokensWait(t *testing.T) {
	refillInterval := 100 * time.Millisecond
	limiter := newRateLimiter(1, refillInterval)
	ctx := context.Background()

	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	limiter.mu.Lock()
	if limiter.permits != 0 {
		t.Errorf("expected 0 permits after first request, got %d", limiter.permits)
	}
	limiter.mu.Unlock()

	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < refillInterval {
		t.Errorf("should have waited for refill, elapsed: %v", elapsed)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 0, -1},
		{0, -1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}
