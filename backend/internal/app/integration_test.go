package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbmigrations "bank-darah-backend/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

type integrationEnv struct {
	cfg     Config
	store   *Store
	app     *App
	pool    *pgxpool.Pool
	handler http.Handler
}

func newIntegrationEnv(t *testing.T, fcm *FCMClient) *integrationEnv {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := dbmigrations.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	cfg := Config{
		Port:                  "8080",
		DatabaseURL:           databaseURL,
		JWTSecret:             "test-secret",
		AuthRequired:          true,
		PMIName:               "PMI Test",
		PMILocation:           "PMI Test Location",
		PMILatitude:           -7.7839,
		PMILongitude:          110.3798,
		EligibleRadius:        10,
		TokenTTL:              time.Hour,
		SchedulerInterval:     time.Hour,
		QREncryptionKey:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		FCMProjectID:          "test-project",
		FCMServiceAccountPath: "",
	}

	qrCipher, err := NewQRCipher(cfg.QREncryptionKey)
	if err != nil {
		t.Fatalf("create qr cipher: %v", err)
	}

	store := NewStore(pool, cfg, qrCipher)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := New(cfg, store, logger, fcm)

	return &integrationEnv{
		cfg:     cfg,
		store:   store,
		app:     app,
		pool:    pool,
		handler: app.Handler(),
	}
}

