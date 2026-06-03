package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool     *pgxpool.Pool
	cfg      Config
	qrCipher *QRCipher
}

func NewStore(pool *pgxpool.Pool, cfg Config, qrCipher *QRCipher) *Store {
	return &Store{pool: pool, cfg: cfg, qrCipher: qrCipher}
}

func (s *Store) RefreshEligibility(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users
		SET is_eligible = TRUE, updated_at = NOW()
		WHERE is_active = TRUE
		  AND last_donation IS NOT NULL
		  AND next_eligible <= CURRENT_DATE
		  AND is_eligible = FALSE
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) FindAdminByUsername(ctx context.Context, username string) (AdminUser, string, error) {
	var admin AdminUser
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT id::TEXT, username, password_hash, full_name, role
		FROM admin_users
		WHERE username = $1 AND is_active = TRUE
	`, username).Scan(&admin.ID, &admin.Username, &passwordHash, &admin.FullName, &admin.Role)
	return admin, passwordHash, err
}

func (s *Store) ListStock(ctx context.Context) ([]BloodStock, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, blood_type, product_type, quantity, safe_threshold, critical_threshold, updated_at
		FROM blood_stock
		ORDER BY
			array_position(ARRAY['A+','A-','B+','B-','O+','O-','AB+','AB-'], blood_type),
			array_position(ARRAY['WB','PRC','FFP','THROMBOCYTE'], product_type)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stock []BloodStock
	for rows.Next() {
		item, err := scanStock(rows)
		if err != nil {
			return nil, err
		}
		stock = append(stock, item)
	}
	return stock, rows.Err()
}

func (s *Store) UpdateStock(ctx context.Context, bloodType, productType string, input StockUpdateRequest, adminID string) (BloodStock, error) {
	if input.Quantity < 0 {
		return BloodStock{}, errBadRequest("quantity tidak boleh negatif")
	}
	if input.Mode == "" {
		input.Mode = "add"
	}
	if input.Mode != "add" && input.Mode != "subtract" && input.Mode != "set" {
		return BloodStock{}, errBadRequest("mode stok tidak valid")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BloodStock{}, err
	}
	defer tx.Rollback(ctx)

	var item BloodStock
	var updateSQL string
	switch input.Mode {
	case "set":
		updateSQL = `
			UPDATE blood_stock SET quantity = $3, updated_at = NOW()
			WHERE blood_type = $1 AND product_type = $2
			RETURNING id::TEXT, blood_type, product_type, quantity, safe_threshold, critical_threshold, updated_at
		`
	case "subtract":
		updateSQL = `
			UPDATE blood_stock SET quantity = GREATEST(0, quantity - $3), updated_at = NOW()
			WHERE blood_type = $1 AND product_type = $2
			RETURNING id::TEXT, blood_type, product_type, quantity, safe_threshold, critical_threshold, updated_at
		`
	default:
		updateSQL = `
			UPDATE blood_stock SET quantity = quantity + $3, updated_at = NOW()
			WHERE blood_type = $1 AND product_type = $2
			RETURNING id::TEXT, blood_type, product_type, quantity, safe_threshold, critical_threshold, updated_at
		`
	}

	if err := tx.QueryRow(ctx, updateSQL, bloodType, productType, input.Quantity).Scan(
		&item.ID,
		&item.BloodType,
		&item.ProductType,
		&item.Quantity,
		&item.SafeThreshold,
		&item.CriticalThreshold,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BloodStock{}, errNotFound("stok tidak ditemukan")
		}
		return BloodStock{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_transactions (blood_type, product_type, quantity, mode, reference, notes, admin_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, bloodType, productType, input.Quantity, input.Mode, nullString(input.Reference), nullString(input.Notes), nullString(adminID))
	if err != nil {
		return BloodStock{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return BloodStock{}, err
	}
	return item, nil
}

func (s *Store) ListHospitals(ctx context.Context) ([]Hospital, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, name, COALESCE(address, ''), COALESCE(latitude::DOUBLE PRECISION, 0),
		       COALESCE(longitude::DOUBLE PRECISION, 0), COALESCE(pic_name, ''),
		       COALESCE(pic_phone, ''), COALESCE(email, ''), is_active
		FROM hospitals
		ORDER BY created_at DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hospitals []Hospital
	for rows.Next() {
		hospital, err := scanHospital(rows)
		if err != nil {
			return nil, err
		}
		hospitals = append(hospitals, hospital)
	}
	return hospitals, rows.Err()
}

func (s *Store) CreateHospital(ctx context.Context, input HospitalRequest) (Hospital, error) {
	input = normalizeHospital(input, s.cfg)
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO hospitals (name, address, latitude, longitude, pic_name, pic_phone, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::TEXT
	`, input.Name, input.Address, input.Latitude, input.Longitude, input.PicName, input.PicPhone, nullString(input.Email)).Scan(&id)
	if err != nil {
		return Hospital{}, err
	}
	return s.GetHospital(ctx, id)
}

