package realtime

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
)

func sampleLog(service string, level model.Level, msg string) model.Log {
	return model.Log{
		Timestamp: time.Now(),
		Service:   service,
		Level:     level,
		Message:   msg,
	}
}

func TestHub_SubscribeReceivesBroadcast(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe(Filter{})
	defer cancel()

	want := sampleLog("api", model.LevelInfo, "hello")
	h.Broadcast([]model.Log{want})

	select {
	case got := <-ch:
		assert.Equal(t, want.Message, got.Message)
	case <-time.After(time.Second):
		t.Fatal("expected to receive a broadcast log")
	}
}

func TestHub_MultipleSubscribersAllReceive(t *testing.T) {
	h := NewHub()
	const n = 5

	chans := make([]<-chan model.Log, n)
	cancels := make([]func(), n)
	for i := 0; i < n; i++ {
		chans[i], cancels[i] = h.Subscribe(Filter{})
	}
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()

	h.Broadcast([]model.Log{sampleLog("api", model.LevelInfo, "fanout")})

	for i := 0; i < n; i++ {
		select {
		case got := <-chans[i]:
			assert.Equal(t, "fanout", got.Message)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive the broadcast", i)
		}
	}
}

func TestHub_FilterByService(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe(Filter{Service: "worker"})
	defer cancel()

	h.Broadcast([]model.Log{
		sampleLog("api", model.LevelInfo, "ignored"),
		sampleLog("worker", model.LevelInfo, "delivered"),
	})

	select {
	case got := <-ch:
		assert.Equal(t, "delivered", got.Message,
			"only the worker log should pass the service filter")
	case <-time.After(time.Second):
		t.Fatal("expected the worker log")
	}

	// No further messages should be queued.
	select {
	case unexpected := <-ch:
		t.Fatalf("did not expect another log, got %q", unexpected.Message)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestHub_FilterByLevel(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe(Filter{Level: model.LevelError})
	defer cancel()

	h.Broadcast([]model.Log{
		sampleLog("api", model.LevelInfo, "ignored"),
		sampleLog("api", model.LevelError, "delivered"),
	})

	select {
	case got := <-ch:
		assert.Equal(t, model.LevelError, got.Level)
	case <-time.After(time.Second):
		t.Fatal("expected the error log")
	}
}

func TestHub_UnsubscribeClosesChannel(t *testing.T) {
	h := NewHub()
	ch, cancel := h.Subscribe(Filter{})

	cancel()

	// Reading a closed channel returns the zero value and ok=false.
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after cancel")
	case <-time.After(time.Second):
		t.Fatal("expected closed channel read to return immediately")
	}
}

func TestHub_SubscriberCount(t *testing.T) {
	h := NewHub()
	assert.Equal(t, 0, h.SubscriberCount())

	_, c1 := h.Subscribe(Filter{})
	_, c2 := h.Subscribe(Filter{})
	assert.Equal(t, 2, h.SubscriberCount())

	c1()
	assert.Equal(t, 1, h.SubscriberCount())

	c2()
	assert.Equal(t, 0, h.SubscriberCount())
}

func TestHub_SlowSubscriberDoesNotBlock(t *testing.T) {
	h := NewHub()
	_, cancel := h.Subscribe(Filter{}) // never read from
	defer cancel()

	done := make(chan struct{})
	go func() {
		// More than buffer capacity; must not deadlock.
		for i := 0; i < 10_000; i++ {
			h.Broadcast([]model.Log{sampleLog("api", model.LevelInfo, "spam")})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Broadcast blocked on a slow subscriber")
	}
}

func TestHub_ConcurrentSubscribeBroadcast(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			h.Broadcast([]model.Log{sampleLog("api", model.LevelInfo, "x")})
		}
	}()

	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_, cancel := h.Subscribe(Filter{})
				cancel()
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 0, h.SubscriberCount())
}

func TestFilter_ZeroValueMatchesEverything(t *testing.T) {
	var f Filter
	require.True(t, f.allows(sampleLog("anything", model.LevelDebug, "m")))
	require.True(t, f.allows(sampleLog("other", model.LevelFatal, "m")))
}
