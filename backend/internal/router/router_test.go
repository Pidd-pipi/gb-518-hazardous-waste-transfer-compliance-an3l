package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/config"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/database"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/router"
	"github.com/gin-gonic/gin"
)

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
	Meta  struct {
		Total int64 `json:"total"`
	} `json:"meta"`
}

type record struct {
	ID               uint     `json:"id"`
	Code             string   `json:"code"`
	Status           string   `json:"status"`
	Version          uint     `json:"version"`
	QuantityKg       float64  `json:"quantityKg"`
	ReceivedWeightKg *float64 `json:"receivedWeightKg"`
	WeightDiffKg     *float64 `json:"weightDiffKg"`
	WeightDiffReason string   `json:"weightDiffReason"`
	ReceivedAt       *string  `json:"receivedAt"`
}

func TestRBACLinkedComplianceWorkflowAndAuditing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConfig(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, redisClient, err := database.Open(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if redisClient != nil {
		t.Fatal("test must use in-memory limiter without Redis")
	}
	engine := router.New(cfg, db, nil, logger)

	viewer := login(t, engine, "viewer")
	operator := login(t, engine, "operator")
	reviewer := login(t, engine, "reviewer")

	response, _ := request(t, engine, http.MethodGet, "/api/generators?page=1&pageSize=10", viewer, "", nil)
	assertStatus(t, response, http.StatusOK)
	response, _ = request(t, engine, http.MethodPost, "/api/manifests", viewer, "viewer-write", manifestPayload("TM-VIEWER", "CP-002"))
	assertStatus(t, response, http.StatusForbidden)
	response, _ = request(t, engine, http.MethodGet, "/api/audits", viewer, "", nil)
	assertStatus(t, response, http.StatusForbidden)

	response, body := request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-valid", manifestPayload("TM-ROUTER-001", "CP-002"))
	assertStatus(t, response, http.StatusCreated)
	manifest := decodeRecord(t, body)
	if manifest.Status != "draft" || response.Header.Get("X-Request-ID") != "manifest-create-valid" {
		t.Fatalf("unexpected manifest response: %+v request-id=%q", manifest, response.Header.Get("X-Request-ID"))
	}

	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", manifest.ID), operator, "manifest-skip", map[string]any{
		"status": "in_transit", "expectedVersion": manifest.Version, "reason": "must not skip submission",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", manifest.ID), operator, "manifest-submit", map[string]any{
		"status": "submitted", "expectedVersion": manifest.Version, "reason": "linked permits checked",
	})
	assertStatus(t, response, http.StatusOK)
	manifest = decodeRecord(t, body)
	if manifest.Status != "submitted" || manifest.Version != 2 {
		t.Fatalf("manifest transition was not persisted: %+v", manifest)
	}

	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", manifest.ID), operator, "manifest-stale", map[string]any{
		"status": "in_transit", "expectedVersion": uint(1), "reason": "stale client must conflict",
	})
	assertStatus(t, response, http.StatusConflict)

	response, body = request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-unverified", manifestPayload("TM-ROUTER-002", "CP-001"))
	assertStatus(t, response, http.StatusCreated)
	unverified := decodeRecord(t, body)
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", unverified.ID), operator, "manifest-block-unverified", map[string]any{
		"status": "submitted", "expectedVersion": unverified.Version, "reason": "must verify carrier first",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	response, body = request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-rejected", manifestPayload("TM-ROUTER-003", "CP-002"))
	assertStatus(t, response, http.StatusCreated)
	rejected := decodeRecord(t, body)
	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", rejected.ID), operator, "manifest-submit-rejected", map[string]any{
		"status": "submitted", "expectedVersion": rejected.Version, "reason": "linked permits checked",
	})
	assertStatus(t, response, http.StatusOK)
	rejected = decodeRecord(t, body)
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", rejected.ID), operator, "manifest-reject", map[string]any{
		"status": "rejected", "expectedVersion": rejected.Version, "reason": "destination permit mismatch",
	})
	assertStatus(t, response, http.StatusOK)
	response, body = request(t, engine, http.MethodPost, "/api/checks", operator, "check-create-rejected", checkPayload("CC-ROUTER-002", rejected.Code))
	assertStatus(t, response, http.StatusCreated)
	rejectedCheck := decodeRecord(t, body)
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", rejectedCheck.ID), reviewer, "rejected-manifest-pass", map[string]any{
		"status": "pass", "expectedVersion": rejectedCheck.Version, "reason": "a rejected manifest must not pass",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	// 联单发运后签收必须带实收重量；超重 5% 的签收转驳回并要求差异原因。
	response, body = request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-weigh", manifestPayload("TM-ROUTER-004", "CP-002"))
	assertStatus(t, response, http.StatusCreated)
	weighed := decodeRecord(t, body)
	for _, step := range []struct {
		status string
		reqid  string
	}{{status: "submitted", reqid: "weigh-submit"}, {status: "in_transit", reqid: "weigh-ship"}} {
		response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", weighed.ID), operator, step.reqid, map[string]any{
			"status": step.status, "expectedVersion": weighed.Version, "reason": "linked permits checked",
		})
		assertStatus(t, response, http.StatusOK)
		weighed = decodeRecord(t, body)
	}

	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", weighed.ID), operator, "receive-without-weight", map[string]any{
		"status": "received", "expectedVersion": weighed.Version, "reason": "actual weight is mandatory",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", weighed.ID), operator, "receive-overweight-no-reason", map[string]any{
		"status": "received", "expectedVersion": weighed.Version, "reason": "overweight receipt must be rejected with reason",
		"receivedWeightKg": 720.0,
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", weighed.ID), operator, "receive-overweight-rejected", map[string]any{
		"status": "received", "expectedVersion": weighed.Version, "reason": "site weighing exceeds plan",
		"receivedWeightKg": 720.0, "weightDiffReason": "现场地磅复磅超出计划 5% 以上，车辆暂扣待复核",
	})
	assertStatus(t, response, http.StatusOK)
	weighed = decodeRecord(t, body)
	if weighed.Status != "rejected" || weighed.ReceivedWeightKg == nil || *weighed.ReceivedWeightKg != 720.0 ||
		weighed.WeightDiffKg == nil || *weighed.WeightDiffKg != 39.5 || weighed.WeightDiffReason == "" || weighed.ReceivedAt == nil {
		t.Fatalf("overweight receipt should be rejected with captured weight/diff/reason: %+v", weighed)
	}

	// 差异在 5% 以内的正常签收：保存实收重量、差值与签收时间。
	response, body = request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-receive", manifestPayload("TM-ROUTER-005", "CP-002"))
	assertStatus(t, response, http.StatusCreated)
	receivable := decodeRecord(t, body)
	for _, step := range []struct {
		status string
		reqid  string
	}{{status: "submitted", reqid: "receive-submit"}, {status: "in_transit", reqid: "receive-ship"}} {
		response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", receivable.ID), operator, step.reqid, map[string]any{
			"status": step.status, "expectedVersion": receivable.Version, "reason": "linked permits checked",
		})
		assertStatus(t, response, http.StatusOK)
		receivable = decodeRecord(t, body)
	}
	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", receivable.ID), operator, "receive-in-tolerance", map[string]any{
		"status": "received", "expectedVersion": receivable.Version, "reason": "site weight matches plan",
		"receivedWeightKg": 685.0,
	})
	assertStatus(t, response, http.StatusOK)
	receivable = decodeRecord(t, body)
	if receivable.Status != "received" || receivable.ReceivedWeightKg == nil || *receivable.ReceivedWeightKg != 685.0 ||
		receivable.WeightDiffKg == nil || *receivable.WeightDiffKg != 4.5 || receivable.ReceivedAt == nil {
		t.Fatalf("receipt should persist actual weight, diff and received time: %+v", receivable)
	}
	// 签收为终态，不允许再迁移。
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", receivable.ID), operator, "receive-final", map[string]any{
		"status": "rejected", "expectedVersion": receivable.Version, "reason": "received manifests are final",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	response, body = request(t, engine, http.MethodPost, "/api/checks", operator, "check-create", checkPayload("CC-ROUTER-001", receivable.Code))
	assertStatus(t, response, http.StatusCreated)
	check := decodeRecord(t, body)
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", check.ID), operator, "operator-decision", map[string]any{
		"status": "pass", "expectedVersion": check.Version, "reason": "operator must not decide",
	})
	assertStatus(t, response, http.StatusForbidden)

	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", check.ID), reviewer, "reviewer-decision", map[string]any{
		"status": "pass", "expectedVersion": check.Version, "reason": "all four evidence groups verified",
	})
	assertStatus(t, response, http.StatusOK)
	check = decodeRecord(t, body)
	if check.Status != "pass" || check.Version != 2 {
		t.Fatalf("review decision was not persisted: %+v", check)
	}

	// 未签收（在途）联单的核验不能通过。
	response, body = request(t, engine, http.MethodPost, "/api/manifests", operator, "manifest-create-intransit-check", manifestPayload("TM-ROUTER-006", "CP-002"))
	assertStatus(t, response, http.StatusCreated)
	inTransit := decodeRecord(t, body)
	for _, step := range []struct {
		status string
	}{{status: "submitted"}, {status: "in_transit"}} {
		response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/manifests/%d/transition", inTransit.ID), operator, "intransit-step", map[string]any{
			"status": step.status, "expectedVersion": inTransit.Version, "reason": "moving in transit for check gate",
		})
		assertStatus(t, response, http.StatusOK)
		inTransit = decodeRecord(t, body)
	}
	response, body = request(t, engine, http.MethodPost, "/api/checks", operator, "check-create-intransit", checkPayload("CC-ROUTER-003", inTransit.Code))
	assertStatus(t, response, http.StatusCreated)
	pendingCheck := decodeRecord(t, body)
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", pendingCheck.ID), reviewer, "pass-before-receipt", map[string]any{
		"status": "pass", "expectedVersion": pendingCheck.Version, "reason": "must not pass before manifest receipt",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)

	// 已驳回联单的核验：通过被拒，只能不通过，不通过后可升级复核。
	response, _ = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", rejectedCheck.ID), reviewer, "rejected-check-pass", map[string]any{
		"status": "pass", "expectedVersion": rejectedCheck.Version, "reason": "rejected manifest cannot pass",
	})
	assertStatus(t, response, http.StatusUnprocessableEntity)
	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", rejectedCheck.ID), reviewer, "rejected-check-fail", map[string]any{
		"status": "fail", "expectedVersion": rejectedCheck.Version, "reason": "manifest rejected for overweight receipt",
	})
	assertStatus(t, response, http.StatusOK)
	rejectedCheck = decodeRecord(t, body)
	response, body = request(t, engine, http.MethodPost, fmt.Sprintf("/api/checks/%d/transition", rejectedCheck.ID), reviewer, "rejected-check-escalate", map[string]any{
		"status": "escalated", "expectedVersion": rejectedCheck.Version, "reason": "escalate rejected manifest for senior review",
	})
	assertStatus(t, response, http.StatusOK)
	if decodeRecord(t, body).Status != "escalated" {
		t.Fatalf("failed check on rejected manifest should be escalatable")
	}

	response, body = request(t, engine, http.MethodGet, "/api/audits?page=1&pageSize=100", reviewer, "audit-read", nil)
	assertStatus(t, response, http.StatusOK)
	if !bytes.Contains(body, []byte("manifest-submit")) || !bytes.Contains(body, []byte("reviewer-decision")) {
		t.Fatalf("expected request IDs in immutable audit list: %s", string(body))
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Meta.Total < 5 {
		t.Fatalf("expected audited mutations, got total=%d error=%v", envelope.Meta.Total, err)
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		AppName: "hazardous-waste-transfer-compliance-test", Environment: "test", Port: "0",
		DatabaseDriver: "sqlite", DatabaseDSN: "file:router-test?mode=memory&cache=shared",
		JWTSecret: "router-test-secret-at-least-32-characters", TokenTTL: time.Hour,
		RequestLimit: 10000, StartupTimeout: 5 * time.Second, ShutdownTimeout: 5 * time.Second,
		ReadHeaderTimeout: time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 10 * time.Second,
	}
}

