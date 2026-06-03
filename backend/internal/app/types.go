package app

import "time"

type AdminUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
	Role     string `json:"role"`
}

type BloodStock struct {
	ID                string    `json:"id"`
	BloodType         string    `json:"bloodType"`
	ProductType       string    `json:"productType"`
	Quantity          int       `json:"quantity"`
	SafeThreshold     int       `json:"safeThreshold"`
	CriticalThreshold int       `json:"criticalThreshold"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type EmergencyRequest struct {
	ID              string     `json:"id"`
	HospitalName    string     `json:"hospitalName"`
	PicName         string     `json:"picName"`
	PicPhone        string     `json:"picPhone"`
	BloodType       string     `json:"bloodType"`
	ProductType     string     `json:"productType"`
	QuantityNeeded  int        `json:"quantityNeeded"`
	UrgencyLevel    string     `json:"urgencyLevel"`
	Notes           string     `json:"notes"`
	Status          string     `json:"status"`
	BroadcastID     string     `json:"broadcastId,omitempty"`
	EligibleCount   int        `json:"eligibleCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	BroadcastSentAt *time.Time `json:"broadcastSentAt,omitempty"`
	FulfilledAt     *time.Time `json:"fulfilledAt,omitempty"`
}

type DonationRecord struct {
	ID            string  `json:"id"`
	Date          string  `json:"date"`
	Location      string  `json:"location"`
	RequestID     string  `json:"requestId,omitempty"`
	BloodPressure string  `json:"bloodPressure"`
	Hemoglobin    float64 `json:"hemoglobin"`
	Weight        float64 `json:"weight"`
	Status        string  `json:"status"`
}

type Donor struct {
	ID              string           `json:"id"`
	UUID            string           `json:"uuid"`
	FullName        string           `json:"fullName"`
	Phone           string           `json:"phone"`
	Email           string           `json:"email,omitempty"`
	BloodType       string           `json:"bloodType"`
	Gender          string           `json:"gender"`
	Address         string           `json:"address"`
	DistanceKm      float64          `json:"distanceKm"`
	LastDonation    string           `json:"lastDonation,omitempty"`
	NextEligible    string           `json:"nextEligible,omitempty"`
	IsEligible      bool             `json:"isEligible"`
	IsActive        bool             `json:"isActive"`
	DonationHistory []DonationRecord `json:"donationHistory,omitempty"`
}

type Hospital struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PicName   string  `json:"picName"`
	PicPhone  string  `json:"picPhone"`
	Email     string  `json:"email,omitempty"`
	IsActive  bool    `json:"isActive"`
}

type LiveResponse struct {
	ID          string    `json:"id"`
	BroadcastID string    `json:"broadcastId"`
	DonorID     string    `json:"donorId"`
	DonorName   string    `json:"donorName"`
	BloodType   string    `json:"bloodType"`
	DistanceKm  float64   `json:"distanceKm"`
	Status      string    `json:"status"`
	RespondedAt time.Time `json:"respondedAt"`
}

type MobileActiveBroadcast struct {
	Broadcast EmergencyRequest `json:"broadcast"`
	Response  LiveResponse     `json:"response"`
}

type BroadcastResult struct {
	BroadcastID    string    `json:"broadcastId"`
	RecipientCount int       `json:"recipientCount"`
	QueuedAt       time.Time `json:"queuedAt"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type MobileRespondRequest struct {
	QRToken     string `json:"qr_token"`
	BroadcastID string `json:"broadcast_id"`
	Status      string `json:"status"`
}

type StockUpdateRequest struct {
	Mode      string `json:"mode"`
	Quantity  int    `json:"quantity"`
	Reference string `json:"reference"`
	Notes     string `json:"notes"`
}

type EmergencyCreateRequest struct {
	HospitalName   string `json:"hospitalName"`
	PicName        string `json:"picName"`
	PicPhone       string `json:"picPhone"`
	BloodType      string `json:"bloodType"`
	ProductType    string `json:"productType"`
	QuantityNeeded int    `json:"quantityNeeded"`
	UrgencyLevel   string `json:"urgencyLevel"`
	Notes          string `json:"notes"`
}

type DonorCreateRequest struct {
	NIK         string  `json:"nik"`
	FullName    string  `json:"fullName"`
	Email       string  `json:"email"`
	Password    string  `json:"password,omitempty"`
	Phone       string  `json:"phone"`
	BloodType   string  `json:"bloodType"`
	Gender      string  `json:"gender"`
	BirthDate   string  `json:"birthDate"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DeviceToken string  `json:"deviceToken"`
}

type DonorUpdateRequest struct {
	FullName  string  `json:"fullName"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	BloodType string  `json:"bloodType"`
	Gender    string  `json:"gender"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DonorStatusRequest struct {
	IsActive bool `json:"isActive"`
}

type DonationCheckinRequest struct {
	DonorUUID  string  `json:"donorUuid"`
	RequestID  string  `json:"requestId"`
	Systolic   int     `json:"systolic"`
	Diastolic  int     `json:"diastolic"`
	Hemoglobin float64 `json:"hemoglobin"`
	Weight     float64 `json:"weight"`
}

type DonationCheckinResult struct {
	DonationID string   `json:"donationId"`
	IsEligible bool     `json:"isEligible"`
	Reasons    []string `json:"reasons"`
	DonorID    string   `json:"-"`
	BloodType  string   `json:"-"`
	ProductType string `json:"-"`
}

type HospitalRequest struct {
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PicName   string  `json:"picName"`
	PicPhone  string  `json:"picPhone"`
	Email     string  `json:"email"`
}

type UpdateDonorRequest struct {
	FullName  string  `json:"fullName"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DeviceTokenRequest struct {
	QRToken     string `json:"qr_token"`
	DeviceToken string `json:"device_token"`
}

type DonorAuthResponse struct {
	Token string `json:"token"`
	Donor
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type DonationHistoryPage struct {
	Items      []DonationRecord `json:"items"`
	Pagination Pagination       `json:"pagination"`
}
