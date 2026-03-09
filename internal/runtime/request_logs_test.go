package runtime

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestClampRequestBufferBoundaries(t *testing.T) {
	if got := clampRequestBuffer(0); got != defaultRequestLogsBufferSize {
		t.Fatalf("expected default buffer %d, got %d", defaultRequestLogsBufferSize, got)
	}
	if got := clampRequestBuffer(3); got != minRequestLogsBufferSize {
		t.Fatalf("expected min buffer %d, got %d", minRequestLogsBufferSize, got)
	}
	if got := clampRequestBuffer(999999); got != maxRequestLogsBufferSize {
		t.Fatalf("expected max buffer %d, got %d", maxRequestLogsBufferSize, got)
	}
	if got := clampRequestBuffer(320); got != 320 {
		t.Fatalf("expected unchanged buffer 320, got %d", got)
	}
}

func TestRequestLogsEndpointRequiresAuthAndReturnsCapturedLogs(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "request-logs.db"))

	unauthRec := doRuntimeRequest(r, http.MethodGet, "/_mar/request-logs", "", "")
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d body=%s", unauthRec.Code, unauthRec.Body.String())
	}

	devCode := requestCodeAndReadDevCode(t, r, "owner@example.com")
	token := loginWithCodeAndReadToken(t, r, "owner@example.com", devCode)

	listRec := doRuntimeRequest(r, http.MethodGet, "/todos", "", token)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /todos, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	rec := doRuntimeRequest(r, http.MethodGet, "/_mar/request-logs?limit=20", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for request logs, got %d body=%s", rec.Code, rec.Body.String())
	}

	type loggedQuery struct {
		SQL    string `json:"sql"`
		Reason string `json:"reason"`
	}
	type loggedRequest struct {
		Path       string        `json:"path"`
		QueryCount int           `json:"queryCount"`
		Queries    []loggedQuery `json:"queries"`
	}
	type requestLogsPayload struct {
		Buffer        int             `json:"buffer"`
		TotalCaptured int             `json:"totalCaptured"`
		Logs          []loggedRequest `json:"logs"`
	}

	var payload requestLogsPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode request logs payload failed: %v body=%s", err, rec.Body.String())
	}
	if payload.Buffer < minRequestLogsBufferSize {
		t.Fatalf("unexpected buffer size: %d", payload.Buffer)
	}
	if payload.TotalCaptured < 1 {
		t.Fatalf("expected totalCaptured > 0, got %d", payload.TotalCaptured)
	}
	if len(payload.Logs) == 0 {
		t.Fatal("expected at least one request log")
	}

	foundTodoList := false
	foundNoPlaceholder := false
	for _, requestLog := range payload.Logs {
		if requestLog.Path == "/todos" {
			foundTodoList = true
			if requestLog.QueryCount < 1 {
				t.Fatalf("expected /todos to execute at least one query, got %d", requestLog.QueryCount)
			}
			if len(requestLog.Queries) == 0 {
				t.Fatal("expected /todos log to include query traces")
			}
			if !strings.Contains(strings.ToUpper(requestLog.Queries[0].SQL), "SELECT") {
				t.Fatalf("expected query trace to include SELECT SQL, got %q", requestLog.Queries[0].SQL)
			}
			if strings.Contains(requestLog.Queries[0].SQL, "$1") || strings.Contains(requestLog.Queries[0].SQL, "$2") {
				t.Fatalf("expected query trace to avoid synthetic placeholders, got %q", requestLog.Queries[0].SQL)
			}
			if strings.TrimSpace(requestLog.Queries[0].Reason) == "" {
				t.Fatalf("expected /todos query trace to include a reason, got %+v", requestLog.Queries[0])
			}
			foundNoPlaceholder = true
		}
	}
	if !foundTodoList {
		t.Fatalf("expected request log for /todos, got logs: %+v", payload.Logs)
	}
	if !foundNoPlaceholder {
		t.Fatalf("expected /todos query trace check to run, got logs: %+v", payload.Logs)
	}
}

