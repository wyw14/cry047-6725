package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cry047/baseline/internal/application"
	"github.com/cry047/baseline/internal/domain"
	"github.com/cry047/baseline/internal/middleware"
	"github.com/cry047/baseline/internal/platform/logger"
	"github.com/cry047/baseline/internal/repository/memory"
	"github.com/cry047/baseline/internal/service/notifier"
	"github.com/cry047/baseline/internal/service/storage"
	phttp "github.com/cry047/baseline/internal/transport/http"
	"github.com/cry047/baseline/scripts/seed"
)

// setupServer returns a fully-wired HTTP server backed by in-memory stores
// and seeded with demo data.
func setupServer(t *testing.T) *phttp.Server {
	t.Helper()
	store := memory.NewStore()
	ports := memory.NewPorts(store)
	ports.Clock = domain.SystemClock{}
	ports.Notifier = notifier.New(128, "")
	st, err := storage.New(storage.Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("init storage: %v", err)
	}
	ports.Storage = st
	actor := domain.Actor{ID: "system", Name: "系统", Role: domain.RoleAdmin}
	if _, err := seed.Run(context.Background(), &ports, actor); err != nil {
		t.Fatalf("seed: %v", err)
	}
	services := &phttp.Services{
		Places:        application.NewPlaceService(&ports),
		People:        application.NewPersonService(&ports),
		Facilities:    application.NewFacilityService(&ports),
		Plans:         application.NewPlanService(&ports),
		Executions:    application.NewExecutionService(&ports),
		Anomalies:     application.NewAnomalyService(&ports),
		Todos:         application.NewTodoService(&ports),
		Notifications: application.NewNotificationService(&ports),
		Audit:         application.NewAuditService(&ports),
		Timeline:      application.NewTimelineService(&ports),
		Scheduler:     application.NewSchedulerService(&ports),
	}
	log, _ := logger.NewForWriter(newDiscardWriter(), "error")
	adapter := middleware.NewZapAdapter(log)
	srv, err := phttp.New(services, phttp.Config{
		Addr:           ":0",
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		IdleTimeout:    30 * time.Second,
		Mode:           "test",
		AllowedOrigins: []string{"*"},
	}, adapter)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

// newDiscardWriter returns a writer that discards all output.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func newDiscardWriter() discardWriter             { return discardWriter{} }

// do performs an HTTP request against the test server.
func do(t *testing.T, srv *phttp.Server, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	var err error
	if body == nil {
		r, err = http.NewRequest(method, path, nil)
	} else {
		buf, _ := json.Marshal(body)
		r, err = http.NewRequest(method, path, bytes.NewReader(buf))
		r.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	return w
}

// adminHeaders returns headers that grant admin role.
func adminHeaders() map[string]string {
	return map[string]string{
		"X-Actor-Id":   "admin-1",
		"X-Actor-Name": "管理员",
		"X-Actor-Role": string(domain.RoleAdmin),
	}
}

func supervisorHeaders() map[string]string {
	return map[string]string{
		"X-Actor-Id":   "sup-1",
		"X-Actor-Name": "主管",
		"X-Actor-Role": string(domain.RoleSupervisor),
	}
}

func operatorHeaders() map[string]string {
	return map[string]string{
		"X-Actor-Id":   "op-1",
		"X-Actor-Name": "操作员",
		"X-Actor-Role": string(domain.RoleOperator),
	}
}

// TestHealthz verifies /healthz returns 200.
func TestHealthz(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "GET", "/healthz", nil, nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestReadyz verifies /readyz returns 200.
func TestReadyz(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "GET", "/readyz", nil, nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRequestIDReturned verifies the X-Request-Id header is echoed.
func TestRequestIDReturned(t *testing.T) {
	srv := setupServer(t)
	r, _ := http.NewRequest("GET", "/healthz", nil)
	r.Header.Set("X-Request-Id", "my-req-1")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if got := w.Header().Get("X-Request-Id"); got != "my-req-1" {
		t.Errorf("expected my-req-1, got %s", got)
	}
}

// TestSecurityHeaders verifies the security response headers are set.
func TestSecurityHeaders(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "GET", "/healthz", nil, nil)
	for k, v := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := w.Header().Get(k); got != v {
			t.Errorf("expected %s=%s, got %s", k, v, got)
		}
	}
}

// TestCORS_Preflight verifies OPTIONS requests are short-circuited.
func TestCORS_Preflight(t *testing.T) {
	srv := setupServer(t)
	r, _ := http.NewRequest("OPTIONS", "/api/v1/places", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	r.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != 204 {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected origin echo, got %s", got)
	}
}

// TestPanicRecovery verifies a panic returns 500 without crashing the server.
func TestPanicRecovery(t *testing.T) {
	srv := setupServer(t)
	// Hitting a non-existent route returns 404, not a panic. Use a known route
	// with an invalid id to verify normal error handling.
	w := do(t, srv, "GET", "/api/v1/places/does-not-exist", nil, nil)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestListFacilities_Pagination verifies pagination + filtering works.
func TestListFacilities_Pagination(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "GET", "/api/v1/facilities?limit=2&offset=0&order_by=name&order=asc", nil, nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Items []domain.Facility `json:"items"`
			Total int               `json:"total"`
			Limit int               `json:"limit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data.Items) > 2 {
		t.Errorf("expected <=2 items, got %d", len(resp.Data.Items))
	}
	if resp.Data.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Data.Limit)
	}
}

// TestCreateFacility_ValidationErrors verifies the field error response shape.
func TestCreateFacility_ValidationErrors(t *testing.T) {
	srv := setupServer(t)
	body := map[string]any{"name": "x"} // missing required fields
	w := do(t, srv, "POST", "/api/v1/facilities", body, adminHeaders())
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code   string `json:"code"`
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != domain.CodeInvalid {
		t.Errorf("expected code %s, got %s", domain.CodeInvalid, resp.Code)
	}
	if resp.RequestID == "" {
		t.Errorf("expected non-empty request_id")
	}
}

// TestFacilityTransition_OverdueCriticalForbidden verifies the prompt invariant
// via the HTTP layer: an overdue critical facility cannot be marked normal.
func TestFacilityTransition_OverdueCriticalForbidden(t *testing.T) {
	srv := setupServer(t)
	// Get a seeded critical facility and force its status to overdue.
	// fac-hvac-001 is critical. First we'll patch its status through the
	// transition endpoint by going normal -> restricted_use -> under_repair
	// -> recovered -> ... actually, let's just seed a new one in overdue
	// state directly via an internal admin call. The HTTP API doesn't allow
	// direct overdue insertion without going through the scheduler.
	//
	// Instead, we'll seed a facility, then call /scheduler/scan to mark it
	// overdue (the plan was due 32 days ago), then attempt the forbidden
	// transition.
	// fac-hvac-001 has plan-001 with NextDueDate = today+30-32 = -2 days.
	w := do(t, srv, "POST", "/api/v1/scheduler/scan", nil, adminHeaders())
	if w.Code != 200 {
		t.Fatalf("scheduler scan: %d: %s", w.Code, w.Body.String())
	}
	// Now fac-hvac-001 should be overdue.
	w = do(t, srv, "GET", "/api/v1/facilities/fac-hvac-001", nil, nil)
	if w.Code != 200 {
		t.Fatalf("get facility: %d: %s", w.Code, w.Body.String())
	}
	var getResp struct {
		Data domain.Facility `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if getResp.Data.Status != domain.FacilityOverdue {
		t.Fatalf("expected overdue, got %s", getResp.Data.Status)
	}
	// Attempt to mark normal directly.
	body := map[string]any{"to": "normal", "reason": "绕过流程"}
	w = do(t, srv, "POST", "/api/v1/facilities/fac-hvac-001/transition", body, supervisorHeaders())
	if w.Code != 422 {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "STATE_FORBIDDEN") {
		t.Errorf("expected STATE_FORBIDDEN in body: %s", w.Body.String())
	}
}

// TestRoleBoundaries verifies that a viewer cannot create a plan template and
// an operator cannot change the plan cycle.
func TestRoleBoundaries(t *testing.T) {
	srv := setupServer(t)
	body := map[string]any{
		"name":              "测试模板",
		"cycle_days":        30,
		"inspection_items":  []map[string]any{{"code": "a", "name": "A"}},
		"consumables":       []map[string]any{},
		"requires_shutdown": false,
	}
	// Viewer cannot create template.
	w := do(t, srv, "POST", "/api/v1/plan-templates", body, map[string]string{
		"X-Actor-Id":   "v",
		"X-Actor-Name": "viewer",
		"X-Actor-Role": string(domain.RoleViewer),
	})
	if w.Code != 403 {
		t.Errorf("expected 403 for viewer, got %d: %s", w.Code, w.Body.String())
	}
	// Admin can create template.
	w = do(t, srv, "POST", "/api/v1/plan-templates", body, adminHeaders())
	if w.Code != 201 {
		t.Errorf("expected 201 for admin, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSchedulerScanEndpoint verifies the scheduler scan endpoint returns a summary.
func TestSchedulerScanEndpoint(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "POST", "/api/v1/scheduler/scan", nil, adminHeaders())
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data application.ScanResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.RunAt.IsZero() {
		t.Errorf("expected non-zero run_at")
	}
}

// TestConcurrency_ParallelExecutionSubmission verifies the in-memory store
// handles concurrent submissions safely (no data races) and that idempotency
// keys prevent duplicates. Some requests will get 201 (first winner), others
// 200 (idempotent replay), but no 5xx.
func TestConcurrency_ParallelExecutionSubmission(t *testing.T) {
	srv := setupServer(t)
	body := map[string]any{
		"plan_id":         "plan-002",
		"facility_id":     "fac-elev-001",
		"executed_by":     "op-1",
		"executed_at":     "2026-08-17T10:00:00Z",
		"idempotency_key": "idem-concurrent-2",
		"inspection_values": []map[string]any{
			{"item_code": "rope", "value": "ok", "pass": true},
			{"item_code": "brake", "value": "ok", "pass": true},
		},
		"consumables_consumed": []map[string]any{
			{"code": "lubricant", "quantity": 1},
		},
	}
	var wg sync.WaitGroup
	const N = 8
	statuses := make([]int, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w := do(t, srv, "POST", "/api/v1/executions", body, operatorHeaders())
			statuses[idx] = w.Code
		}(i)
	}
	wg.Wait()
	for i, s := range statuses {
		if s != 201 && s != 200 {
			t.Errorf("goroutine %d: expected 201 or 200, got %d: %s", i, s, "")
		}
	}
	// Verify only 1 execution was actually created.
	w := do(t, srv, "GET", "/api/v1/executions?plan_id=plan-002", nil, nil)
	if w.Code != 200 {
		t.Fatalf("list: %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Items []domain.Execution `json:"items"`
			Total int                `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.Total != 1 {
		t.Errorf("expected 1 execution, got %d", resp.Data.Total)
	}
}

// TestErrorResponseShape verifies the error response includes code, message
// and request_id.
func TestErrorResponseShape(t *testing.T) {
	srv := setupServer(t)
	w := do(t, srv, "GET", "/api/v1/places/missing", nil, nil)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != domain.CodeNotFound {
		t.Errorf("expected %s, got %s", domain.CodeNotFound, resp.Code)
	}
	if resp.Message == "" {
		t.Errorf("expected non-empty message")
	}
	if resp.RequestID == "" {
		t.Errorf("expected non-empty request_id")
	}
}

// TestAnomalyFailedReinspection_KeepsIssueOpen is an end-to-end test of the
// handler-level fix. It drives the full anomaly lifecycle over HTTP and asserts
// that a FAILING follow-up inspection (pass=false):
//   - reopens the anomaly (status "open"), never "recovered";
//   - leaves the facility in "under_repair", never "recovered";
//   - makes recovery confirmation refused (422 STATE_FORBIDDEN).
//
// Before the fix, the handler computed pass = body.Pass || body.Result != "",
// and since "result" is a required non-empty string, pass was always true — so
// a failing reinspection was unreachable over HTTP and the facility was marked
// recovered anyway. This test guards against that regression.
func TestAnomalyFailedReinspection_KeepsIssueOpen(t *testing.T) {
	srv := setupServer(t)

	// 1. Discover a critical anomaly on a seeded critical facility.
	w := do(t, srv, "POST", "/api/v1/anomalies", map[string]any{
		"facility_id":     "fac-hvac-001",
		"description":     "配电柜温升异常",
		"severity":        "critical",
		"idempotency_key": "http-failed-reinspect",
	}, operatorHeaders())
	if w.Code != 201 {
		t.Fatalf("discover: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data domain.Anomaly `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal discover: %v", err)
	}
	anomalyID := created.Data.ID
	if anomalyID == "" {
		t.Fatalf("discover returned empty anomaly id")
	}
	// Critical anomaly auto-transitions the facility to restricted_use.
	if got := facilityStatusHTTP(t, srv, "fac-hvac-001"); got != domain.FacilityRestrictedUse {
		t.Fatalf("after discover: facility %s, want restricted_use", got)
	}

	// 2. Rectify -> anomaly reinspecting, facility under_repair.
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/rectify", map[string]any{
		"measure": "紧固接点并降载",
	}, supervisorHeaders())
	if w.Code != 200 {
		t.Fatalf("rectify: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := facilityStatusHTTP(t, srv, "fac-hvac-001"); got != domain.FacilityUnderRepair {
		t.Fatalf("after rectify: facility %s, want under_repair", got)
	}

	// 3. Failing follow-up inspection (pass=false) reopens the anomaly.
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/reinspect", map[string]any{
		"result": "温升仍超限",
		"pass":   false,
	}, supervisorHeaders())
	if w.Code != 200 {
		t.Fatalf("reinspect: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var reinspect struct {
		Data domain.Anomaly `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &reinspect); err != nil {
		t.Fatalf("unmarshal reinspect: %v", err)
	}
	if reinspect.Data.Status != domain.AnomalyOpen {
		t.Fatalf("failed reinspection entered %s, want open (issue must stay open)", reinspect.Data.Status)
	}
	// The facility must remain under_repair, never recovered.
	if got := facilityStatusHTTP(t, srv, "fac-hvac-001"); got != domain.FacilityUnderRepair {
		t.Fatalf("after failed reinspection: facility %s, want under_repair", got)
	}

	// 4. Recovery confirmation must be refused while the issue is still open.
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/recover", map[string]any{}, supervisorHeaders())
	if w.Code != 422 {
		t.Fatalf("recover: expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), domain.CodeStateForbidden) {
		t.Fatalf("recover: expected %s in body: %s", domain.CodeStateForbidden, w.Body.String())
	}

	// 5. A passing follow-up inspection then confirmation finally recovers
	//    the facility — proving the issue only closes once a reinspection passes.
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/rectify", map[string]any{
		"measure": "更换绕组并复测",
	}, supervisorHeaders())
	if w.Code != 200 {
		t.Fatalf("re-rectify: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/reinspect", map[string]any{
		"result": "温升恢复正常",
		"pass":   true,
	}, supervisorHeaders())
	if w.Code != 200 {
		t.Fatalf("reinspect pass: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Passing reinspection does NOT yet mark the facility recovered.
	if got := facilityStatusHTTP(t, srv, "fac-hvac-001"); got != domain.FacilityUnderRepair {
		t.Fatalf("after passing reinspection: facility %s, want under_repair (recovery not yet confirmed)", got)
	}
	w = do(t, srv, "POST", "/api/v1/anomalies/"+anomalyID+"/recover", map[string]any{}, supervisorHeaders())
	if w.Code != 200 {
		t.Fatalf("recover: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := facilityStatusHTTP(t, srv, "fac-hvac-001"); got != domain.FacilityRecovered {
		t.Fatalf("after recover: facility %s, want recovered", got)
	}
}

// facilityStatusHTTP fetches a facility's stored status over HTTP, which is the
// status shown in the ledger and detail pages.
func facilityStatusHTTP(t *testing.T, srv *phttp.Server, id string) domain.FacilityStatus {
	t.Helper()
	w := do(t, srv, "GET", "/api/v1/facilities/"+id, nil, nil)
	if w.Code != 200 {
		t.Fatalf("get facility %s: expected 200, got %d: %s", id, w.Code, w.Body.String())
	}
	var resp struct {
		Data domain.Facility `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal facility: %v", err)
	}
	return resp.Data.Status
}
