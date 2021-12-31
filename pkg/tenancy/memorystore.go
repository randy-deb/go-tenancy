package tenancy

import (
	"errors"
	"sync"
)

func NewInMemoryTenantStore() *InMemoryTenantStore {
	return &InMemoryTenantStore{
		data: []Tenant{
			{Id: "1", Scheme: "http", Name: "Dev", Host: "localhost:5100", VirtualPath: "dev"},
			{Id: "2", Scheme: "http", Name: "Stg", Host: "localhost:5100", VirtualPath: "stg"},
		},
	}
}

type InMemoryTenantStore struct {
	sync.RWMutex
	data []Tenant
}

func (store *InMemoryTenantStore) Resolve(scheme string, host string, path string) (*Tenant, error) {
	store.Lock()
	defer store.Unlock()

	for _, item := range store.data {
		if item.Scheme == scheme && item.Host == host && item.VirtualPath == path {
			return &item, nil
		}
	}

	return nil, errors.New("Tenant not found")
}