func TestRequestLogsGiveReasonsToAllQueriesInEntityRequest(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "request-logs-entity-reasons.db"))
	devCode := requestCodeAndReadDevCode(t, r, "owner@example.com")
	token := loginWithCodeAndReadToken(t, r, "owner@example.com", devCode)

	rec := doRuntimeRequest(r, http.MethodGet, "/todos", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /todos, got %d body=%s", rec.Code, rec.Body.String())
	}

	logsRec := doRuntimeRequest(r, http.MethodGet, "/_mar/request-logs?limit=20", "", token)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for request logs, got %d body=%s", logsRec.Code, logsRec.Body.String())
	}

	type loggedQuery struct {
		SQL    string `json:"sql"`
		Reason string `json:"reason"`
	}
	type loggedRequest struct {
		Path    string        `json:"path"`
		Queries []loggedQuery `json:"queries"`
	}
	type requestLogsPayload struct {
		Logs []loggedRequest `json:"logs"`
	}

	var payload requestLogsPayload
	if err := json.Unmarshal(logsRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode request logs payload failed: %v body=%s", err, logsRec.Body.String())
	}

	foundTodoList := false
	foundListReason := false
	for _, requestLog := range payload.Logs {
		if requestLog.Path != "/todos" {
			continue
		}
		foundTodoList = true
		if len(requestLog.Queries) == 0 {
			t.Fatal("expected /todos request log to include query traces")
		}
		for _, query := range requestLog.Queries {
			if strings.TrimSpace(query.Reason) == "" {
				t.Fatalf("expected every /todos query to include a reason, got %+v", query)
			}
			if strings.Contains(strings.ToUpper(query.SQL), `FROM "TODOS"`) && query.Reason == "Load rows for the entity list" {
				foundListReason = true
			}
		}
	}
	if !foundTodoList {
		t.Fatalf("expected request log for /todos, got logs: %+v", payload.Logs)
	}
	if !foundListReason {
		t.Fatalf("expected entity-list reason for /todos queries, got logs: %+v", payload.Logs)
	}
}

