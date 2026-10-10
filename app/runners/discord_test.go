package runners

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/foundation"
	"github.com/stretchr/testify/require"
)

// stubApp embeds the framework Application so only Context needs implementing.
type stubApp struct {
	foundation.Application

	ctx     context.Context
	entered chan struct{}
	once    sync.Once
}

// Context signals entered (when set) so a test can deterministically observe
// that Run reached its blocking wait.
func (s *stubApp) Context() context.Context {
	if s.entered != nil {
		s.once.Do(func() {
			close(s.entered)
		})
	}

	return s.ctx
}

func TestDiscordRunnerSignature(t *testing.T) {
	runner := NewDiscord(&stubApp{})

	require.Equal(t, "discord", runner.Signature())
}

func TestDiscordRunnerShouldRun(t *testing.T) {
	runner := NewDiscord(&stubApp{})

	require.True(t, runner.ShouldRun())
}

func TestDiscordRunnerRunBlocksUntilContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	runner := NewDiscord(&stubApp{ctx: ctx, entered: entered})

	done := make(chan error, 1)
	go func() {
		done <- runner.Run()
	}()

	// Wait until Run is inside the blocking wait before asserting it has not
	// returned.
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("Run did not reach the blocking wait")
	}

	select {
	case err := <-done:
		t.Fatalf("Run returned before the context was cancelled: %v", err)
	default:
	}

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after the context was cancelled")
	}
}

func TestDiscordRunnerShutdownReturnsNil(t *testing.T) {
	runner := NewDiscord(&stubApp{})

	require.NoError(t, runner.Shutdown())
}