func login(t *testing.T, engine http.Handler, username string) string {
	t.Helper()
	response, body := request(t, engine, http.MethodPost, "/api/auth/login", "", "", map[string]any{
		"username": username, "password": "Admin123!",
	})
	assertStatus(t, response, http.StatusOK)
	var envelope struct {
		Data struct {
			Token string `json:"token"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data.Token == "" || envelope.Data.Role != username {
		t.Fatalf("login %s failed: role=%q error=%v body=%s", username, envelope.Data.Role, err, string(body))
	}
	return envelope.Data.Token
}

func request(t *testing.T, engine http.Handler, method, path, token, requestID string, payload any) (*http.Response, []byte) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	engine.ServeHTTP(recorder, req)
	return recorder.Result(), recorder.Body.Bytes()
}

func assertStatus(t *testing.T, response *http.Response, expected int) {
	t.Helper()
	if response.StatusCode != expected {
		t.Fatalf("expected HTTP %d, got %d", expected, response.StatusCode)
	}
}

func decodeRecord(t *testing.T, body []byte) record {
	t.Helper()
	var envelope struct {
		Data record `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Data.ID == 0 {
		t.Fatalf("decode record: %v body=%s", err, string(body))
	}
	return envelope.Data
}

func manifestPayload(code, carrier string) map[string]any {
	return map[string]any{
		"code": code, "name": "路由集成测试联单", "description": "valid linked transfer manifest",
		"generatorCode": "WG-001", "carrierCode": carrier, "wasteCode": "HW08-900-249-08", "quantityKg": 680.5,
		"destination": "合规处置中心 A", "facility": "东区危废暂存区", "owner": "operator", "category": "危废转运",
		"riskLevel": "medium", "metricValue": 68, "metricUnit": "score", "effectiveAt": time.Now().UTC().Format(time.RFC3339),
		"evidence": "minio://evidence/tests/manifest.pdf", "relatedCode": strings.ReplaceAll(code, "TM", "REL"),
	}
}

func checkPayload(code, manifest string) map[string]any {
	return map[string]any{
		"code": code, "name": "路由集成测试核验", "description": "linked compliance decision",
		"manifestCode": manifest, "checklist": "产废许可、承运资质、联单数量、处置去向", "decisionBasis": "",
		"facility": "复核中心", "owner": "reviewer", "category": "联单复核", "riskLevel": "medium",
		"metricValue": 92, "metricUnit": "score", "effectiveAt": time.Now().UTC().Format(time.RFC3339),
		"evidence": "minio://evidence/tests/check.pdf", "relatedCode": manifest,
	}
}
