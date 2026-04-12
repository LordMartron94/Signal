package internal

type Signal struct {
}

func SignalCreate() *Signal {
	return &Signal{}
}

type SignalStore struct {
	storedSignals []Signal
}

func SignalStoreCreate() *SignalStore {
	return &SignalStore{
		storedSignals: make([]Signal, 0),
	}
}

func SignalStoreStore(store *SignalStore, signal Signal) error {
	store.storedSignals = append(store.storedSignals, signal)

	return nil
}

func SignalStoreStoredAmountGet(store *SignalStore) int {
	return len(store.storedSignals)
}
