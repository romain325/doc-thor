package routes

import (
	"net/http"

	"github.com/romain325/doc-thor/server/discovery"
)

func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func Backends(builderEndpoints []string, storageEndpoint string, storageUseSSL bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statuses []discovery.BackendStatus
		for _, url := range builderEndpoints {
			statuses = append(statuses, discovery.CheckBuilder(url))
		}
		statuses = append(statuses, discovery.CheckStorage(storageEndpoint, storageUseSSL))
		writeJSON(w, http.StatusOK, statuses)
	}
}
