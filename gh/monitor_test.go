package gh

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnsureOnce(t *testing.T) {
	var count int32
	fn := func() error {
		atomic.AddInt32(&count, 1)
		return nil
	}
	ensureOnce(fn, time.Millisecond*10)
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))

	atomic.StoreInt32(&count, 0)
	fn = func() error {
		val := atomic.AddInt32(&count, 1)
		if val == 1 {
			return nil
		}

		return errors.New("again")
	}
	ensureOnce(fn, time.Millisecond*10)
	assert.Equal(t, int32(1), atomic.LoadInt32(&count))
}
