package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type VersionResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
}

type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	Environment  string `json:"environment"`
	RepoURL      string `json:"repo_url"`
	SlackChannel string `json:"slack_channel"`
	Tier         string `json:"tier"`
	Language     string `json:"language"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type BootstrapManifest struct {
	RepoTemplate    string   `json:"repo_template"`
	CIWorkflow      string   `json:"ci_workflow"`
	Dockerfile      bool     `json:"dockerfile"`
	HealthCheckPath string   `json:"health_check_path"`
	MetricsPath     string   `json:"metrics_path"`
	Dashboard       string   `json:"dashboard"`
	Alerts          []string `json:"alerts"`
}

type CreateServiceResponse struct {
	Service   Service           `json:"service"`
	Bootstrap BootstrapManifest `json:"bootstrap"`
}

var (
	servicesByID = map[string]Service{}
	servicesMu   sync.Mutex
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("failed to encode json:", err)
	}
}

func healthHandler(w http.ResponseWriter, s *http.Request) {
	if s.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: ErrorDetail{
				Code:    "method_not_allowed",
				Message: "method not allowed",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "service-registry-api",
	})
}

func versionHandler(w http.ResponseWriter, s *http.Request) {
	if s.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: ErrorDetail{
				Code:    "method_not_allowed",
				Message: "method not allowed",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, VersionResponse{
		Service: "service-registry-api",
		Version: "v0.0.1",
	})
}

func createServiceHandler(w http.ResponseWriter, s *http.Request) {
	var service Service

	if err := json.NewDecoder(s.Body).Decode(&service); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "invalid json request body",
			},
		})
		return
	}

	if service.ID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "id is required",
			},
		})
		return
	}

	if service.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "name is required",
			},
		})
		return
	}

	if service.Owner == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "owner is required",
			},
		})
		return
	}

	if service.Environment == "" {
		service.Environment = "dev"
	}

	if service.Environment != "dev" && service.Environment != "staging" && service.Environment != "prod" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "environment must be one of: dev, staging, prod",
			},
		})
		return
	}

	if service.Tier == "" {
		service.Tier = "tier-3"
	}

	if service.Tier != "tier-1" && service.Tier != "tier-2" && service.Tier != "tier-3" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "tier must be either tier-1, tier-2, or tier-3",
			},
		})
		return
	}

	if service.Language == "" {
		service.Language = "go"
	}

	if service.Language != "go" && service.Language != "python" && service.Language != "node" && service.Language != "java" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "language must be either go, python, node or java",
			},
		})
		return
	}

	if service.RepoURL != "" && !strings.HasPrefix(service.RepoURL, "https://") && !strings.HasPrefix(service.RepoURL, "http://") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "repo_url must start with either http:// or https://",
			},
		})
		return
	}

	if service.SlackChannel != "" && !strings.HasPrefix(service.SlackChannel, "#") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "invalid_request",
				Message: "expected prefix of # for slack channel name",
			},
		})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	service.CreatedAt = now
	service.UpdatedAt = now

	servicesMu.Lock()
	if _, exists := servicesByID[service.ID]; exists {
		servicesMu.Unlock()
		writeJSON(w, http.StatusConflict, ErrorResponse{
			Error: ErrorDetail{
				Code:    "conflict",
				Message: "service id already exists",
			},
		})
		return
	}
	servicesByID[service.ID] = service
	servicesMu.Unlock()

	response := CreateServiceResponse{
		Service: service,
		Bootstrap: BootstrapManifest{
			RepoTemplate:    "go-service-template",
			CIWorkflow:      "github-actions-go",
			Dockerfile:      true,
			HealthCheckPath: "/health",
			MetricsPath:     "/metrics",
			Dashboard:       "default-service-dashboard",
			Alerts:          []string{"high-error-rate", "high-latency"},
		},
	}

	writeJSON(w, http.StatusCreated, response)
}

func listServicesHandler(w http.ResponseWriter, s *http.Request) {
	servicesMu.Lock()
	services := make([]Service, 0, len(servicesByID))

	for _, service := range servicesByID {
		services = append(services, service)
	}
	servicesMu.Unlock()
	sort.Slice(services, func(i, j int) bool {
		return services[i].ID < services[j].ID
	})

	writeJSON(w, http.StatusOK, services)
}

func serviceHandler(w http.ResponseWriter, s *http.Request) {
	if s.Method != http.MethodPost && s.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: ErrorDetail{
				Code:    "method_not_allowed",
				Message: "method not allowed",
			},
		})
		return
	}

	if s.Method == http.MethodPost {
		createServiceHandler(w, s)
		return
	}

	if s.Method == http.MethodGet {
		listServicesHandler(w, s)
		return
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/version", versionHandler)
	mux.HandleFunc("/services", serviceHandler)

	addr := ":8080"
	log.Println("start listening on port 8080")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
