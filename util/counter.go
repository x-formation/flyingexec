package util

import "sync/atomic"

type Counter uint32

func (n *Counter) Next() uint32 {
	return atomic.AddUint32((*uint32)(n), 1)
}
