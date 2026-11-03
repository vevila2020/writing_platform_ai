package schema

import (
	agentsession "github.com/vevila2020/writing_platform_ai/agent/session"
)

// CacheKeyGenerator derives an opaque, stable provider cache-routing key.
type CacheKeyGenerator func(agentsession.Key) (string, error)