func TestMobileRegisterLoginAndPasswordHashIntegration(t *testing.T) {
	env := newIntegrationEnv(t, nil)
	suffix := uniqueSuffix()

	registerPayload := map[string]interface{}{
		"nik":       "9" + suffix[:15],
		"fullName":  "Integration Donor " + suffix,
		"email":     "donor." + suffix + "@example.test",
		"password":  "secret123",
		"phone":     "+62812" + suffix[:8],
		"bloodType": "O+",
		"gender":    "M",
		"birthDate": "1990-01-01",
		"address":   "Jl. Integration",
		"latitude":  -7.78,
		"longitude": 110.38,
	}

	status, body := performJSONRequest(t, env.handler, http.MethodPost, "/api/v1/mobile/register", registerPayload, "")
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from register, got %d with body %s", status, body)
	}

	response := decodeEnvelopeMap(t, body)
	data := response["data"].(map[string]interface{})
	donorID := data["id"].(string)
	if data["token"] == "" {
		t.Fatal("expected registration response token")
	}

	var passwordHash string
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COALESCE(password_hash, '')
		FROM users
		WHERE id::TEXT = $1
	`, donorID).Scan(&passwordHash); err != nil {
		t.Fatalf("query password hash: %v", err)
	}
	if passwordHash == "" || passwordHash == "secret123" || !strings.HasPrefix(passwordHash, "$2") {
		t.Fatalf("expected bcrypt password hash, got %q", passwordHash)
	}

	status, body = performJSONRequest(t, env.handler, http.MethodPost, "/api/v1/mobile/login", map[string]string{
		"email":    registerPayload["email"].(string),
		"password": "secret123",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d with body %s", status, body)
	}
	data = decodeEnvelopeMap(t, body)["data"].(map[string]interface{})
	if data["token"] == "" {
		t.Fatal("expected login response token")
	}

	status, body = performJSONRequest(t, env.handler, http.MethodPost, "/api/v1/mobile/login", map[string]string{
		"email":    registerPayload["email"].(string),
		"password": "wrong-password",
	}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 from invalid login, got %d with body %s", status, body)
	}
}

func TestDonationHistoryIntegration(t *testing.T) {
	env := newIntegrationEnv(t, nil)
	donor := mustCreateDonor(t, env, "history", true)
	secondDonor := mustCreateDonor(t, env, "empty", true)

	ctx := context.Background()
	if _, err := env.pool.Exec(ctx, `
		INSERT INTO donation_history (donor_id, donation_date, pmi_location, blood_pressure, hemoglobin, weight, is_eligible, status)
		VALUES
		  ($1, DATE '2026-01-01', 'PMI A', '120/80', 13.1, 60, TRUE, 'COMPLETED'),
		  ($1, DATE '2026-03-01', 'PMI B', '121/81', 13.2, 61, TRUE, 'COMPLETED'),
		  ($1, DATE '2026-05-01', 'PMI C', '122/82', 13.3, 62, TRUE, 'COMPLETED')
	`, donor.ID); err != nil {
		t.Fatalf("seed donation history: %v", err)
	}

	token, err := signDonorToken(env.cfg.JWTSecret, donor, time.Hour)
	if err != nil {
		t.Fatalf("sign donor token: %v", err)
	}

	status, body := performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/mobile/history?page=1&limit=2", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from history, got %d with body %s", status, body)
	}
	data := decodeEnvelopeMap(t, body)["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 paginated items, got %d", len(items))
	}
	firstItem := items[0].(map[string]interface{})
	if firstItem["date"] != "2026-05-01" {
		t.Fatalf("expected newest donation first, got %v", firstItem["date"])
	}
	pagination := data["pagination"].(map[string]interface{})
	if int(pagination["total"].(float64)) != 3 || int(pagination["total_pages"].(float64)) != 2 {
		t.Fatalf("unexpected pagination: %#v", pagination)
	}

	status, body = performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/mobile/history?start_date=2026-02-01&end_date=2026-04-30", nil, token)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from filtered history, got %d with body %s", status, body)
	}
	data = decodeEnvelopeMap(t, body)["data"].(map[string]interface{})
	items = data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(items))
	}

	emptyToken, err := signDonorToken(env.cfg.JWTSecret, secondDonor, time.Hour)
	if err != nil {
		t.Fatalf("sign empty donor token: %v", err)
	}
	status, body = performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/mobile/history", nil, emptyToken)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from empty history, got %d with body %s", status, body)
	}
	data = decodeEnvelopeMap(t, body)["data"].(map[string]interface{})
	items = data["items"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %d", len(items))
	}

	status, body = performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/mobile/history", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 without donor token, got %d with body %s", status, body)
	}
}

func TestHospitalDeactivationIntegration(t *testing.T) {
	env := newIntegrationEnv(t, nil)
	adminToken := mustSignAdminToken(t, env.cfg.JWTSecret)
	suffix := uniqueSuffix()

	var hospitalID string
	if err := env.pool.QueryRow(context.Background(), `
		INSERT INTO hospitals (name, address, pic_name, pic_phone, email, is_active)
		VALUES ($1, 'Jl. Rumah Sakit', 'PIC Test', '+6281200000000', $2, TRUE)
		RETURNING id::TEXT
	`, "RS Integration "+suffix, "rs."+suffix+"@example.test").Scan(&hospitalID); err != nil {
		t.Fatalf("insert hospital: %v", err)
	}

	status, body := performJSONRequest(t, env.handler, http.MethodDelete, "/api/v1/hospitals/"+hospitalID, nil, adminToken)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from deactivate hospital, got %d with body %s", status, body)
	}
	if decodeEnvelopeMap(t, body)["message"] != "hospital deactivated" {
		t.Fatalf("unexpected deactivate response: %s", body)
	}

	status, body = performJSONRequest(t, env.handler, http.MethodDelete, "/api/v1/hospitals/"+hospitalID, nil, adminToken)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for already inactive hospital, got %d with body %s", status, body)
	}

	status, body = performJSONRequest(t, env.handler, http.MethodDelete, "/api/v1/hospitals/not-found", nil, adminToken)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404 for missing hospital, got %d with body %s", status, body)
	}

	status, body = performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/hospitals", nil, adminToken)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from hospital list, got %d with body %s", status, body)
	}
	hospitals := decodeEnvelopeMap(t, body)["data"].([]interface{})
	for _, item := range hospitals {
		hospital := item.(map[string]interface{})
		if hospital["id"] == hospitalID {
			t.Fatal("expected inactive hospital to be excluded from default list")
		}
	}

	status, body = performJSONRequest(t, env.handler, http.MethodGet, "/api/v1/hospitals?include_inactive=true", nil, adminToken)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from inclusive hospital list, got %d with body %s", status, body)
	}
	hospitals = decodeEnvelopeMap(t, body)["data"].([]interface{})
	found := false
	for _, item := range hospitals {
		hospital := item.(map[string]interface{})
		if hospital["id"] == hospitalID {
			found = true
			if hospital["isActive"].(bool) {
				t.Fatal("expected deactivated hospital to remain inactive")
			}
		}
	}
	if !found {
		t.Fatal("expected include_inactive list to include deactivated hospital")
	}
}

func TestDonorCheckinRequestsFilterIntegration(t *testing.T) {
	env := newIntegrationEnv(t, nil)
	adminToken := mustSignAdminToken(t, env.cfg.JWTSecret)
	donor := mustCreateDonor(t, env, "checkin-filter", false)
	ctx := context.Background()

	acceptedRequest, err := env.store.CreateEmergencyRequest(ctx, EmergencyCreateRequest{
		HospitalName:   "RS Eligible " + uniqueSuffix(),
		PicName:        "PIC Eligible",
		PicPhone:       "+6281200011111",
		BloodType:      donor.BloodType,
		ProductType:    "WB",
		QuantityNeeded: 2,
		UrgencyLevel:   "URGENT",
	}, "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("create accepted request: %v", err)
	}
	ignoredRequest, err := env.store.CreateEmergencyRequest(ctx, EmergencyCreateRequest{
		HospitalName:   "RS Belum Direspons " + uniqueSuffix(),
		PicName:        "PIC Belum Direspons",
		PicPhone:       "+6281200022222",
		BloodType:      donor.BloodType,
		ProductType:    "PRC",
		QuantityNeeded: 3,
		UrgencyLevel:   "URGENT",
	}, "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("create ignored request: %v", err)
	}

	acceptedBroadcast, err := env.store.BroadcastRequest(ctx, acceptedRequest.ID, nil)
	if err != nil {
		t.Fatalf("broadcast accepted request: %v", err)
	}
	if _, err := env.store.BroadcastRequest(ctx, ignoredRequest.ID, nil); err != nil {
		t.Fatalf("broadcast ignored request: %v", err)
	}
	if _, err := env.store.RespondToBroadcast(ctx, donor.UUID, acceptedBroadcast.BroadcastID, "ACCEPTED"); err != nil {
		t.Fatalf("respond accepted request: %v", err)
	}

	status, body := performJSONRequest(
		t,
		env.handler,
		http.MethodGet,
		"/api/v1/donors/"+url.PathEscape(donor.UUID)+"/checkin-requests",
		nil,
		adminToken,
	)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from donor checkin requests, got %d with body %s", status, body)
	}

	requests := decodeEnvelopeMap(t, body)["data"].([]interface{})
	if len(requests) != 1 {
		t.Fatalf("expected only the donor-accepted active request, got %d: %s", len(requests), body)
	}
	request := requests[0].(map[string]interface{})
	if request["id"] != acceptedRequest.ID {
		t.Fatalf("expected request %s, got %v", acceptedRequest.ID, request["id"])
	}
	if request["responseStatus"] != "ACCEPTED" {
		t.Fatalf("expected responseStatus ACCEPTED, got %v", request["responseStatus"])
	}
}

func TestEligibilityRefreshIntegration(t *testing.T) {
	env := newIntegrationEnv(t, nil)

	donor59 := mustCreateDonor(t, env, "elig59", false)
	donor60 := mustCreateDonor(t, env, "elig60", false)
	donor61 := mustCreateDonor(t, env, "elig61", false)

	if _, err := env.pool.Exec(context.Background(), `
		UPDATE users
		SET last_donation = CURRENT_DATE - INTERVAL '59 days',
		    is_eligible = TRUE
		WHERE id::TEXT = $1
	`, donor59.ID); err != nil {
		t.Fatalf("seed 59-day donor: %v", err)
	}
	if _, err := env.pool.Exec(context.Background(), `
		UPDATE users
		SET last_donation = CURRENT_DATE - INTERVAL '60 days',
		    is_eligible = FALSE
		WHERE id::TEXT = $1
	`, donor60.ID); err != nil {
		t.Fatalf("seed 60-day donor: %v", err)
	}
	if _, err := env.pool.Exec(context.Background(), `
		UPDATE users
		SET last_donation = CURRENT_DATE - INTERVAL '61 days',
		    is_eligible = FALSE
		WHERE id::TEXT = $1
	`, donor61.ID); err != nil {
		t.Fatalf("seed 61-day donor: %v", err)
	}

	if _, err := env.store.RefreshEligibility(context.Background()); err != nil {
		t.Fatalf("refresh eligibility: %v", err)
	}

	assertEligibility := func(donorID string, expected bool) {
		t.Helper()
		var actual bool
		if err := env.pool.QueryRow(context.Background(), `
			SELECT is_eligible
			FROM users
			WHERE id::TEXT = $1
		`, donorID).Scan(&actual); err != nil {
			t.Fatalf("query eligibility: %v", err)
		}
		if actual != expected {
			t.Fatalf("expected donor %s eligibility %v, got %v", donorID, expected, actual)
		}
	}

	assertEligibility(donor59.ID, false)
	assertEligibility(donor60.ID, true)
	assertEligibility(donor61.ID, true)
}

func TestDonationThankYouNotificationIntegration(t *testing.T) {
	var sentCount int32
	successFCM := newFakeFCMClient(false, &sentCount)
	env := newIntegrationEnv(t, successFCM)

	donor := mustCreateDonor(t, env, "notify", false)
	if _, err := env.pool.Exec(context.Background(), `
		UPDATE users
		SET device_token = 'device-token-success'
		WHERE id::TEXT = $1
	`, donor.ID); err != nil {
		t.Fatalf("set donor device token: %v", err)
	}

	var donationID string
	if err := env.pool.QueryRow(context.Background(), `
		INSERT INTO donation_history (donor_id, donation_date, pmi_location, blood_pressure, hemoglobin, weight, is_eligible, status)
		VALUES ($1, CURRENT_DATE, 'PMI Test', '120/80', 13.5, 60, TRUE, 'COMPLETED')
		RETURNING id::TEXT
	`, donor.ID).Scan(&donationID); err != nil {
		t.Fatalf("insert donation history: %v", err)
	}

	if err := env.store.SendDonationThankYouNotification(context.Background(), donationID, donor.ID, successFCM); err != nil {
		t.Fatalf("send thank-you notification: %v", err)
	}
	if err := env.store.SendDonationThankYouNotification(context.Background(), donationID, donor.ID, successFCM); err != nil {
		t.Fatalf("send duplicate thank-you notification: %v", err)
	}
	if got := atomic.LoadInt32(&sentCount); got != 1 {
		t.Fatalf("expected one successful send, got %d", got)
	}

	var logCount int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM notification_logs
		WHERE donation_id::TEXT = $1 AND type = 'DONATION_THANK_YOU'
	`, donationID).Scan(&logCount); err != nil {
		t.Fatalf("count notification logs: %v", err)
	}
	if logCount != 1 {
		t.Fatalf("expected one notification log, got %d", logCount)
	}

	missingTokenDonor := mustCreateDonor(t, env, "notoken", false)
	var missingDonationID string
	if err := env.pool.QueryRow(context.Background(), `
		INSERT INTO donation_history (donor_id, donation_date, pmi_location, blood_pressure, hemoglobin, weight, is_eligible, status)
		VALUES ($1, CURRENT_DATE, 'PMI Test', '120/80', 13.5, 60, TRUE, 'COMPLETED')
		RETURNING id::TEXT
	`, missingTokenDonor.ID).Scan(&missingDonationID); err != nil {
		t.Fatalf("insert donation history without device token: %v", err)
	}
	if err := env.store.SendDonationThankYouNotification(context.Background(), missingDonationID, missingTokenDonor.ID, successFCM); err != nil {
		t.Fatalf("send thank-you without device token: %v", err)
	}

	var missingLogCount int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM notification_logs
		WHERE donation_id::TEXT = $1
	`, missingDonationID).Scan(&missingLogCount); err != nil {
		t.Fatalf("count missing-token notification logs: %v", err)
	}
	if missingLogCount != 0 {
		t.Fatalf("expected no notification log for missing token, got %d", missingLogCount)
	}

	var failedCount int32
	failingEnv := newIntegrationEnv(t, newFakeFCMClient(true, &failedCount))
	failingDonor := mustCreateDonor(t, failingEnv, "notifyfail", false)
	if _, err := failingEnv.pool.Exec(context.Background(), `
		UPDATE users
		SET device_token = 'device-token-fail',
		    is_eligible = TRUE,
		    last_donation = CURRENT_DATE - INTERVAL '61 days'
		WHERE id::TEXT = $1
	`, failingDonor.ID); err != nil {
		t.Fatalf("prepare failing donor: %v", err)
	}

	adminToken := mustSignAdminToken(t, failingEnv.cfg.JWTSecret)
	status, body := performJSONRequest(t, failingEnv.handler, http.MethodPost, "/api/v1/donations/checkin", map[string]interface{}{
		"donorUuid":  failingDonor.UUID,
		"systolic":   120,
		"diastolic":  80,
		"hemoglobin": 13.5,
		"weight":     60,
	}, adminToken)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from checkin with failing FCM, got %d with body %s", status, body)
	}

	var completedCount int
	if err := failingEnv.pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM donation_history
		WHERE donor_id::TEXT = $1 AND status = 'COMPLETED'
	`, failingDonor.ID).Scan(&completedCount); err != nil {
		t.Fatalf("count completed donations: %v", err)
	}
	if completedCount == 0 {
		t.Fatal("expected completed donation to persist despite FCM failure")
	}
	if got := atomic.LoadInt32(&failedCount); got == 0 {
		t.Fatal("expected failing FCM client to be called")
	}
}