func TestRequestLogsAddAuthQueryReasons(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "request-logs-auth-reasons.db"))
	devCode := requestCodeAndReadDevCode(t, r, "owner@example.com")

	rec := doRuntimeRequest(r, http.MethodPost, "/auth/login", `{"email":"owner@example.com","code":"`+devCode+`"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	token := readTokenFromLoginResponse(t, rec.Body.Bytes())
	logsRec := doRuntimeRequest(r, http.MethodGet, "/_mar/request-logs?limit=20", "", token)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for request logs, got %d body=%s", logsRec.Code, logsRec.Body.String())
	}

	type loggedQuery struct {
		SQL    string `json:"sql"`
		Reason string `json:"reason"`
	}
	type loggedRequest struct {
		Path    string        `json:"path"`
		Queries []loggedQuery `json:"queries"`
	}
	type requestLogsPayload struct {
		Logs []loggedRequest `json:"logs"`
	}

	var payload requestLogsPayload
	if err := json.Unmarshal(logsRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode request logs payload failed: %v body=%s", err, logsRec.Body.String())
	}

	foundLoginLog := false
	foundCodeReason := false
	foundSessionReason := false
	for _, requestLog := range payload.Logs {
		if requestLog.Path != "/auth/login" {
			continue
		}
		foundLoginLog = true
		for _, query := range requestLog.Queries {
			if strings.Contains(strings.ToUpper(query.SQL), "FROM MAR_AUTH_CODES") && query.Reason == "Verify the latest login code for this email" {
				foundCodeReason = true
			}
			if strings.Contains(strings.ToUpper(query.SQL), "INSERT INTO MAR_SESSIONS") && query.Reason == "Create a new authenticated session" {
				foundSessionReason = true
			}
		}
	}
	if !foundLoginLog {
		t.Fatalf("expected auth/login request log, got logs: %+v", payload.Logs)
	}
	if !foundCodeReason {
		t.Fatalf("expected auth/login query reason for auth codes lookup, got logs: %+v", payload.Logs)
	}
	if !foundSessionReason {
		t.Fatalf("expected auth/login query reason for session insert, got logs: %+v", payload.Logs)
	}
}

func TestRequestLogsEndpointMasksSensitiveValues(t *testing.T) {
	requireSQLite3(t)

	r := mustNewAuthRuntime(t, filepath.Join(t.TempDir(), "request-logs-masked.db"))
	devCode := requestCodeAndReadDevCode(t, r, "owner@example.com")
	token := loginWithCodeAndReadToken(t, r, "owner@example.com", devCode)

	rawEmail := "secret.owner@example.com"
	rawCode := "654321"
	rawToken := "tok_ABC123xyz987"

	r.requestLogs.add(requestLogEntry{
		Method:       http.MethodPost,
		Path:         "/auth/login",
		Route:        "/auth/login",
		Status:       http.StatusUnauthorized,
		DurationMs:   12.3,
		Timestamp:    "2026-03-04 10:10:10",
		QueryCount:   2,
		QueryTimeMs:  3.4,
		ErrorMessage: "Authorization: Bearer " + rawToken + " devCode: " + rawCode + " email: " + rawEmail,
		Queries: []requestLogQuery{
			{SQL: "SELECT * FROM mar_auth_codes WHERE email = '" + rawEmail + "' AND code = '" + rawCode + "'"},
			{SQL: "INSERT INTO mar_sessions (token, user_id, email) VALUES ('" + rawToken + "', 1, '" + rawEmail + "')"},
		},
	})

	rec := doRuntimeRequest(r, http.MethodGet, "/_mar/request-logs?limit=5", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for request logs, got %d body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	normalizedBody := strings.ReplaceAll(strings.ReplaceAll(body, "\\u003c", "<"), "\\u003e", ">")
	if strings.Contains(body, rawEmail) {
		t.Fatalf("expected email to be masked, got body=%s", body)
	}
	if strings.Contains(body, rawCode) {
		t.Fatalf("expected code to be masked, got body=%s", body)
	}
	if strings.Contains(body, rawToken) {
		t.Fatalf("expected token to be masked, got body=%s", body)
	}
	if !strings.Contains(normalizedBody, "<omitted>") {
		t.Fatalf("expected omitted marker in body=%s", body)
	}
	if !strings.Contains(normalizedBody, ", 1,") {
		t.Fatalf("expected non-sensitive SQL values to remain visible in body=%s", body)
	}
	if strings.Contains(normalizedBody, "<masked-email>") || strings.Contains(normalizedBody, "<masked>") {
		t.Fatalf("expected no legacy masked markers in body=%s", body)
	}
}

func requestCodeAndReadDevCode(t *testing.T, r *Runtime, email string) string {
	t.Helper()
	body := `{"email":"` + email + `"}`
	rec := doRuntimeRequest(r, http.MethodPost, "/auth/request-code", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("request-code failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		DevCode string `json:"devCode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode request-code response failed: %v body=%s", err, rec.Body.String())
	}
	if strings.TrimSpace(response.DevCode) == "" {
		t.Fatalf("expected devCode in response, got body=%s", rec.Body.String())
	}
	return response.DevCode
}

func loginWithCodeAndReadToken(t *testing.T, r *Runtime, email, code string) string {
	t.Helper()
	body := `{"email":"` + email + `","code":"` + code + `"}`
	rec := doRuntimeRequest(r, http.MethodPost, "/auth/login", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	return readTokenFromLoginResponse(t, rec.Body.Bytes())
}

func readTokenFromLoginResponse(t *testing.T, body []byte) string {
	t.Helper()
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode login response failed: %v body=%s", err, string(body))
	}
	if strings.TrimSpace(response.Token) == "" {
		t.Fatalf("expected token in login response, got body=%s", string(body))
	}
	return response.Token
}

func doRuntimeRequest(r *Runtime, method, path, body, token string) *httptest.ResponseRecorder {
	payload := bytes.NewBufferString(body)
	req := httptest.NewRequest(method, path, payload)
	if strings.TrimSpace(body) != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.handleHTTP(rec, req)
	return rec
}
