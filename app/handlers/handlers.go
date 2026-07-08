// Package handlers loads crashdummy's mapping and proxy definitions and
// registers their routes on an HTTP mux.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/igorsilva-dev/crashdummy/app/chaos"
	"github.com/igorsilva-dev/crashdummy/app/models"
)

const (
	mappingsDir = "mappings"
	proxiesDir  = "proxies"
	stubsDir    = "stubs"
)

// validMethods is the set of HTTP methods a mapping or proxy may declare.
var validMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        30,
		MaxIdleConnsPerHost: 30,
	},
}

// Register loads every mapping and proxy definition from disk and registers
// its route on mux.
func Register(mux *http.ServeMux) error {
	mappings, err := loadMappings()
	if err != nil {
		return fmt.Errorf("loading mappings: %w", err)
	}
	for _, mapping := range mappings {
		registerMapping(mux, mapping)
	}

	proxies, err := loadProxies()
	if err != nil {
		return fmt.Errorf("loading proxies: %w", err)
	}
	for _, proxy := range proxies {
		registerProxy(mux, proxy)
	}

	return nil
}

func loadMappings() ([]models.Mapping, error) {
	entries, err := os.ReadDir(mappingsDir)
	if err != nil {
		return nil, err
	}

	var mappings []models.Mapping
	for _, entry := range entries {
		mapping, err := loadMapping(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("mapping %s: %w", entry.Name(), err)
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func loadMapping(name string) (models.Mapping, error) {
	var mapping models.Mapping

	data, err := os.ReadFile(filepath.Join(mappingsDir, name))
	if err != nil {
		return mapping, err
	}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return mapping, err
	}

	mapping.Request.Method = strings.ToUpper(strings.TrimSpace(mapping.Request.Method))
	if mapping.Request.Method == "" {
		mapping.Request.Method = http.MethodGet
	}
	if !validMethods[mapping.Request.Method] {
		return mapping, fmt.Errorf("unsupported request method %q", mapping.Request.Method)
	}
	if mapping.Request.URL == "" {
		return mapping, fmt.Errorf("request url is required")
	}

	if mapping.Response.Status == 0 {
		mapping.Response.Status = http.StatusOK
	}
	if mapping.Response.Status < 100 || mapping.Response.Status > 599 {
		return mapping, fmt.Errorf("response status %d out of range", mapping.Response.Status)
	}

	stub, err := os.ReadFile(filepath.Join(stubsDir, mapping.Response.BodyFileName))
	if err != nil {
		return mapping, fmt.Errorf("stub %s: %w", mapping.Response.BodyFileName, err)
	}
	if !json.Valid(stub) {
		return mapping, fmt.Errorf("stub %s: not valid JSON", mapping.Response.BodyFileName)
	}
	mapping.MappedResponse = string(stub)

	return mapping, nil
}

func loadProxies() ([]models.Proxy, error) {
	entries, err := os.ReadDir(proxiesDir)
	if err != nil {
		return nil, err
	}

	var proxies []models.Proxy
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(proxiesDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("proxy %s: %w", entry.Name(), err)
		}
		var proxy models.Proxy
		if err := json.Unmarshal(data, &proxy); err != nil {
			return nil, fmt.Errorf("proxy %s: %w", entry.Name(), err)
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

func registerMapping(mux *http.ServeMux, mapping models.Mapping) {
	pattern := mapping.Request.Method + " " + mapping.Request.URL
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Chaos-Type", "mock")
		w.WriteHeader(mapping.Response.Status)
		fmt.Fprint(w, mapping.MappedResponse)
	})
}

func registerProxy(mux *http.ServeMux, proxy models.Proxy) {
	latency := chaos.New(int64(proxy.LatencyInMilliseconds), int64(proxy.JitterInMilliseconds))

	mux.HandleFunc(proxy.Path, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(latency.Duration())

		req, err := http.NewRequestWithContext(r.Context(), proxy.Method, proxy.Upstream, nil)
		if err != nil {
			writeProxyError(w, fmt.Errorf("building upstream request: %w", err))
			return
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			writeProxyError(w, fmt.Errorf("calling upstream: %w", err))
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Chaos-Type", "proxy")
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("proxy %s: copying upstream response: %v", proxy.Path, err)
		}
	})
}

func writeProxyError(w http.ResponseWriter, err error) {
	log.Printf("proxy error: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Chaos-Type", "proxy")
	w.WriteHeader(http.StatusBadGateway)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encodeErr != nil {
		log.Printf("writing proxy error response: %v", encodeErr)
	}
}