func mustCreateDonor(t *testing.T, env *integrationEnv, label string, withPassword bool) Donor {
	t.Helper()

	suffix := uniqueSuffix()
	request := DonorCreateRequest{
		NIK:       "8" + suffix[:15],
		FullName:  "Test Donor " + label + " " + suffix,
		Email:     fmt.Sprintf("%s.%s@example.test", label, suffix),
		Phone:     "+62813" + suffix[:8],
		BloodType: "O+",
		Gender:    "M",
		BirthDate: "1990-01-01",
		Address:   "Jl. Test",
		Latitude:  -7.78,
		Longitude: 110.38,
	}
	if withPassword {
		request.Password = "secret123"
	}

	donor, err := env.store.CreateDonor(context.Background(), request)
	if err != nil {
		t.Fatalf("create donor: %v", err)
	}
	return donor
}

func mustSignAdminToken(t *testing.T, secret string) string {
	t.Helper()

	token, err := signToken(secret, AdminUser{
		ID:       "11111111-1111-1111-1111-111111111111",
		Username: "operator",
		Role:     "OPERATOR",
	}, time.Hour)
	if err != nil {
		t.Fatalf("sign admin token: %v", err)
	}
	return token
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path string, body interface{}, bearerToken string) (int, string) {
	t.Helper()

	var requestBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		requestBody = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, requestBody)
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func decodeEnvelopeMap(t *testing.T, body string) map[string]interface{} {
	t.Helper()

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return decoded
}

func uniqueSuffix() string {
	return fmt.Sprintf("%015d", time.Now().UnixNano()%1_000_000_000_000_000)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newFakeFCMClient(shouldFail bool, counter *int32) *FCMClient {
	return &FCMClient{
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		projectID:   "test-project",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(counter, 1)
				statusCode := http.StatusOK
				body := `{"name":"projects/test/messages/1"}`
				if shouldFail {
					statusCode = http.StatusInternalServerError
					body = `{"error":"fcm failed"}`
				}
				return &http.Response{
					StatusCode: statusCode,
					Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
}
