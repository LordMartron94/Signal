package signal

import "signal/internal"

type SignalStore = internal.SignalStore

func SignalStoreCreate() *SignalStore {
	return internal.SignalStoreCreate()
}
