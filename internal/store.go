package internal

type Signal struct {
}

func SignalCreate() *Signal {
	return &Signal{}
}

type SignalStore struct {
}

func SignalStoreCreate() *SignalStore {
	return &SignalStore{}
}

func SignalStoreStore(store *SignalStore, signal Signal) error {
	return nil
}

func SignalStoreStoredAmountGet(store *SignalStore) int {
	return 1
}
