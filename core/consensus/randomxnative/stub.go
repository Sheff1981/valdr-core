//go:build !randomx_native || !cgo

package randomxnative

import "errors"

var ErrNativeDisabled = errors.New("RandomX native adapter is not enabled; build with CGO_ENABLED=1 and -tags randomx_native")

type Hasher struct{}

func New() (*Hasher, error) { return nil, ErrNativeDisabled }
func (h *Hasher) Close()    {}
func (h *Hasher) Hash(key, input []byte) ([]byte, error) {
	return nil, ErrNativeDisabled
}
