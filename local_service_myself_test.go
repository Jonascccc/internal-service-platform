package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// helper: resets shared state between tests
func resetStore() {
	servicesMu.Lock()
	servicesByID = map[string]Service{}
	servicesMu.Unlock()
}

func TestHealthReturns200(t *testing.T) {
	// YOUR TURN:
	// 1. create a GET request to /health using httptest.NewRequest
	// 2. create a httptest.NewRecorder()
	// 3. call healthHandler(recorder, request)
	// 4. check recorder.Code == 200
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestVersionReturns200(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	versionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateServiceReturns201(t *testing.T) {
	// 1. resetStore()
	// 2. POST body: {"id":"svc-1","name":"My Service","owner":"team-a"}
	// 3. call serviceHandler
	// 4. check 201
	resetStore()

	body := `{"id":"svc-1","name":"Payment","owner":"Payment Team"}`

	readerbody := strings.NewReader(body)

	req := httptest.NewRequest(http.MethodPost, "/services", readerbody)
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

}

func TestCreateServiceMissingIDReturns400(t *testing.T) {
	// body missing "id" field → expect 400
	resetStore()

	body := `{"name":"Payment","owner":"Payment Team"}`

	readerBody := strings.NewReader(body)

	req := httptest.NewRequest(http.MethodPost, "/services", readerBody)
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateServiceDuplicateIDReturns409(t *testing.T) {
	// 1. resetStore()
	// 2. create the same service twice
	// 3. second call → expect 409
	resetStore()

	body := `{"id":"svc-01","name":"Payment","owner":"Payment Team","environment":"dev"}`

	readerBody := strings.NewReader(body)
	readerBody2 := strings.NewReader(body)

	req1 := httptest.NewRequest(http.MethodPost, "/services", readerBody)
	rec1 := httptest.NewRecorder()
	serviceHandler(rec1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/services", readerBody2)
	rec2 := httptest.NewRecorder()
	serviceHandler(rec2, req2)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec1.Code, rec1.Body.String())
	}

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusConflict, rec2.Code, rec2.Body.String())
	}
}

