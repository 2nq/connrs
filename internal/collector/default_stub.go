//go:build !windows

package collector

import "errors"

func NewDefault() (Poller, error) {
	return nil, errors.New("connrs only supports Windows")
}
