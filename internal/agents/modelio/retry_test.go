package modelio

import (
	"context"
	"net/http"
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
)

func TestTransientModelErrorStopsWhenRunContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := &providers.APIError{StatusCode: http.StatusTooManyRequests}
	if IsRetryable(ctx, err) {
		t.Fatal("cancelled run must not schedule another provider retry")
	}
}
