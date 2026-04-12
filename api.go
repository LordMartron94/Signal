package signal

import "signal/internal"

type Signal = internal.Signal

func SignalCreate() *Signal {
	return internal.SignalCreate()
}

type SignalStore = internal.SignalStore

func SignalStoreCreate() *SignalStore {
	return internal.SignalStoreCreate()
}

func SignalStoreStore(store *SignalStore, in Signal) error {
	return internal.SignalStoreStore(store, in)
}

func SignalStoreStoredAmountGet(store *SignalStore) int {
	return internal.SignalStoreStoredAmountGet(store)
}
