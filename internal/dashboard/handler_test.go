package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrcl-oss/checkIndex/report"
)

func TestHandlerRendersEmptyStateAndFreshness(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{Collections: map[string]report.Collection{}}, nil
	}, time.Minute)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	snapshot := service.Snapshot()

	recorder := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := recorder.Body.String()
	for _, text := range []string{
		"No index usage data is available yet.",
		"Data freshness",
		"Last attempt",
		"Last successful collection",
		"name=\"viewport\"",
		"http-equiv=\"refresh\" content=\"60\"",
		snapshot.LastAttempt.UTC().Format(time.RFC3339),
		snapshot.LastSuccess.UTC().Format(time.RFC3339),
	} {
		if !strings.Contains(body, text) {
			t.Errorf("response does not contain %q", text)
		}
	}
}

func TestHandlerUsesConfiguredRefreshInterval(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{}, nil
	}, time.Minute)
	recorder := httptest.NewRecorder()
	NewHandlerWithRefreshInterval(service, 15*time.Second).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `http-equiv="refresh" content="15"`) {
		t.Fatal("response does not use the configured refresh interval")
	}
}

// A <meta http-equiv="refresh"> is scheduled by the browser as soon as it is
// parsed; removing the element later from JavaScript does not cancel it. The
// full-page reload would discard the user's expanded/collapsed sections, so the
// meta refresh must only apply when JavaScript (live refresh) is unavailable.
func TestHandlerMetaRefreshOnlyAppliesWithoutJavaScript(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{}, nil
	}, time.Minute)
	recorder := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	body := recorder.Body.String()
	const wrapped = `<noscript><meta http-equiv="refresh" content="60"></noscript>`
	if !strings.Contains(body, wrapped) {
		t.Fatalf("meta refresh must be wrapped in <noscript>; want %q in response", wrapped)
	}
	if strings.Count(body, `http-equiv="refresh"`) != 1 {
		t.Fatal("response must contain exactly one meta refresh declaration")
	}
}

func TestHandlerRendersStatusEscapedNamesAndNoRawErrors(t *testing.T) {
	const rawError = "connection failed: mongodb://alice:top-secret@db.example"
	const clusterName = "<script>alert(1)</script>"
	const collectionName = "orders<&>"
	const indexName = "idx<&>"
	const nodeName = "<img src=x onerror=alert(1)>"
	partialReport := report.Report{
		Collections: map[string]report.Collection{
			collectionName: {
				indexName: {
					Clusters: map[string]report.Cluster{
						clusterName: {
							Nodes: map[string]report.NodeUsage{
								nodeName: {Count: 5_000_000_000, Primary: false},
							},
							Total: 5_000_000_000,
						},
					},
					Total: 5_000_000_000,
				},
			},
		},
		Partial: true,
	}
	attempt := 0
	service := NewService(func(context.Context) (report.Report, error) {
		attempt++
		switch attempt {
		case 1:
			complete := partialReport
			complete.Partial = false
			return complete, nil
		case 2:
			return partialReport, errors.New("one node did not respond")
		default:
			return report.Report{Partial: true}, errors.New(rawError)
		}
	}, time.Minute)
	for range 3 {
		_ = service.Refresh(context.Background())
	}
	snapshot := service.Snapshot()

	recorder := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	for _, banner := range []string{
		"The report contains partial data.",
		"The latest collection attempt failed.",
		"The displayed report may be stale.",
	} {
		if !strings.Contains(body, banner) {
			t.Errorf("response does not contain status banner %q", banner)
		}
	}
	for _, name := range []string{clusterName, collectionName, indexName, nodeName} {
		if !strings.Contains(body, html.EscapeString(name)) {
			t.Errorf("response does not contain escaped name %q", name)
		}
		if strings.Contains(body, name) {
			t.Errorf("response contains unescaped name %q", name)
		}
	}
	if strings.Contains(body, rawError) || strings.Contains(body, "top-secret") {
		t.Fatal("response exposed raw collection error details")
	}
	for _, timestamp := range []time.Time{snapshot.LastAttempt, snapshot.LastSuccess} {
		if !strings.Contains(body, timestamp.UTC().Format(time.RFC3339)) {
			t.Errorf("response does not contain freshness timestamp %s", timestamp.UTC().Format(time.RFC3339))
		}
	}
	if !strings.Contains(body, "Non-primary") || strings.Contains(body, "Secondary") {
		t.Fatal("response did not use the safe non-primary role label")
	}
	if !strings.Contains(body, "Operations") || !strings.Contains(body, "Role") {
		t.Fatal("response is missing semantic table headings")
	}
}

