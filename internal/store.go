package internal

import (
	"essence"
	"fmt"
	"foundation/extensions"
)

type Signal struct {
	id essence.UUID
}

func (s *Signal) GetUUID() essence.UUID {
	return s.id
}

func SignalCreate() *Signal {
	genId, _ := essence.UUIDv7GenerateRandom()

	return &Signal{
		id: genId,
	}
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

func SignalStoreRemoveByID(store *SignalStore, id essence.UUID) error {
	modified, removedCount := extensions.RemoveWhereInPlace(store.storedSignals, func(item Signal) bool {
		return item.GetUUID() == id
	})

	if removedCount == 0 {
		return fmt.Errorf("could not find id '%s'", id.String())
	}

	store.storedSignals = modified

	return nil
}

func SignalStoreStoredAmountGet(store *SignalStore) int {
	return len(store.storedSignals)
}
