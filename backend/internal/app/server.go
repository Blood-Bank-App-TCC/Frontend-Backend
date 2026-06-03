package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type App struct {
	cfg     Config
	store   *Store
	logger  *slog.Logger
	fcm     *FCMClient
	limiter *broadcastLimiter
	mobile  *broadcastLimiter
}

func New(cfg Config, store *Store, logger *slog.Logger, fcm *FCMClient) *App {
	return &App{
		cfg:     cfg,
		store:   store,
		logger:  logger,
		fcm:     fcm,
		limiter: newBroadcastLimiter(10, time.Minute),
		mobile:  newBroadcastLimiter(5, time.Minute),
	}
}

func (a *App) Handler() http.Handler {
	return http.HandlerFunc(a.serveHTTP)
}

func (a *App) serveHTTP(w http.ResponseWriter, r *http.Request) {
	a.setCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/api/v1") {
		fail(w, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan.")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	if path == "" {
		path = "/"
	}

	if r.Method == http.MethodGet && path == "/health" {
		a.handleHealth(w, r)
		return
	}
	if r.Method == http.MethodPost && path == "/auth/admin/login" {
		a.handleLogin(w, r)
		return
	}
	if strings.HasPrefix(path, "/mobile") {
		a.routeMobile(w, r, path)
		return
	}

	if a.cfg.AuthRequired {
		admin, err := parseToken(a.cfg.JWTSecret, bearerToken(r))
		if err != nil {
			fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid atau sudah kedaluwarsa.")
			return
		}
		r = r.WithContext(withAdmin(r.Context(), admin))
	}

	a.routeProtected(w, r, path)
}

func (a *App) routeMobile(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case r.Method == http.MethodPost && path == "/mobile/respond":
		a.handleMobileRespond(w, r)
	case r.Method == http.MethodGet && path == "/mobile/broadcast/active":
		a.handleMobileActiveBroadcast(w, r)
	case r.Method == http.MethodPost && path == "/mobile/login":
		a.handleMobileLogin(w, r)
	case r.Method == http.MethodPost && path == "/mobile/register":
		a.handleMobileRegister(w, r)
	case r.Method == http.MethodGet && path == "/mobile/donor":
		a.handleMobileGetDonor(w, r)
	case r.Method == http.MethodGet && path == "/mobile/stock":
		a.handleMobileListStock(w, r)
	case r.Method == http.MethodPut && path == "/mobile/donor":
		a.handleMobileUpdateDonor(w, r)
	default:
		fail(w, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan.")
	}
}