func TestListServicesReturnsSortedByID(t *testing.T) {
	// 1. resetStore()
	// 2. create "svc-b" then "svc-a"
	// 3. GET /services
	// 4. decode response, check order: svc-a comes before svc-b
	resetStore()

	body1 := `{"id":"svc-b","name":"Payment","owner":"Payment Team","environment":"dev"}`
	body2 := `{"id":"svc-a","name":"Payment","owner":"Payment Team","environment":"dev"}`

	readerBody1 := strings.NewReader(body1)
	readerBody2 := strings.NewReader(body2)

	req1 := httptest.NewRequest(http.MethodPost, "/services", readerBody1)
	rec1 := httptest.NewRecorder()
	serviceHandler(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec1.Code, rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/services", readerBody2)
	rec2 := httptest.NewRecorder()
	serviceHandler(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec2.Code, rec2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec3 := httptest.NewRecorder()

	serviceHandler(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, rec3.Code, rec3.Body.String())
	}

	var services []Service
	err := json.NewDecoder(rec3.Body).Decode(&services)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if len(services) != 2 {
		t.Fatalf("the length of services is not correct, expected %d, got %d", 2, len(services))
	}

	if services[0].ID != "svc-a" || services[1].ID != "svc-b" {
		t.Fatalf("the returned services is not in the correct order, got %q then %q, expected %q then %q", services[0].ID, services[1].ID, "svc-a", "svc-b")
	}
}

func TestCreateServiceInvalidEnvironmentReturns400(t *testing.T) {
	resetStore()

	body := `{"id":"svc-01","name":"Payment Service","owner":"Payment Team","environment":"test"}`
	readerbody := strings.NewReader(body)

	req := httptest.NewRequest(http.MethodPost, "/services", readerbody)
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateServiceSetsTimestamps(t *testing.T) {
	resetStore()

	body := `{"id":"svc-time","name":"Timed Service","owner":"Platform Team"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response CreateServiceResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	service := response.Service
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if service.CreatedAt == "" {
		t.Fatalf("expected created_at to be set")
	}

	if service.UpdatedAt == "" {
		t.Fatalf("expected updated_at to be set")
	}

	_, err = time.Parse(time.RFC3339, service.CreatedAt)
	if err != nil {
		t.Fatalf("expected created_at to use RFC3339 format, got %q", service.CreatedAt)
	}

	_, err = time.Parse(time.RFC3339, service.UpdatedAt)
	if err != nil {
		t.Fatalf("expected updated_at to use RFC3339 format, got %q", service.UpdatedAt)
	}

}

func TestListServicesIncludesTimestamps(t *testing.T) {
	resetStore()
	var services []Service

	body := `{"id":"svc-time","owner":"david","name":"david chen"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/services", nil)
	rec = httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected code %d, got %d", http.StatusOK, rec.Code)
	}

	err := json.NewDecoder(rec.Body).Decode(&services)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", rec.Body)
	}

	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	_, err = time.Parse(time.RFC3339, services[0].CreatedAt)
	if err != nil {
		t.Fatalf("expected created_at to use RFC3339 format, got %q", services[0].CreatedAt)
	}

	_, err = time.Parse(time.RFC3339, services[0].UpdatedAt)
	if err != nil {
		t.Fatalf("expected updated_at to use RFC3339 format, got %q", services[0].UpdatedAt)
	}

}

func TestCreateServiceDefaultsTier(t *testing.T) {
	resetStore()

	body := `{"id":"svc-tier","name":"Tiered Service","owner":"Platform Team"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected code %d, but got status code %d, the error body is %v", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response CreateServiceResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	service := response.Service
	if err != nil {
		t.Fatalf("unable to decode the reponse body: %v", err)
	}

	if service.Tier != "tier-3" {
		t.Fatalf("expected default tier is %q, got %q", "tier-3", service.Tier)
	}

}

func TestCreateServiceInvalidTierReturns400(t *testing.T) {
	resetStore()

	body := `{"id":"svc-bad-tier","name":"Bad Tier Service","owner":"Platform Team","tier":"critical"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected code is %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateServiceDefaultsLanguage(t *testing.T) {
	resetStore()

	body := `{"id":"svc-lang","name":"Language Service","owner":"Platform Team"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response CreateServiceResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatalf("unable to parse the response result: %v", err)
	}

	service := response.Service

	if service.Language != "go" {
		t.Fatalf("expected default language is %q, got %q", "go", service.Language)
	}
}

func TestCreateServiceInvalidLanguageReturns400(t *testing.T) {
	resetStore()

	body := `{"id":"svc-bad-lang","name":"Bad Language Service","owner":"Platform Team","language":"ruby"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected code %d, got %d, body is %q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	// POST /services
	// call serviceHandler
	// expect http.StatusBadRequest
}

func TestCreateServiceInvalidRepoURLReturns400(t *testing.T) {
	resetStore()

	body := `{"id":"svc-bad-repo","name":"Bad Repo Service","owner":"Platform Team","repo_url":"not-a-url"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected code %d, got %d, body is: %q", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	// call serviceHandler
	// expect http.StatusBadRequest
}

func TestCreateServiceInvalidSlackChannelReturns400(t *testing.T) {
	resetStore()

	body := `{"id":"svc-bad-slack","name":"Bad Slack Service","owner":"Platform Team","slack_channel":"platform-alerts"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected code %d, got %d, the body is: %v", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestCreateServiceReturnsBootstrapManifest(t *testing.T) {
	resetStore()

	body := `{"id":"svc-bootstrap","name":"Bootstrap Service","owner":"Platform Team","language":"go","tier":"tier-1"}`

	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(body))
	rec := httptest.NewRecorder()

	serviceHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("exptected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var serviceResponse CreateServiceResponse

	err := json.NewDecoder(rec.Body).Decode(&serviceResponse)
	if err != nil {
		t.Fatalf("cannot parse the result response, got %v", err)
	}

	if serviceResponse.Service.ID != "svc-bootstrap" {
		t.Fatalf("expected to have id %q, got %q", "svc-boostrap", serviceResponse.Service.ID)
	}

	if serviceResponse.Bootstrap.RepoTemplate != "go-service-template" {
		t.Fatalf("expected to have id %q, got %q", "svc-boostrap", serviceResponse.Bootstrap.RepoTemplate)
	}

	if serviceResponse.Bootstrap.CIWorkflow != "github-actions-go" {
		t.Fatalf("expected to have id %q, got %q", "svc-boostrap", serviceResponse.Bootstrap.CIWorkflow)
	}

	if !serviceResponse.Bootstrap.Dockerfile {
		t.Fatalf("expected dockerfile to be enabled")
	}

	if serviceResponse.Bootstrap.HealthCheckPath != "/health" {
		t.Fatalf("expted health check path %q, got %q", "/health", serviceResponse.Bootstrap.HealthCheckPath)
	}

	if serviceResponse.Bootstrap.Dashboard != "default-service-dashboard" {
		t.Fatalf("expected bootstrap dashboard %q, got %q", "default-service-dashboard", serviceResponse.Bootstrap.Dashboard)
	}

	if serviceResponse.Bootstrap.MetricsPath != "/metrics" {
		t.Fatalf("exptected metrics path %q, got %q", "/metrics", serviceResponse.Bootstrap.MetricsPath)
	}

	if len(serviceResponse.Bootstrap.Alerts) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(serviceResponse.Bootstrap.Alerts))
	}
}
