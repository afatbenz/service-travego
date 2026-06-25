package waai

import (
	"log"
	"service-travego/internal/wagy"
	"sync"
)

// WagyClientRegistry manages WagyClient per device (thread-safe, lazy init)
type WagyClientRegistry struct {
	mu      sync.RWMutex
	clients map[string]*wagy.WagyClient
}

func NewWagyClientRegistry() *WagyClientRegistry {
	return &WagyClientRegistry{
		clients: make(map[string]*wagy.WagyClient),
	}
}

// GetClient mengembalikan WagyClient untuk assistant_device_id tertentu.
// Membuat client baru (lazy) jika belum ada di cache.
func (r *WagyClientRegistry) GetClient(assistantDeviceID, deviceToken string) *wagy.WagyClient {
	if assistantDeviceID == "" || deviceToken == "" {
		log.Printf("[WAAI][Registry] Empty assistantDeviceID or deviceToken, cannot create client")
		return nil
	}

	r.mu.RLock()
	client, exists := r.clients[assistantDeviceID]
	r.mu.RUnlock()

	if exists {
		return client
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if client, exists = r.clients[assistantDeviceID]; exists {
		return client
	}

	client = wagy.NewWagyClient(assistantDeviceID, deviceToken)
	r.clients[assistantDeviceID] = client

	log.Printf("[WAAI][Registry] Registered WagyClient for device: %s", assistantDeviceID)
	return client
}

// Invalidate menghapus client dari cache (berguna saat token di-rotate atau device di-nonaktifkan)
func (r *WagyClientRegistry) Invalidate(assistantDeviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, assistantDeviceID)
	log.Printf("[WAAI][Registry] Invalidated client for device: %s", assistantDeviceID)
}