func (a *App) routeProtected(w http.ResponseWriter, r *http.Request, path string) {
	segments := splitPath(path)

	switch {
	case r.Method == http.MethodGet && path == "/auth/me":
		a.handleMe(w, r)

	case r.Method == http.MethodGet && path == "/stock":
		a.handleListStock(w, r)
	case r.Method == http.MethodPut && len(segments) == 3 && segments[0] == "stock":
		a.handleUpdateStock(w, r, segments[1], segments[2])

	case r.Method == http.MethodGet && path == "/emergency/requests":
		a.handleListRequests(w, r)
	case r.Method == http.MethodPost && path == "/emergency/requests":
		a.handleCreateRequest(w, r)
	case r.Method == http.MethodGet && len(segments) == 4 && segments[0] == "emergency" && segments[1] == "requests" && segments[3] == "eligible-donors":
		a.handleEligibleDonors(w, r, segments[2])
	case r.Method == http.MethodPost && len(segments) == 4 && segments[0] == "emergency" && segments[1] == "requests" && segments[3] == "broadcast":
		a.handleBroadcast(w, r, segments[2])
	case r.Method == http.MethodGet && len(segments) == 4 && segments[0] == "emergency" && segments[1] == "requests" && segments[3] == "live-responses":
		a.handleLiveResponses(w, r, segments[2])
	case r.Method == http.MethodPut && len(segments) == 4 && segments[0] == "emergency" && segments[1] == "requests" && segments[3] == "close":
		a.handleCloseRequest(w, r, segments[2])

	case r.Method == http.MethodGet && path == "/donors":
		a.handleListDonors(w, r)
	case r.Method == http.MethodPost && path == "/donors":
		a.handleCreateDonor(w, r)
	case r.Method == http.MethodPut && len(segments) == 3 && segments[0] == "donors" && segments[2] == "status":
		a.handleUpdateDonorStatus(w, r, segments[1])
	case r.Method == http.MethodGet && len(segments) == 2 && segments[0] == "donors":
		a.handleGetDonor(w, r, segments[1])
	case r.Method == http.MethodPut && len(segments) == 2 && segments[0] == "donors":
		a.handleUpdateDonor(w, r, segments[1])

	case r.Method == http.MethodPost && path == "/donations/checkin":
		a.handleCheckin(w, r)

	case r.Method == http.MethodGet && path == "/hospitals":
		a.handleListHospitals(w, r)
	case r.Method == http.MethodPost && path == "/hospitals":
		a.handleCreateHospital(w, r)
	case r.Method == http.MethodPut && len(segments) == 2 && segments[0] == "hospitals":
		a.handleUpdateHospital(w, r, segments[1])

	default:
		fail(w, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan.")
	}
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := a.store.pool.Ping(ctx); err != nil {
		dbStatus = "error"
	}

	ok(w, map[string]interface{}{
		"status":    "ok",
		"backend":   "ok",
		"database":  dbStatus,
		"timestamp": time.Now().UTC(),
	}, http.StatusOK)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input LoginRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body login tidak valid.")
		return
	}

	admin, passwordHash, err := a.store.FindAdminByUsername(r.Context(), input.Username)
	if err != nil || !verifyPassword(passwordHash, input.Password) {
		fail(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Username atau password salah.")
		return
	}

	token, err := signToken(a.cfg.JWTSecret, admin, a.cfg.TokenTTL)
	if err != nil {
		a.serverError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(a.cfg.TokenTTL.Seconds()),
	})
	ok(w, map[string]interface{}{"token": token, "user": admin}, http.StatusOK)
}

func (a *App) handleMobileRespond(w http.ResponseWriter, r *http.Request) {
	var input MobileRespondRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body respons donor tidak valid.")
		return
	}
	if strings.TrimSpace(input.QRToken) == "" || strings.TrimSpace(input.BroadcastID) == "" || strings.TrimSpace(input.Status) == "" {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "qr_token, broadcast_id, dan status wajib diisi.")
		return
	}
	if !a.mobile.Allow(strings.TrimSpace(input.QRToken)) {
		fail(w, http.StatusTooManyRequests, "RATE_LIMITED", "Respons terlalu sering. Coba lagi sebentar.")
		return
	}

	response, err := a.store.RespondToBroadcast(r.Context(), input.QRToken, input.BroadcastID, input.Status)
	a.respond(w, response, err, http.StatusOK)
}

func (a *App) handleMobileActiveBroadcast(w http.ResponseWriter, r *http.Request) {
	qrToken := strings.TrimSpace(r.URL.Query().Get("qr_token"))
	if qrToken == "" {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "qr_token wajib diisi.")
		return
	}

	active, found, err := a.store.ActiveBroadcastForDonor(r.Context(), qrToken)
	if err != nil {
		a.respond(w, nil, err, http.StatusOK)
		return
	}
	if !found {
		ok(w, map[string]interface{}{"broadcast": nil, "response": nil}, http.StatusOK)
		return
	}
	ok(w, active, http.StatusOK)
}

func (a *App) handleMobileLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		LoginKey string `json:"loginKey"`
	}
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body login tidak valid.")
		return
	}

	key := input.LoginKey
	if key == "" {
		key = input.Email
	}

	donor, err := a.store.FindDonorByLogin(r.Context(), key)
	a.respond(w, donor, err, http.StatusOK)
}

func (a *App) handleMobileRegister(w http.ResponseWriter, r *http.Request) {
	var input DonorCreateRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body registrasi tidak valid.")
		return
	}
	donor, err := a.store.CreateDonor(r.Context(), input)
	a.respond(w, donor, err, http.StatusCreated)
}