func TestHandlerRejectsNonGetMethods(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{}, nil
	}, time.Minute)
	recorder := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestHandlerRendersCollapsibleSectionsAndFilterInput(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{
			Collections: map[string]report.Collection{
				"orders": {
					"customer_id": {
						Clusters: map[string]report.Cluster{
							"cluster-north": {
								Nodes: map[string]report.NodeUsage{
									"node-1": {Count: 10, Primary: true},
								},
								Total: 10,
							},
						},
						Total: 10,
					},
				},
			},
		}, nil
	}, time.Minute)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	NewHandler(service).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, `id="filter-input"`) {
		t.Error("response does not contain the filter input element")
	}

	expectedSnippets := []string{
		`<details id="cluster-0" class="cluster" data-name="cluster-north" data-path="cluster:cluster-north">`,
		`<details id="cluster-0-col-0" class="collection" data-name="orders" data-path="col:cluster-north/orders">`,
		`<details id="cluster-0-col-0-idx-0" class="index" data-name="customer_id" data-path="idx:cluster-north/orders/customer_id">`,
		`<summary`,
		`id="filter-case"`,
		`id="filter-exact"`,
		`id="refresh-interval-select"`,
		`<option value="600000">10 min</option>`,
		`<option value="300000" selected>5 min</option>`,
		`<option value="60000">1 min</option>`,
		`<option value="30000">30s</option>`,
		`<option value="10000">10s</option>`,
		`<option value="0">Never (reload only)</option>`,
		`<nav class="sidebar-toc"`,
		`<a href="#cluster-0"`,
		`<a href="#cluster-0-col-0"`,
		`<a id="toc-cluster-0-col-0-idx-0" href="#cluster-0-col-0-idx-0"`,
		`border-left: 5px solid #2563eb;`,
		`border-left: 5px solid #7c3aed;`,
		`border-left: 5px solid #0d9488;`,
		`data-action="toggle-all-clusters"`,
		`data-action="toggle-cluster-collections"`,
		`data-action="toggle-collection-indexes"`,
		`data-action="toggle-toc-clusters"`,
		`data-action="toggle-toc-collections"`,
		`scroll-padding-top: calc(var(--filter-bar-height) + 1rem);`,
		`scroll-margin-top: calc(var(--filter-bar-height) + 1rem);`,
		`width: max-content;`,
		`initDataRefresh`,
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(body, snippet) {
			t.Errorf("response does not contain expected snippet %q", snippet)
		}
	}

	if strings.Contains(body, `class="cluster" open`) ||
		strings.Contains(body, `class="collection" open`) ||
		strings.Contains(body, `class="index" open`) ||
		strings.Contains(body, `class="toc-cluster" open`) ||
		strings.Contains(body, `class="toc-collection" open`) {
		t.Error("collapsible sections must not have the 'open' attribute by default")
	}
}

func TestHandlerServesJSONSnapshot(t *testing.T) {
	service := NewService(func(context.Context) (report.Report, error) {
		return report.Report{
			Collections: map[string]report.Collection{
				"users": {
					"id_1": {
						Clusters: map[string]report.Cluster{
							"cluster-a": {
								Nodes: map[string]report.NodeUsage{
									"node-a": {Count: 42, Primary: true},
								},
								Total: 42,
							},
						},
						Total: 42,
					},
				},
			},
		}, nil
	}, time.Minute)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/?format=json", nil),
		func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Accept", "application/json")
			return r
		}(),
	} {
		recorder := httptest.NewRecorder()
		NewHandler(service).ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
		body := recorder.Body.String()
		if !strings.Contains(body, `"cluster-a"`) || !strings.Contains(body, `"users"`) || !strings.Contains(body, `"id_1"`) {
			t.Errorf("JSON response %q missing expected cluster/collection/index keys", body)
		}
	}
}

