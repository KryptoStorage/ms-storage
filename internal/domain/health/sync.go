package health

import "time"

type SyncStatus struct {
	Status      Status   `json:"status"`
	LastSync   time.Time `json:"last_sync"`
	NextSync   time.Time `json:"next_sync"`
	State      string   `json:"state"`
	Description string  `json:"description,omitempty"`
}
