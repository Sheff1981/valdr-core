//go:build randomx_native && cgo

package randomxnative

/*
#cgo CFLAGS: -I${SRCDIR}/../../../third_party/randomx/include
#cgo linux LDFLAGS: -L${SRCDIR}/../../../third_party/randomx/lib -lrandomx -lstdc++ -lm
#cgo darwin LDFLAGS: -L${SRCDIR}/../../../third_party/randomx/lib -lrandomx -lc++ -lm
#cgo windows LDFLAGS: -L${SRCDIR}/../../../third_party/randomx/lib -lrandomx -lstdc++ -lws2_32

#include <stdlib.h>
#include "randomx.h"
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

var (
	ErrInit       = errors.New("RandomX native initialization failed")
	ErrEmptyKey   = errors.New("RandomX key must not be empty")
	ErrEmptyInput = errors.New("RandomX input must not be empty")
)

type Hasher struct {
	mu    sync.Mutex
	cache *C.randomx_cache
	vm    *C.randomx_vm
	key   []byte
	flags C.randomx_flags
}

func New() (*Hasher, error) {
	flags := C.randomx_get_flags()
	cache := C.randomx_alloc_cache(flags)
	if cache == nil {
		flags = C.RANDOMX_FLAG_DEFAULT
		cache = C.randomx_alloc_cache(flags)
		if cache == nil {
			return nil, ErrInit
		}
	}

	vm := C.randomx_create_vm(flags, cache, nil)
	if vm == nil {
		C.randomx_release_cache(cache)
		flags = C.RANDOMX_FLAG_DEFAULT
		cache = C.randomx_alloc_cache(flags)
		if cache == nil {
			return nil, ErrInit
		}
		vm = C.randomx_create_vm(flags, cache, nil)
		if vm == nil {
			C.randomx_release_cache(cache)
			return nil, ErrInit
		}
	}

	return &Hasher{cache: cache, vm: vm, flags: flags}, nil
}

func (h *Hasher) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.vm != nil {
		C.randomx_destroy_vm(h.vm)
		h.vm = nil
	}
	if h.cache != nil {
		C.randomx_release_cache(h.cache)
		h.cache = nil
	}
	h.key = nil
}

func (h *Hasher) Hash(key, input []byte) ([]byte, error) {
	if h == nil || h.cache == nil || h.vm == nil {
		return nil, ErrInit
	}
	if len(key) == 0 {
		return nil, ErrEmptyKey
	}
	if len(input) == 0 {
		return nil, ErrEmptyInput
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !equalBytes(h.key, key) {
		C.randomx_init_cache(
			h.cache,
			unsafe.Pointer(&key[0]),
			C.size_t(len(key)),
		)
		h.key = append(h.key[:0], key...)
	}

	out := make([]byte, C.RANDOMX_HASH_SIZE)
	C.randomx_calculate_hash(
		h.vm,
		unsafe.Pointer(&input[0]),
		C.size_t(len(input)),
		unsafe.Pointer(&out[0]),
	)
	return out, nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