func (s *Store) UpdateHospital(ctx context.Context, id string, input HospitalRequest) (Hospital, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE hospitals
		SET name = COALESCE(NULLIF($2, ''), name),
		    address = COALESCE($3, address),
		    latitude = CASE WHEN $4::DOUBLE PRECISION <> 0 THEN $4 ELSE latitude END,
		    longitude = CASE WHEN $5::DOUBLE PRECISION <> 0 THEN $5 ELSE longitude END,
		    pic_name = COALESCE(NULLIF($6, ''), pic_name),
		    pic_phone = COALESCE(NULLIF($7, ''), pic_phone),
		    email = $8
		WHERE id::TEXT = $1
	`, id, input.Name, input.Address, input.Latitude, input.Longitude, input.PicName, input.PicPhone, nullString(input.Email))
	if err != nil {
		return Hospital{}, err
	}
	return s.GetHospital(ctx, id)
}

func (s *Store) GetHospital(ctx context.Context, id string) (Hospital, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::TEXT, name, COALESCE(address, ''), COALESCE(latitude::DOUBLE PRECISION, 0),
		       COALESCE(longitude::DOUBLE PRECISION, 0), COALESCE(pic_name, ''),
		       COALESCE(pic_phone, ''), COALESCE(email, ''), is_active
		FROM hospitals
		WHERE id::TEXT = $1
	`, id)
	hospital, err := scanHospital(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hospital{}, errNotFound("rumah sakit tidak ditemukan")
	}
	return hospital, err
}

