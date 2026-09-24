package interceptors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFixedWindowLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := newFixedWindowLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	assert.True(t, l.allow("a"))
	assert.True(t, l.allow("a"))
	assert.False(t, l.allow("a"))
	assert.True(t, l.allow("b"), "limits are per key")

	now = now.Add(time.Minute)
	assert.True(t, l.allow("a"), "window resets")
}