func TestHandlerDownloadsCachedRawReport(t *testing.T) {
	collectionReport := report.Report{
		Collections: map[string]report.Collection{
			"users": {
				"id_1": {
					Clusters: map[string]report.Cluster{
						"cluster-a": {
							Nodes: map[string]report.NodeUsage{
								"node-a": {Count: 42, Primary: true},
							},
							Total: 42,
						},
					},
					Total: 42,
				},
			},
		},
		Partial: true,
	}
	collectorCalls := 0
	service := NewService(func(context.Context) (report.Report, error) {
		collectorCalls++
		return collectionReport, nil
	}, time.Minute)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	handler := NewHandler(service)
	download := httptest.NewRecorder()
	handler.ServeHTTP(download, httptest.NewRequest(http.MethodGet, "/download/report.json", nil))
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want %d", download.Code, http.StatusOK)
	}
	if got := download.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("download Content-Type = %q", got)
	}
	if got := download.Header().Get("Content-Disposition"); got != `attachment; filename="checkindex-report.json"` {
		t.Errorf("download Content-Disposition = %q", got)
	}
	expected, err := json.Marshal(collectionReport)
	if err != nil {
		t.Fatalf("json.Marshal(report) error = %v", err)
	}
	if got := strings.TrimSpace(download.Body.String()); got != string(expected) {
		t.Errorf("download body = %s, want raw report JSON %s", got, expected)
	}
	if collectorCalls != 1 {
		t.Errorf("collector calls = %d, want 1 (download must use cached report)", collectorCalls)
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", page.Code, http.StatusOK)
	}
	if !strings.Contains(page.Body.String(), `href="/download/report.json"`) {
		t.Error("dashboard is missing the report download control")
	}

	emptyService := NewService(func(context.Context) (report.Report, error) {
		return report.Report{Collections: map[string]report.Collection{}}, nil
	}, time.Minute)
	if err := emptyService.Refresh(context.Background()); err != nil {
		t.Fatalf("empty Refresh() error = %v", err)
	}
	emptyDownload := httptest.NewRecorder()
	NewHandler(emptyService).ServeHTTP(emptyDownload, httptest.NewRequest(http.MethodGet, "/download/report.json", nil))
	if got := strings.TrimSpace(emptyDownload.Body.String()); got != `{}` {
		t.Errorf("empty report body = %q, want {}", got)
	}
}

func TestHandlerConfigEndpoints(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	refreshed := make(chan struct{}, 1)
	service := NewService(func(context.Context) (report.Report, error) {
		select {
		case refreshed <- struct{}{}:
		default:
		}
		return report.Report{}, nil
	}, time.Minute)

	handler := NewHandlerWithConfig(service, time.Minute, configPath)

	// 1. GET /api/config returns 200 with starter config when file is missing
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/config code = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"dbName":"database"`) {
		t.Errorf("expected default dbName in %s", rec.Body.String())
	}

	// 2. POST /api/config rejects invalid JSON
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader("bad json")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST bad json code = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// 3. POST /api/config rejects invalid configuration (missing cluster)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(`{"dbName":"test","authSource":"admin","cluster":{}}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST empty cluster code = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// 4. POST /api/config saves valid configuration and triggers refresh
	validConfig := `{
		"dbName": "production_db",
		"authSource": "admin",
		"hostMapping": {"m1": "127.0.0.1:27017"},
		"cluster": {
			"primary": {
				"initialUri": "mongodb://127.0.0.1:27017",
				"username": "admin",
				"password": "secretpassword"
			}
		}
	}`
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(validConfig)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST valid config code = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify file was saved on disk
	dto, err := LoadConfigDTO(configPath)
	if err != nil {
		t.Fatalf("LoadConfigDTO() after save failed: %v", err)
	}
	if dto.DBName != "production_db" {
		t.Errorf("saved dbName = %q, want production_db", dto.DBName)
	}
	if !dto.Cluster["primary"].HasPassword {
		t.Errorf("expected HasPassword=true on saved profile")
	}

	// Verify PUT/DELETE on /api/config return 405 Method Not Allowed
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/config", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /api/config code = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