func (s *Store) ListRequests(ctx context.Context) ([]EmergencyRequest, error) {
	rows, err := s.pool.Query(ctx, requestSelectSQL()+`
		ORDER BY br.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []EmergencyRequest
	for rows.Next() {
		request, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (s *Store) GetRequest(ctx context.Context, id string) (EmergencyRequest, error) {
	row := s.pool.QueryRow(ctx, requestSelectSQL()+` WHERE br.id::TEXT = $1`, id)
	request, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmergencyRequest{}, errNotFound("permintaan tidak ditemukan")
	}
	return request, err
}

func (s *Store) CreateEmergencyRequest(ctx context.Context, input EmergencyCreateRequest, adminID string) (EmergencyRequest, error) {
	if input.HospitalName == "" || input.PicName == "" || input.PicPhone == "" || input.BloodType == "" {
		return EmergencyRequest{}, errBadRequest("data permintaan darurat belum lengkap")
	}
	if input.ProductType == "" {
		input.ProductType = "PRC"
	}
	if input.UrgencyLevel == "" {
		input.UrgencyLevel = "NORMAL"
	}
	if input.QuantityNeeded <= 0 {
		return EmergencyRequest{}, errBadRequest("jumlah kantong harus lebih dari 0")
	}

	hospitalID, err := s.findOrCreateHospital(ctx, input)
	if err != nil {
		return EmergencyRequest{}, err
	}

	var id string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO blood_requests (hospital_id, requested_by, blood_type, product_type, quantity_needed, urgency_level, notes, admin_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::TEXT
	`, hospitalID, input.PicName, input.BloodType, input.ProductType, input.QuantityNeeded, input.UrgencyLevel, input.Notes, nullString(adminID)).Scan(&id)
	if err != nil {
		return EmergencyRequest{}, err
	}

	donors, err := s.EligibleDonorsForRequest(ctx, id)
	if err != nil {
		return EmergencyRequest{}, err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE blood_requests SET eligible_count = $2 WHERE id::TEXT = $1`, id, len(donors))

	return s.GetRequest(ctx, id)
}

func (s *Store) EligibleDonorsForRequest(ctx context.Context, requestID string) ([]Donor, error) {
	request, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, donorSelectSQL()+`
		WHERE u.blood_type = $3
		  AND u.is_active = TRUE
		  AND u.is_eligible = TRUE
		  AND (u.next_eligible IS NULL OR u.next_eligible <= CURRENT_DATE)
		  AND ST_DWithin(
		    u.location,
		    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
		    $4
		  )
		ORDER BY distance_km ASC
	`, s.cfg.PMILongitude, s.cfg.PMILatitude, request.BloodType, s.cfg.EligibleRadius*1000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	donors, err := scanDonors(rows)
	if err != nil {
		return nil, err
	}
	_, _ = s.pool.Exec(ctx, `UPDATE blood_requests SET eligible_count = $2 WHERE id::TEXT = $1`, requestID, len(donors))
	return donors, nil
}

func (s *Store) BroadcastRequest(ctx context.Context, requestID string, fcm *FCMClient) (BroadcastResult, error) {
	request, err := s.GetRequest(ctx, requestID)
	if err != nil {
		return BroadcastResult{}, err
	}
	donors, err := s.EligibleDonorsForRequest(ctx, requestID)
	if err != nil {
		return BroadcastResult{}, err
	}

	broadcastID := request.BroadcastID
	if broadcastID == "" {
		broadcastID = "broadcast-" + request.ID
	}
	queuedAt := time.Now().UTC()
	messageTitle := fmt.Sprintf("%s: Butuh Donor %s", request.UrgencyLevel, request.BloodType)
	messageBody := fmt.Sprintf("%s membutuhkan %d kantong %s untuk %s.", s.cfg.PMIName, request.QuantityNeeded, request.BloodType, request.HospitalName)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BroadcastResult{}, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE blood_requests
		SET status = 'ACTIVE', broadcast_id = $2, broadcast_sent_at = NOW(), eligible_count = $3
		WHERE id::TEXT = $1
	`, requestID, broadcastID, len(donors))
	if err != nil {
		return BroadcastResult{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO emergency_broadcasts (broadcast_id, request_id, blood_type, urgency_level, message_title, message_body, eligible_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (broadcast_id) DO UPDATE
		SET eligible_count = EXCLUDED.eligible_count,
		    message_title = EXCLUDED.message_title,
		    message_body = EXCLUDED.message_body,
		    status = 'ACTIVE',
		    closed_at = NULL
	`, broadcastID, requestID, request.BloodType, request.UrgencyLevel, messageTitle, messageBody, len(donors))
	if err != nil {
		return BroadcastResult{}, err
	}

	for _, donor := range donors {
		_, err = tx.Exec(ctx, `
			INSERT INTO live_responses (broadcast_id, request_id, donor_id, donor_name, donor_blood, distance_km, status)
			VALUES ($1, $2, $3, $4, $5, $6, 'NO_RESPONSE')
			ON CONFLICT (broadcast_id, donor_id) DO UPDATE
			SET donor_name = EXCLUDED.donor_name,
			    donor_blood = EXCLUDED.donor_blood,
			    distance_km = EXCLUDED.distance_km
		`, broadcastID, requestID, donor.ID, donor.FullName, donor.BloodType, donor.DistanceKm)
		if err != nil {
			return BroadcastResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return BroadcastResult{}, err
	}

	result := BroadcastResult{BroadcastID: broadcastID, RecipientCount: len(donors), QueuedAt: queuedAt}
	s.sendBroadcastNotifications(ctx, fcm, request, donors, broadcastID, messageTitle, messageBody)
	return result, nil
}

type broadcastDeviceToken struct {
	DonorID string
	Token   string
}

func (s *Store) sendBroadcastNotifications(ctx context.Context, fcm *FCMClient, request EmergencyRequest, donors []Donor, broadcastID, title, body string) {
	if fcm == nil || len(donors) == 0 {
		return
	}

	recipients, err := s.deviceTokensForDonors(ctx, donors)
	if err != nil {
		slog.Default().Error("fetch fcm device tokens", "request_id", request.ID, "broadcast_id", broadcastID, "error", err)
		return
	}
	if len(recipients) == 0 {
		return
	}

	tokens := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		tokens = append(tokens, recipient.Token)
	}

	data := map[string]string{
		"type":            "emergency_broadcast",
		"broadcast_id":    broadcastID,
		"request_id":      request.ID,
		"blood_type":      request.BloodType,
		"product_type":    request.ProductType,
		"urgency_level":   request.UrgencyLevel,
		"hospital_name":   request.HospitalName,
		"quantity_needed": strconv.Itoa(request.QuantityNeeded),
	}

	errs := fcm.SendBatch(ctx, tokens, title, body, data)
	for i, err := range errs {
		if err == nil {
			continue
		}
		slog.Default().Error(
			"send fcm notification",
			"request_id", request.ID,
			"broadcast_id", broadcastID,
			"donor_id", recipients[i].DonorID,
			"token", recipients[i].Token,
			"error", err,
		)
	}
}

func (s *Store) deviceTokensForDonors(ctx context.Context, donors []Donor) ([]broadcastDeviceToken, error) {
	donorIDs := make([]pgtype.UUID, 0, len(donors))
	for _, donor := range donors {
		var donorID pgtype.UUID
		if err := donorID.Scan(donor.ID); err != nil {
			return nil, fmt.Errorf("parse donor id %q: %w", donor.ID, err)
		}
		donorIDs = append(donorIDs, donorID)
	}
	if len(donorIDs) == 0 {
		return nil, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, device_token
		FROM users
		WHERE id = ANY($1) AND device_token IS NOT NULL
	`, donorIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []broadcastDeviceToken
	for rows.Next() {
		var recipient broadcastDeviceToken
		if err := rows.Scan(&recipient.DonorID, &recipient.Token); err != nil {
			return nil, err
		}
		recipient.Token = strings.TrimSpace(recipient.Token)
		if recipient.Token == "" {
			continue
		}
		recipients = append(recipients, recipient)
	}
	return recipients, rows.Err()
}

func (s *Store) ExpireBroadcasts(ctx context.Context) (int64, error) {
	var expiredCount int64
	var requestsUpdated int64
	err := s.pool.QueryRow(ctx, `
		WITH expired AS (
			UPDATE emergency_broadcasts
			SET status = 'EXPIRED', closed_at = NOW()
			WHERE status = 'ACTIVE' AND expires_at < NOW()
			RETURNING broadcast_id
		), request_update AS (
			UPDATE blood_requests
			SET status = 'EXPIRED'
			WHERE broadcast_id IN (SELECT broadcast_id FROM expired)
			RETURNING 1
		)
		SELECT (SELECT COUNT(*) FROM expired), (SELECT COUNT(*) FROM request_update)
	`).Scan(&expiredCount, &requestsUpdated)
	if err != nil {
		return 0, err
	}
	return expiredCount, nil
}

func (s *Store) ListLiveResponses(ctx context.Context, requestID string) ([]LiveResponse, error) {
	if s.cfg.DemoMode {
		if err := s.advanceLiveResponses(ctx, requestID); err != nil {
			return nil, err
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, broadcast_id, donor_id::TEXT, donor_name, donor_blood,
		       distance_km::DOUBLE PRECISION, status::TEXT, COALESCE(response_at, created_at)
		FROM live_responses
		WHERE request_id::TEXT = $1 AND status <> 'NO_RESPONSE'
		ORDER BY response_at ASC, distance_km ASC
	`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []LiveResponse
	for rows.Next() {
		var response LiveResponse
		if err := rows.Scan(
			&response.ID,
			&response.BroadcastID,
			&response.DonorID,
			&response.DonorName,
			&response.BloodType,
			&response.DistanceKm,
			&response.Status,
			&response.RespondedAt,
		); err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, rows.Err()
}

func (s *Store) RespondToBroadcast(ctx context.Context, qrToken, broadcastID, status string) (LiveResponse, error) {
	qrToken = strings.TrimSpace(qrToken)
	broadcastID = strings.TrimSpace(broadcastID)
	status = strings.ToUpper(strings.TrimSpace(status))
	if qrToken == "" || broadcastID == "" || status == "" {
		return LiveResponse{}, errBadRequest("qr_token, broadcast_id, dan status wajib diisi")
	}
	if !validMobileResponseStatus(status) {
		return LiveResponse{}, errBadRequest("status respons tidak valid")
	}

	donor, err := s.activeDonorByQRToken(ctx, qrToken)
	if err != nil {
		return LiveResponse{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LiveResponse{}, err
	}
	defer tx.Rollback(ctx)

	var requestID string
	err = tx.QueryRow(ctx, `
		SELECT request_id::TEXT
		FROM emergency_broadcasts
		WHERE broadcast_id = $1 AND status = 'ACTIVE'
	`, broadcastID).Scan(&requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LiveResponse{}, errBadRequest("broadcast tidak aktif atau tidak ditemukan")
	}
	if err != nil {
		return LiveResponse{}, err
	}

	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status::TEXT
		FROM live_responses
		WHERE broadcast_id = $1 AND donor_id::TEXT = $2
		FOR UPDATE
	`, broadcastID, donor.ID).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return LiveResponse{}, errBadRequest("pendonor tidak termasuk daftar broadcast")
	}
	if err != nil {
		return LiveResponse{}, err
	}
	if !validMobileStatusTransition(currentStatus, status) {
		return LiveResponse{}, errBadRequest("transisi status respons tidak valid")
	}

	response, err := scanLiveResponse(tx.QueryRow(ctx, `
		UPDATE live_responses
		SET status = $3, response_at = NOW()
		WHERE broadcast_id = $1 AND donor_id::TEXT = $2
		RETURNING id::TEXT, broadcast_id, donor_id::TEXT, donor_name, donor_blood,
		          distance_km::DOUBLE PRECISION, status::TEXT, COALESCE(response_at, created_at)
	`, broadcastID, donor.ID, status))
	if err != nil {
		return LiveResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return LiveResponse{}, err
	}

	if err := s.refreshBroadcastSummary(ctx, requestID); err != nil {
		slog.Default().Error("refresh broadcast summary after mobile response", "broadcast_id", broadcastID, "request_id", requestID, "error", err)
	}

	return response, nil
}

func (s *Store) ActiveBroadcastForDonor(ctx context.Context, qrToken string) (MobileActiveBroadcast, bool, error) {
	donor, err := s.activeDonorByQRToken(ctx, qrToken)
	if err != nil {
		return MobileActiveBroadcast{}, false, err
	}

	request, err := scanRequest(s.pool.QueryRow(ctx, requestSelectSQL()+`
		JOIN emergency_broadcasts eb ON eb.broadcast_id = br.broadcast_id
		JOIN live_responses lr ON lr.broadcast_id = eb.broadcast_id AND lr.request_id = br.id
		WHERE eb.status = 'ACTIVE' AND lr.donor_id::TEXT = $1
		ORDER BY eb.created_at DESC
		LIMIT 1
	`, donor.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return MobileActiveBroadcast{}, false, nil
	}
	if err != nil {
		return MobileActiveBroadcast{}, false, err
	}

	response, err := s.liveResponseForDonor(ctx, request.BroadcastID, donor.ID)
	if err != nil {
		return MobileActiveBroadcast{}, false, err
	}

	return MobileActiveBroadcast{Broadcast: request, Response: response}, true, nil
}

func (s *Store) ListDonors(ctx context.Context, search string) ([]Donor, error) {
	pattern := "%" + strings.ToLower(search) + "%"
	rows, err := s.pool.Query(ctx, donorSelectSQL()+`
		WHERE $3 = '%%'
		   OR LOWER(u.full_name) LIKE $3
		   OR LOWER(u.phone) LIKE $3
		   OR LOWER(u.blood_type) LIKE $3
		ORDER BY u.created_at DESC, u.full_name ASC
	`, s.cfg.PMILongitude, s.cfg.PMILatitude, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDonors(rows)
}

func (s *Store) GetDonor(ctx context.Context, key string) (Donor, error) {
	lookupKey := s.decryptQRLookupKey(key)
	row := s.pool.QueryRow(ctx, donorSelectSQL()+`
		WHERE u.id::TEXT = $3 OR u.qr_token = $4 OR u.qr_token = $5
		LIMIT 1
	`, s.cfg.PMILongitude, s.cfg.PMILatitude, lookupKey, strings.TrimSpace(key), prefixedQRToken(key))
	donor, err := scanDonor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Donor{}, errNotFound("pendonor tidak ditemukan")
	}
	if err != nil {
		return Donor{}, err
	}

	history, err := s.DonationHistory(ctx, donor.ID)
	if err != nil {
		return Donor{}, err
	}
	donor.DonationHistory = history
	return donor, nil
}

func (s *Store) FindDonorByLogin(ctx context.Context, loginKey string) (Donor, error) {
	trimmed := strings.TrimSpace(loginKey)
	if trimmed == "" {
		return Donor{}, errBadRequest("kunci login tidak boleh kosong")
	}

	row := s.pool.QueryRow(ctx, donorSelectSQL()+`
		WHERE (LOWER(u.email) = LOWER($3) OR u.phone = $3 OR u.nik = $3) AND u.is_active = TRUE
		LIMIT 1
	`, s.cfg.PMILongitude, s.cfg.PMILatitude, trimmed)
	donor, err := scanDonor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Donor{}, errNotFound("akun pendonor tidak ditemukan")
	}
	if err != nil {
		return Donor{}, err
	}

	history, err := s.DonationHistory(ctx, donor.ID)
	if err != nil {
		return Donor{}, err
	}
	donor.DonationHistory = history
	return donor, nil
}

func (s *Store) CreateDonor(ctx context.Context, input DonorCreateRequest) (Donor, error) {
	if input.FullName == "" || input.Phone == "" || input.BloodType == "" || input.Gender == "" || input.Address == "" {
		return Donor{}, errBadRequest("data pendonor belum lengkap")
	}
	if input.NIK == "" {
		input.NIK = generateNIK()
	}
	if input.BirthDate == "" {
		input.BirthDate = "1990-01-01"
	}
	if input.Latitude == 0 {
		input.Latitude = s.cfg.PMILatitude + 0.04
	}
	if input.Longitude == 0 {
		input.Longitude = s.cfg.PMILongitude + 0.03
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Donor{}, err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (
			qr_token, nik, full_name, email, phone, blood_type, birth_date, gender, address,
			latitude, longitude, location, device_token, is_eligible, is_active
		)
		VALUES (
			'pending:' || gen_random_uuid()::TEXT, $1, $2, $3, $4, $5, $6::DATE, $7, $8,
			$9::DECIMAL, $10::DECIMAL, ST_SetSRID(ST_MakePoint($10::DOUBLE PRECISION, $9::DOUBLE PRECISION), 4326)::geography, $11, TRUE, TRUE
		)
		RETURNING id::TEXT
	`, input.NIK, input.FullName, nullString(input.Email), input.Phone, input.BloodType, input.BirthDate, input.Gender, input.Address, input.Latitude, input.Longitude, nullString(input.DeviceToken)).Scan(&id)
	if err != nil {
		return Donor{}, err
	}

	qrToken, err := s.encryptQRToken(id)
	if err != nil {
		return Donor{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET qr_token = $2, updated_at = NOW()
		WHERE id::TEXT = $1
	`, id, qrToken)
	if err != nil {
		return Donor{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Donor{}, err
	}
	return s.GetDonor(ctx, id)
}

func (s *Store) activeDonorByQRToken(ctx context.Context, qrToken string) (Donor, error) {
	trimmed := strings.TrimSpace(qrToken)
	if trimmed == "" {
		return Donor{}, errBadRequest("qr_token wajib diisi")
	}

	lookupKey := s.decryptQRLookupKey(trimmed)
	donor, err := scanDonor(s.pool.QueryRow(ctx, donorSelectSQL()+`
		WHERE (u.id::TEXT = $3 OR u.qr_token = $4 OR u.qr_token = $5)
		  AND u.is_active = TRUE
		LIMIT 1
	`, s.cfg.PMILongitude, s.cfg.PMILatitude, lookupKey, trimmed, prefixedQRToken(trimmed)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Donor{}, errNotFound("token donor tidak valid")
	}
	return donor, err
}

func (s *Store) ReencryptQRTokens(ctx context.Context, logger *slog.Logger) (int64, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if s.qrCipher == nil {
		return 0, errors.New("QR cipher is not configured")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, qr_token
		FROM users
		WHERE qr_token NOT LIKE 'encrypted:%'
		ORDER BY created_at ASC
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type candidate struct {
		ID      string
		QRToken string
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.ID, &item.QRToken); err != nil {
			return 0, err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	logger.Info("reencrypting QR tokens", "total", len(candidates))
	var updated int64
	for _, item := range candidates {
		qrToken, err := s.encryptQRToken(item.ID)
		if err != nil {
			return updated, err
		}
		tag, err := s.pool.Exec(ctx, `
			UPDATE users
			SET qr_token = $3, updated_at = NOW()
			WHERE id::TEXT = $1 AND qr_token = $2
		`, item.ID, item.QRToken, qrToken)
		if err != nil {
			return updated, err
		}
		updated += tag.RowsAffected()
		logger.Info("reencrypted QR token", "user_id", item.ID, "updated", tag.RowsAffected())
	}

	return updated, nil
}

func (s *Store) UpdateDonor(ctx context.Context, id string, input DonorUpdateRequest) (Donor, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE users
		SET full_name = COALESCE(NULLIF($2, ''), full_name),
		    email = $3,
		    phone = COALESCE(NULLIF($4, ''), phone),
		    blood_type = COALESCE(NULLIF($5, ''), blood_type),
		    gender = COALESCE(NULLIF($6, ''), gender),
		    address = COALESCE(NULLIF($7, ''), address),
		    latitude = CASE WHEN $8::DOUBLE PRECISION <> 0 THEN $8 ELSE latitude END,
		    longitude = CASE WHEN $9::DOUBLE PRECISION <> 0 THEN $9 ELSE longitude END,
		    location = ST_SetSRID(
		      ST_MakePoint(
		        CASE WHEN $9::DOUBLE PRECISION <> 0 THEN $9 ELSE longitude END,
		        CASE WHEN $8::DOUBLE PRECISION <> 0 THEN $8 ELSE latitude END
		      ),
		      4326
		    )::geography,
		    updated_at = NOW()
		WHERE id::TEXT = $1
	`, id, input.FullName, nullString(input.Email), input.Phone, input.BloodType, input.Gender, input.Address, input.Latitude, input.Longitude)
	if err != nil {
		return Donor{}, err
	}
	return s.GetDonor(ctx, id)
}

func (s *Store) UpdateDonorByQRToken(ctx context.Context, qrToken string, input UpdateDonorRequest) (*Donor, error) {
	query := `UPDATE users SET
		full_name   = COALESCE(NULLIF($2,''), full_name),
		email       = COALESCE(NULLIF($3,''), email),
		phone       = COALESCE(NULLIF($4,''), phone),
		address     = COALESCE(NULLIF($5,''), address),
		latitude    = CASE WHEN $6 != 0 THEN $6 ELSE latitude END,
		longitude   = CASE WHEN $7 != 0 THEN $7 ELSE longitude END,
		updated_at  = NOW()
	WHERE qr_token = $1
	RETURNING id::TEXT, qr_token, full_name, phone, COALESCE(email,''),
	          blood_type, gender, COALESCE(address,''),
	          0::DOUBLE PRECISION AS distance_km,
	          COALESCE(to_char(last_donation,'YYYY-MM-DD'),''),
	          COALESCE(to_char(next_eligible,'YYYY-MM-DD'),''),
	          is_eligible, is_active`
	row := s.pool.QueryRow(ctx, query, qrToken,
		input.FullName, input.Email, input.Phone, input.Address, input.Latitude, input.Longitude)
	donor, err := scanDonor(row)
	if err != nil {
		return nil, err
	}
	return &donor, nil
}

func (s *Store) CloseRequest(ctx context.Context, requestID string) (EmergencyRequest, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EmergencyRequest{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE blood_requests
		SET status = 'CLOSED', fulfilled_at = NOW()
		WHERE id::TEXT = $1
	`, requestID)
	if err != nil {
		return EmergencyRequest{}, err
	}
	if tag.RowsAffected() == 0 {
		return EmergencyRequest{}, errNotFound("permintaan tidak ditemukan")
	}

	_, err = tx.Exec(ctx, `
		UPDATE emergency_broadcasts
		SET status = 'CLOSED', closed_at = NOW()
		WHERE request_id::TEXT = $1 AND status = 'ACTIVE'
	`, requestID)
	if err != nil {
		return EmergencyRequest{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return EmergencyRequest{}, err
	}
	return s.GetRequest(ctx, requestID)
}

func (s *Store) UpdateDonorStatus(ctx context.Context, id string, isActive bool) (Donor, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users
		SET is_active = $2, updated_at = NOW()
		WHERE id::TEXT = $1
	`, id, isActive)
	if err != nil {
		return Donor{}, err
	}
	if tag.RowsAffected() == 0 {
		return Donor{}, errNotFound("pendonor tidak ditemukan")
	}
	return s.GetDonor(ctx, id)
}

func (s *Store) DonationHistory(ctx context.Context, donorID string) ([]DonationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::TEXT, to_char(donation_date, 'YYYY-MM-DD'), COALESCE(pmi_location, ''),
		       COALESCE(request_id::TEXT, ''), COALESCE(blood_pressure, ''),
		       COALESCE(hemoglobin::DOUBLE PRECISION, 0), COALESCE(weight::DOUBLE PRECISION, 0), status::TEXT
		FROM donation_history
		WHERE donor_id::TEXT = $1
		ORDER BY donation_date DESC, created_at DESC
	`, donorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DonationRecord
	for rows.Next() {
		var record DonationRecord
		if err := rows.Scan(
			&record.ID,
			&record.Date,
			&record.Location,
			&record.RequestID,
			&record.BloodPressure,
			&record.Hemoglobin,
			&record.Weight,
			&record.Status,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) CheckinDonation(ctx context.Context, input DonationCheckinRequest, adminID string) (DonationCheckinResult, error) {
	if input.DonorUUID == "" {
		return DonationCheckinResult{}, errBadRequest("token donor wajib diisi")
	}
	donor, err := s.GetDonor(ctx, input.DonorUUID)
	if err != nil {
		return DonationCheckinResult{}, err
	}

	eligible, reasons := validateMedical(donor, input)
	status := "COMPLETED"
	if !eligible {
		status = "REJECTED"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DonationCheckinResult{}, err
	}
	defer tx.Rollback(ctx)

	var donationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO donation_history (
			donor_id, request_id, donation_date, pmi_location, blood_pressure,
			hemoglobin, weight, is_eligible, disqualify_reason, status, admin_id
		)
		VALUES ($1, $2, CURRENT_DATE, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::TEXT
	`, donor.ID, nullString(input.RequestID), s.cfg.PMILocation, fmt.Sprintf("%d/%d", input.Systolic, input.Diastolic),
		input.Hemoglobin, input.Weight, eligible, strings.Join(reasons, ", "), status, nullString(adminID)).Scan(&donationID)
	if err != nil {
		return DonationCheckinResult{}, err
	}

	if eligible {
		_, err = tx.Exec(ctx, `
			UPDATE users
			SET last_donation = CURRENT_DATE, is_eligible = FALSE, updated_at = NOW()
			WHERE id::TEXT = $1
		`, donor.ID)
		if err != nil {
			return DonationCheckinResult{}, err
		}
	}

	if input.RequestID != "" {
		_, err = tx.Exec(ctx, `
			UPDATE live_responses
			SET status = 'CHECKED_IN',
			    checkin_at = NOW(),
			    response_at = COALESCE(response_at, NOW())
			WHERE request_id::TEXT = $1 AND donor_id::TEXT = $2
		`, input.RequestID, donor.ID)
		if err != nil {
			return DonationCheckinResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return DonationCheckinResult{}, err
	}

	if input.RequestID != "" {
		_ = s.refreshBroadcastSummary(ctx, input.RequestID)
	}

	return DonationCheckinResult{DonationID: donationID, IsEligible: eligible, Reasons: reasons}, nil
}

func (s *Store) findOrCreateHospital(ctx context.Context, input EmergencyCreateRequest) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id::TEXT FROM hospitals
		WHERE LOWER(name) = LOWER($1)
		LIMIT 1
	`, input.HospitalName).Scan(&id)
	if err == nil {
		_, _ = s.pool.Exec(ctx, `
			UPDATE hospitals
			SET pic_name = COALESCE(NULLIF($2, ''), pic_name),
			    pic_phone = COALESCE(NULLIF($3, ''), pic_phone)
			WHERE id::TEXT = $1
		`, id, input.PicName, input.PicPhone)
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO hospitals (name, pic_name, pic_phone, latitude, longitude, is_active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		RETURNING id::TEXT
	`, input.HospitalName, input.PicName, input.PicPhone, s.cfg.PMILatitude, s.cfg.PMILongitude).Scan(&id)
	return id, err
}

func (s *Store) advanceLiveResponses(ctx context.Context, requestID string) error {
	var broadcastID string
	var sentAt sql.NullTime
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(broadcast_id, ''), broadcast_sent_at
		FROM blood_requests
		WHERE id::TEXT = $1
	`, requestID).Scan(&broadcastID, &sentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound("permintaan tidak ditemukan")
	}
	if err != nil || !sentAt.Valid || broadcastID == "" {
		return err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT donor_id::TEXT, status::TEXT
		FROM live_responses
		WHERE request_id::TEXT = $1
		ORDER BY distance_km ASC, created_at ASC
	`, requestID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type candidate struct {
		DonorID string
		Status  string
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.DonorID, &item.Status); err != nil {
			return err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	statuses := []string{"ACCEPTED", "ON_THE_WAY", "ACCEPTED", "DECLINED", "CHECKED_IN", "ON_THE_WAY", "ACCEPTED"}
	offsets := []time.Duration{1500 * time.Millisecond, 3900 * time.Millisecond, 6300 * time.Millisecond, 8700 * time.Millisecond, 11100 * time.Millisecond, 13500 * time.Millisecond, 15900 * time.Millisecond}
	elapsed := time.Since(sentAt.Time)

	for i, item := range candidates {
		if i >= len(statuses) || elapsed < offsets[i] || item.Status != "NO_RESPONSE" {
			continue
		}
		responseAt := sentAt.Time.Add(offsets[i])
		_, err = s.pool.Exec(ctx, `
			UPDATE live_responses
			SET status = $3, response_at = $4
			WHERE request_id::TEXT = $1 AND donor_id::TEXT = $2 AND status = 'NO_RESPONSE'
		`, requestID, item.DonorID, statuses[i], responseAt)
		if err != nil {
			return err
		}
	}

	return s.refreshBroadcastSummary(ctx, requestID)
}

func (s *Store) refreshBroadcastSummary(ctx context.Context, requestID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE emergency_broadcasts eb
		SET responded_count = summary.responded_count,
		    accepted_count = summary.accepted_count,
		    checkedin_count = summary.checkedin_count
		FROM (
		  SELECT broadcast_id,
		         COUNT(*) FILTER (WHERE status <> 'NO_RESPONSE') AS responded_count,
		         COUNT(*) FILTER (WHERE status = 'ACCEPTED') AS accepted_count,
		         COUNT(*) FILTER (WHERE status = 'CHECKED_IN') AS checkedin_count
		  FROM live_responses
		  WHERE request_id::TEXT = $1
		  GROUP BY broadcast_id
		) AS summary
		WHERE eb.broadcast_id = summary.broadcast_id
	`, requestID)
	return err
}

func scanStock(row pgx.Row) (BloodStock, error) {
	var item BloodStock
	err := row.Scan(
		&item.ID,
		&item.BloodType,
		&item.ProductType,
		&item.Quantity,
		&item.SafeThreshold,
		&item.CriticalThreshold,
		&item.UpdatedAt,
	)
	return item, err
}

func scanHospital(row pgx.Row) (Hospital, error) {
	var hospital Hospital
	err := row.Scan(
		&hospital.ID,
		&hospital.Name,
		&hospital.Address,
		&hospital.Latitude,
		&hospital.Longitude,
		&hospital.PicName,
		&hospital.PicPhone,
		&hospital.Email,
		&hospital.IsActive,
	)
	return hospital, err
}

func scanRequest(row pgx.Row) (EmergencyRequest, error) {
	var request EmergencyRequest
	var broadcastSentAt sql.NullTime
	var fulfilledAt sql.NullTime
	err := row.Scan(
		&request.ID,
		&request.HospitalName,
		&request.PicName,
		&request.PicPhone,
		&request.BloodType,
		&request.ProductType,
		&request.QuantityNeeded,
		&request.UrgencyLevel,
		&request.Notes,
		&request.Status,
		&request.BroadcastID,
		&request.EligibleCount,
		&request.CreatedAt,
		&broadcastSentAt,
		&fulfilledAt,
	)
	if broadcastSentAt.Valid {
		request.BroadcastSentAt = &broadcastSentAt.Time
	}
	if fulfilledAt.Valid {
		request.FulfilledAt = &fulfilledAt.Time
	}
	return request, err
}

func scanDonors(rows pgx.Rows) ([]Donor, error) {
	var donors []Donor
	for rows.Next() {
		donor, err := scanDonor(rows)
		if err != nil {
			return nil, err
		}
		donors = append(donors, donor)
	}
	return donors, rows.Err()
}

func scanDonor(row pgx.Row) (Donor, error) {
	var donor Donor
	err := row.Scan(
		&donor.ID,
		&donor.UUID,
		&donor.FullName,
		&donor.Phone,
		&donor.Email,
		&donor.BloodType,
		&donor.Gender,
		&donor.Address,
		&donor.DistanceKm,
		&donor.LastDonation,
		&donor.NextEligible,
		&donor.IsEligible,
		&donor.IsActive,
	)
	if math.IsNaN(donor.DistanceKm) {
		donor.DistanceKm = 0
	}
	donor.UUID = publicQRToken(donor.UUID)
	return donor, err
}

func scanLiveResponse(row pgx.Row) (LiveResponse, error) {
	var response LiveResponse
	err := row.Scan(
		&response.ID,
		&response.BroadcastID,
		&response.DonorID,
		&response.DonorName,
		&response.BloodType,
		&response.DistanceKm,
		&response.Status,
		&response.RespondedAt,
	)
	return response, err
}

func (s *Store) liveResponseForDonor(ctx context.Context, broadcastID, donorID string) (LiveResponse, error) {
	response, err := scanLiveResponse(s.pool.QueryRow(ctx, `
		SELECT id::TEXT, broadcast_id, donor_id::TEXT, donor_name, donor_blood,
		       distance_km::DOUBLE PRECISION, status::TEXT, COALESCE(response_at, created_at)
		FROM live_responses
		WHERE broadcast_id = $1 AND donor_id::TEXT = $2
		LIMIT 1
	`, broadcastID, donorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return LiveResponse{}, errBadRequest("pendonor tidak termasuk daftar broadcast")
	}
	return response, err
}

func requestSelectSQL() string {
	return `
		SELECT br.id::TEXT,
		       COALESCE(h.name, ''),
		       COALESCE(br.requested_by, h.pic_name, ''),
		       COALESCE(h.pic_phone, ''),
		       br.blood_type,
		       br.product_type,
		       br.quantity_needed,
		       br.urgency_level::TEXT,
		       COALESCE(br.notes, ''),
		       br.status::TEXT,
		       COALESCE(br.broadcast_id, ''),
		       br.eligible_count,
		       br.created_at,
		       br.broadcast_sent_at,
		       br.fulfilled_at
		FROM blood_requests br
		LEFT JOIN hospitals h ON h.id = br.hospital_id
	`
}

func donorSelectSQL() string {
	return `
		SELECT u.id::TEXT,
		       u.qr_token,
		       u.full_name,
		       u.phone,
		       COALESCE(u.email, ''),
		       u.blood_type,
		       u.gender,
		       COALESCE(u.address, ''),
		       COALESCE(ROUND((ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) / 1000)::NUMERIC, 1)::DOUBLE PRECISION, 0) AS distance_km,
		       COALESCE(to_char(u.last_donation, 'YYYY-MM-DD'), ''),
		       COALESCE(to_char(u.next_eligible, 'YYYY-MM-DD'), ''),
		       u.is_eligible,
		       u.is_active
		FROM users u
	`
}

func validateMedical(donor Donor, input DonationCheckinRequest) (bool, []string) {
	var reasons []string
	hbMinimum := 13.0
	if donor.Gender == "F" {
		hbMinimum = 12.5
	}

	if !donor.IsActive {
		reasons = append(reasons, "pendonor nonaktif")
	}
	if !donor.IsEligible {
		reasons = append(reasons, "jeda donor belum terpenuhi")
	}
	if input.Hemoglobin < hbMinimum {
		reasons = append(reasons, fmt.Sprintf("Hb minimal %.1f g/dL", hbMinimum))
	}
	if input.Systolic < 100 || input.Systolic > 170 {
		reasons = append(reasons, "sistolik harus 100-170 mmHg")
	}
	if input.Diastolic < 70 || input.Diastolic > 100 {
		reasons = append(reasons, "diastolik harus 70-100 mmHg")
	}
	if input.Weight < 45 {
		reasons = append(reasons, "berat badan minimal 45 kg")
	}

	return len(reasons) == 0, reasons
}

func normalizeHospital(input HospitalRequest, cfg Config) HospitalRequest {
	if input.Latitude == 0 {
		input.Latitude = cfg.PMILatitude
	}
	if input.Longitude == 0 {
		input.Longitude = cfg.PMILongitude
	}
	return input
}

func nullString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func generateNIK() string {
	return fmt.Sprintf("9%015d", time.Now().UnixNano()%1_000_000_000_000_000)
}

type appError struct {
	status  int
	code    string
	message string
}

func (e appError) Error() string {
	return e.message
}

func errBadRequest(message string) error {
	return appError{status: 400, code: "BAD_REQUEST", message: message}
}

func errNotFound(message string) error {
	return appError{status: 404, code: "NOT_FOUND", message: message}
}

func validMobileResponseStatus(status string) bool {
	return status == "ACCEPTED" || status == "DECLINED" || status == "ON_THE_WAY"
}

func validMobileStatusTransition(current, next string) bool {
	switch current {
	case "NO_RESPONSE":
		return next == "ACCEPTED" || next == "DECLINED"
	case "ACCEPTED":
		return next == "ON_THE_WAY"
	default:
		return false
	}
}

func (s *Store) encryptQRToken(userID string) (string, error) {
	if s.qrCipher == nil {
		return "", errors.New("QR cipher is not configured")
	}
	ciphertext, err := s.qrCipher.Encrypt(userID)
	if err != nil {
		return "", err
	}
	return qrTokenEncryptedPrefix + ciphertext, nil
}

func (s *Store) decryptQRLookupKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if s.qrCipher == nil {
		return trimmed
	}

	ciphertext := strings.TrimPrefix(trimmed, qrTokenEncryptedPrefix)
	plaintext, err := s.qrCipher.Decrypt(ciphertext)
	if err != nil {
		return trimmed
	}
	return plaintext
}

func prefixedQRToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" || strings.HasPrefix(trimmed, qrTokenEncryptedPrefix) {
		return trimmed
	}
	return qrTokenEncryptedPrefix + trimmed
}

func publicQRToken(token string) string {
	return strings.TrimPrefix(token, qrTokenEncryptedPrefix)
}