func (a *App) handleMobileGetDonor(w http.ResponseWriter, r *http.Request) {
	qrToken := strings.TrimSpace(r.URL.Query().Get("qr_token"))
	if qrToken == "" {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "qr_token wajib diisi.")
		return
	}

	donor, err := a.store.activeDonorByQRToken(r.Context(), qrToken)
	if err != nil {
		a.respond(w, nil, err, http.StatusOK)
		return
	}

	// Fetch history as well
	history, err := a.store.DonationHistory(r.Context(), donor.ID)
	if err == nil {
		donor.DonationHistory = history
	}

	ok(w, donor, http.StatusOK)
}

func (a *App) handleMobileListStock(w http.ResponseWriter, r *http.Request) {
	stock, err := a.store.ListStock(r.Context())
	a.respond(w, stock, err, http.StatusOK)
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	ok(w, adminFromContext(r.Context()), http.StatusOK)
}

func (a *App) handleListStock(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	stock, err := a.store.ListStock(r.Context())
	a.respond(w, stock, err, http.StatusOK)
}

func (a *App) handleUpdateStock(w http.ResponseWriter, r *http.Request, bloodType, productType string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input StockUpdateRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body update stok tidak valid.")
		return
	}
	admin := adminFromContext(r.Context())
	item, err := a.store.UpdateStock(r.Context(), bloodType, productType, input, admin.ID)
	a.respond(w, item, err, http.StatusOK)
}

func (a *App) handleListRequests(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	requests, err := a.store.ListRequests(r.Context())
	a.respond(w, requests, err, http.StatusOK)
}

func (a *App) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input EmergencyCreateRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body permintaan darurat tidak valid.")
		return
	}
	admin := adminFromContext(r.Context())
	request, err := a.store.CreateEmergencyRequest(r.Context(), input, admin.ID)
	a.respond(w, request, err, http.StatusCreated)
}

func (a *App) handleEligibleDonors(w http.ResponseWriter, r *http.Request, requestID string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	request, err := a.store.GetRequest(r.Context(), requestID)
	if err != nil {
		a.respond(w, nil, err, http.StatusOK)
		return
	}
	donors, err := a.store.EligibleDonorsForRequest(r.Context(), requestID)
	if err != nil {
		a.respond(w, nil, err, http.StatusOK)
		return
	}
	request.EligibleCount = len(donors)
	ok(w, map[string]interface{}{
		"request": request,
		"donors":  donors,
		"count":   len(donors),
	}, http.StatusOK)
}

func (a *App) handleBroadcast(w http.ResponseWriter, r *http.Request, requestID string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	admin := adminFromContext(r.Context())
	limitKey := admin.ID
	if limitKey == "" {
		limitKey = r.RemoteAddr
	}
	if !a.limiter.Allow(limitKey) {
		fail(w, http.StatusTooManyRequests, "RATE_LIMITED", "Broadcast terlalu sering. Batas maksimum 10 kali per menit.")
		return
	}

	result, err := a.store.BroadcastRequest(r.Context(), requestID, a.fcm)
	a.respond(w, result, err, http.StatusOK)
}

func (a *App) handleLiveResponses(w http.ResponseWriter, r *http.Request, requestID string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	responses, err := a.store.ListLiveResponses(r.Context(), requestID)
	a.respond(w, responses, err, http.StatusOK)
}

func (a *App) handleListDonors(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	donors, err := a.store.ListDonors(r.Context(), r.URL.Query().Get("search"))
	a.respond(w, donors, err, http.StatusOK)
}

func (a *App) handleGetDonor(w http.ResponseWriter, r *http.Request, key string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	donor, err := a.store.GetDonor(r.Context(), key)
	a.respond(w, donor, err, http.StatusOK)
}

func (a *App) handleCreateDonor(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input DonorCreateRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body pendonor tidak valid.")
		return
	}
	donor, err := a.store.CreateDonor(r.Context(), input)
	a.respond(w, donor, err, http.StatusCreated)
}

func (a *App) handleUpdateDonor(w http.ResponseWriter, r *http.Request, id string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input DonorUpdateRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body update pendonor tidak valid.")
		return
	}
	donor, err := a.store.UpdateDonor(r.Context(), id, input)
	a.respond(w, donor, err, http.StatusOK)
}

