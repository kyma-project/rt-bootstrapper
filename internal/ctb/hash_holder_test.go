package ctb_test

import (
	"testing"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	"github.com/stretchr/testify/assert"
)

func TestHashHolder_GetSet(t *testing.T) {
	h := ctb.NewHashHolder()
	assert.Equal(t, "", h.Get())

	h.Set("abc123")
	assert.Equal(t, "abc123", h.Get())
}

func TestHashHolder_ComputeAndSet(t *testing.T) {
	h := ctb.NewHashHolder()
	h.ComputeAndSet("hello")
	// SHA-256 of "hello"
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", h.Get())
}

func TestHashHolder_ComputeAndSet_Empty(t *testing.T) {
	h := ctb.NewHashHolder()
	h.ComputeAndSet("")
	// SHA-256 of ""
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", h.Get())
}

func TestHashHolder_ConcurrentAccess(t *testing.T) {
	h := ctb.NewHashHolder()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			h.Set("value")
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		_ = h.Get()
	}
	<-done
}