func (a *App) handleUpdateDonorStatus(w http.ResponseWriter, r *http.Request, id string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input DonorStatusRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body status pendonor tidak valid.")
		return
	}
	donor, err := a.store.UpdateDonorStatus(r.Context(), id, input.IsActive)
	a.respond(w, donor, err, http.StatusOK)
}

func (a *App) handleCheckin(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input DonationCheckinRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body check-in tidak valid.")
		return
	}
	admin := adminFromContext(r.Context())
	result, err := a.store.CheckinDonation(r.Context(), input, admin.ID)
	if err == nil && result.IsEligible {
		donor, donorErr := a.store.GetDonor(r.Context(), input.DonorUUID)
		if donorErr == nil {
			_, stockErr := a.store.UpdateStock(r.Context(), donor.BloodType, "WB", StockUpdateRequest{
				Mode:      "add",
				Quantity:  1,
				Reference: result.DonationID,
				Notes:     "auto update dari checkin donor",
			}, admin.ID)
			if stockErr != nil {
				a.logger.Error("auto stock update after checkin failed", "error", stockErr, "donation_id", result.DonationID)
			}
		}
	}
	a.respond(w, result, err, http.StatusOK)
}

func (a *App) handleListHospitals(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	hospitals, err := a.store.ListHospitals(r.Context())
	a.respond(w, hospitals, err, http.StatusOK)
}

func (a *App) handleCreateHospital(w http.ResponseWriter, r *http.Request) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input HospitalRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body rumah sakit tidak valid.")
		return
	}
	hospital, err := a.store.CreateHospital(r.Context(), input)
	a.respond(w, hospital, err, http.StatusCreated)
}

func (a *App) handleUpdateHospital(w http.ResponseWriter, r *http.Request, id string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	var input HospitalRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body update rumah sakit tidak valid.")
		return
	}
	hospital, err := a.store.UpdateHospital(r.Context(), id, input)
	a.respond(w, hospital, err, http.StatusOK)
}

func (a *App) handleMobileUpdateDonor(w http.ResponseWriter, r *http.Request) {
	qrToken := r.URL.Query().Get("qr_token")
	if strings.TrimSpace(qrToken) == "" {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "qr_token wajib diisi.")
		return
	}
	var input UpdateDonorRequest
	if err := decodeJSON(r, &input); err != nil {
		fail(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}
	donor, err := a.store.UpdateDonorByQRToken(r.Context(), qrToken, input)
	if err != nil {
		a.serverError(w, err)
		return
	}
	ok(w, donor, http.StatusOK)
}

func (a *App) handleCloseRequest(w http.ResponseWriter, r *http.Request, requestID string) {
	if !a.requireRole(w, r, "SUPER_ADMIN", "OPERATOR") {
		return
	}
	req, err := a.store.CloseRequest(r.Context(), requestID)
	if err != nil {
		a.serverError(w, err)
		return
	}
	ok(w, req, http.StatusOK)
}

func (a *App) requireRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	admin := adminFromContext(r.Context())
	for _, role := range roles {
		if strings.EqualFold(admin.Role, role) {
			return true
		}
	}
	fail(w, http.StatusForbidden, "FORBIDDEN", "Akses ditolak. Hak akses tidak cukup.")
	return false
}

func (a *App) respond(w http.ResponseWriter, data interface{}, err error, status int) {
	if err == nil {
		ok(w, data, status)
		return
	}

	var appErr appError
	if errors.As(err, &appErr) {
		fail(w, appErr.status, appErr.code, appErr.message)
		return
	}
	a.serverError(w, err)
}

func (a *App) serverError(w http.ResponseWriter, err error) {
	a.logger.Error("request failed", "error", err)
	fail(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Terjadi kesalahan pada server.")
}

func (a *App) setCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	raw := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		value, err := url.PathUnescape(segment)
		if err != nil {
			value = segment
		}
		segments = append(segments, value)
	}
	return segments
}

type broadcastLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	hits   map[string][]time.Time
}

func newBroadcastLimiter(limit int, window time.Duration) *broadcastLimiter {
	return &broadcastLimiter{limit: limit, window: window, hits: make(map[string][]time.Time)}
}

func (l *broadcastLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
	var kept []time.Time
	for _, hit := range l.hits[key] {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	kept = append(kept, now)
	l.hits[key] = kept
	return true
}
