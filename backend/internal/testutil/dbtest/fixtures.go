package dbtest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saas/city-stories-guide/backend/internal/domain"
)

// CityOpts overrides defaults for InsertCity.
type CityOpts struct {
	Name           string
	Country        string
	CenterLat      float64
	CenterLng      float64
	RadiusKm       float64
	IsActive       *bool
	DownloadSizeMB float64
}

// InsertCity creates a city row with sensible defaults. Override any field via opts.
func InsertCity(t *testing.T, pool *pgxpool.Pool, opts ...CityOpts) *domain.City {
	t.Helper()

	o := CityOpts{
		Name:      "Test City",
		Country:   "Testland",
		CenterLat: 41.7151,
		CenterLng: 44.8271,
		RadiusKm:  10.0,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.Name != "" {
			o.Name = merge.Name
		}
		if merge.Country != "" {
			o.Country = merge.Country
		}
		if merge.CenterLat != 0 {
			o.CenterLat = merge.CenterLat
		}
		if merge.CenterLng != 0 {
			o.CenterLng = merge.CenterLng
		}
		if merge.RadiusKm != 0 {
			o.RadiusKm = merge.RadiusKm
		}
		if merge.IsActive != nil {
			o.IsActive = merge.IsActive
		}
		if merge.DownloadSizeMB != 0 {
			o.DownloadSizeMB = merge.DownloadSizeMB
		}
	}

	isActive := true
	if o.IsActive != nil {
		isActive = *o.IsActive
	}

	city := &domain.City{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO cities (name, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, country, center_lat, center_lng, radius_km, is_active, download_size_mb, deleted_at, created_at, updated_at`,
		o.Name, o.Country, o.CenterLat, o.CenterLng, o.RadiusKm, isActive, o.DownloadSizeMB,
	).Scan(&city.ID, &city.Name, &city.Country, &city.CenterLat, &city.CenterLng,
		&city.RadiusKm, &city.IsActive, &city.DownloadSizeMB, &city.DeletedAt, &city.CreatedAt, &city.UpdatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert city: %v", err)
	}
	return city
}

// UserOpts overrides defaults for InsertUser.
type UserOpts struct {
	Email        string
	Name         string
	AuthProvider domain.AuthProvider
	LanguagePref string
	IsAnonymous  *bool
	IsAdmin      *bool
}

// InsertUser creates a user row with sensible defaults.
func InsertUser(t *testing.T, pool *pgxpool.Pool, opts ...UserOpts) *domain.User {
	t.Helper()

	o := UserOpts{
		AuthProvider: domain.AuthProviderEmail,
		LanguagePref: "en",
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.Email != "" {
			o.Email = merge.Email
		}
		if merge.Name != "" {
			o.Name = merge.Name
		}
		if merge.AuthProvider != "" {
			o.AuthProvider = merge.AuthProvider
		}
		if merge.LanguagePref != "" {
			o.LanguagePref = merge.LanguagePref
		}
		if merge.IsAnonymous != nil {
			o.IsAnonymous = merge.IsAnonymous
		}
		if merge.IsAdmin != nil {
			o.IsAdmin = merge.IsAdmin
		}
	}

	isAnonymous := true
	if o.IsAnonymous != nil {
		isAnonymous = *o.IsAnonymous
	}
	isAdmin := false
	if o.IsAdmin != nil {
		isAdmin = *o.IsAdmin
	}

	var emailPtr, namePtr *string
	if o.Email != "" {
		emailPtr = &o.Email
	}
	if o.Name != "" {
		namePtr = &o.Name
	}

	user := &domain.User{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, name, auth_provider, language_pref, is_anonymous, is_admin)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, name, auth_provider, language_pref, is_anonymous, is_admin, created_at, updated_at`,
		emailPtr, namePtr, o.AuthProvider, o.LanguagePref, isAnonymous, isAdmin,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AuthProvider, &user.LanguagePref,
		&user.IsAnonymous, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert user: %v", err)
	}
	return user
}

// POIOpts overrides defaults for InsertPOI.
type POIOpts struct {
	CityID        int
	Name          string
	Lat           float64
	Lng           float64
	Type          domain.POIType
	InterestScore int16
	Status        domain.POIStatus
}

// InsertPOI creates a POI row. CityID is required (pass 0 to auto-create a city).
func InsertPOI(t *testing.T, pool *pgxpool.Pool, opts ...POIOpts) *domain.POI {
	t.Helper()

	o := POIOpts{
		Name:          "Test POI",
		Lat:           41.7151,
		Lng:           44.8271,
		Type:          domain.POITypeBuilding,
		InterestScore: 50,
		Status:        domain.POIStatusActive,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.CityID != 0 {
			o.CityID = merge.CityID
		}
		if merge.Name != "" {
			o.Name = merge.Name
		}
		if merge.Lat != 0 {
			o.Lat = merge.Lat
		}
		if merge.Lng != 0 {
			o.Lng = merge.Lng
		}
		if merge.Type != "" {
			o.Type = merge.Type
		}
		if merge.InterestScore != 0 {
			o.InterestScore = merge.InterestScore
		}
		if merge.Status != "" {
			o.Status = merge.Status
		}
	}

	if o.CityID == 0 {
		city := InsertCity(t, pool)
		o.CityID = city.ID
	}

	poi := &domain.POI{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO poi (city_id, name, location, type, interest_score, status)
		 VALUES ($1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5, $6, $7)
		 RETURNING id, city_id, name, ST_Y(location::geometry) AS lat, ST_X(location::geometry) AS lng,
		           type, tags, interest_score, status, created_at, updated_at`,
		o.CityID, o.Name, o.Lat, o.Lng, o.Type, o.InterestScore, o.Status,
	).Scan(&poi.ID, &poi.CityID, &poi.Name, &poi.Lat, &poi.Lng,
		&poi.Type, &poi.Tags, &poi.InterestScore, &poi.Status, &poi.CreatedAt, &poi.UpdatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert poi: %v", err)
	}
	return poi
}

// StoryOpts overrides defaults for InsertStory.
type StoryOpts struct {
	POIID      int
	Language   string
	Text       string
	LayerType  domain.StoryLayerType
	OrderIndex int16
	Status     domain.StoryStatus
}

// InsertStory creates a story row. POIID is required (pass 0 to auto-create a POI).
func InsertStory(t *testing.T, pool *pgxpool.Pool, opts ...StoryOpts) *domain.Story {
	t.Helper()

	o := StoryOpts{
		Language:  "en",
		Text:      "Once upon a time in a test...",
		LayerType: domain.StoryLayerGeneral,
		Status:    domain.StoryStatusActive,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.POIID != 0 {
			o.POIID = merge.POIID
		}
		if merge.Language != "" {
			o.Language = merge.Language
		}
		if merge.Text != "" {
			o.Text = merge.Text
		}
		if merge.LayerType != "" {
			o.LayerType = merge.LayerType
		}
		if merge.OrderIndex != 0 {
			o.OrderIndex = merge.OrderIndex
		}
		if merge.Status != "" {
			o.Status = merge.Status
		}
	}

	if o.POIID == 0 {
		poi := InsertPOI(t, pool)
		o.POIID = poi.ID
	}

	story := &domain.Story{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO story (poi_id, language, text, layer_type, order_index, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, poi_id, language, text, audio_url, duration_sec, layer_type,
		           order_index, is_inflation, confidence, sources, status, created_at, updated_at`,
		o.POIID, o.Language, o.Text, o.LayerType, o.OrderIndex, o.Status,
	).Scan(&story.ID, &story.POIID, &story.Language, &story.Text, &story.AudioURL,
		&story.DurationSec, &story.LayerType, &story.OrderIndex, &story.IsInflation,
		&story.Confidence, &story.Sources, &story.Status, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert story: %v", err)
	}
	return story
}

// ReportOpts overrides defaults for InsertReport.
type ReportOpts struct {
	StoryID int
	UserID  string
	Type    domain.ReportType
	Comment string
}

// InsertReport creates a report row. StoryID and UserID are auto-created if zero/empty.
func InsertReport(t *testing.T, pool *pgxpool.Pool, opts ...ReportOpts) *domain.Report {
	t.Helper()

	o := ReportOpts{
		Type: domain.ReportTypeWrongFact,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.StoryID != 0 {
			o.StoryID = merge.StoryID
		}
		if merge.UserID != "" {
			o.UserID = merge.UserID
		}
		if merge.Type != "" {
			o.Type = merge.Type
		}
		if merge.Comment != "" {
			o.Comment = merge.Comment
		}
	}

	if o.StoryID == 0 {
		story := InsertStory(t, pool)
		o.StoryID = story.ID
	}
	if o.UserID == "" {
		user := InsertUser(t, pool)
		o.UserID = user.ID
	}

	var commentPtr *string
	if o.Comment != "" {
		commentPtr = &o.Comment
	}

	report := &domain.Report{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO report (story_id, user_id, type, comment)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, story_id, user_id, type, comment, status, created_at`,
		o.StoryID, o.UserID, o.Type, commentPtr,
	).Scan(&report.ID, &report.StoryID, &report.UserID, &report.Type,
		&report.Comment, &report.Status, &report.CreatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert report: %v", err)
	}
	return report
}

// PurchaseOpts overrides defaults for InsertPurchase.
type PurchaseOpts struct {
	UserID   string
	Type     domain.PurchaseType
	CityID   *int
	Platform string
	Price    float64
}

// InsertPurchase creates a purchase row.
func InsertPurchase(t *testing.T, pool *pgxpool.Pool, opts ...PurchaseOpts) *domain.Purchase {
	t.Helper()

	o := PurchaseOpts{
		Type:     domain.PurchaseTypeCityPack,
		Platform: "ios",
		Price:    4.99,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.UserID != "" {
			o.UserID = merge.UserID
		}
		if merge.Type != "" {
			o.Type = merge.Type
		}
		if merge.CityID != nil {
			o.CityID = merge.CityID
		}
		if merge.Platform != "" {
			o.Platform = merge.Platform
		}
		if merge.Price != 0 {
			o.Price = merge.Price
		}
	}

	if o.UserID == "" {
		user := InsertUser(t, pool)
		o.UserID = user.ID
	}

	purchase := &domain.Purchase{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO purchase (user_id, type, city_id, platform, price)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, type, city_id, platform, price, is_ltd, expires_at, created_at`,
		o.UserID, o.Type, o.CityID, o.Platform, o.Price,
	).Scan(&purchase.ID, &purchase.UserID, &purchase.Type, &purchase.CityID,
		&purchase.Platform, &purchase.Price, &purchase.IsLTD, &purchase.ExpiresAt, &purchase.CreatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert purchase: %v", err)
	}
	return purchase
}

// DeviceTokenOpts overrides defaults for InsertDeviceToken.
type DeviceTokenOpts struct {
	UserID   string
	Token    string
	Platform domain.DevicePlatform
}

// InsertDeviceToken creates a device_token row.
func InsertDeviceToken(t *testing.T, pool *pgxpool.Pool, opts ...DeviceTokenOpts) *domain.DeviceToken {
	t.Helper()

	o := DeviceTokenOpts{
		Token:    "test-token-" + randomSuffix(),
		Platform: domain.DevicePlatformIOS,
	}
	if len(opts) > 0 {
		merge := opts[0]
		if merge.UserID != "" {
			o.UserID = merge.UserID
		}
		if merge.Token != "" {
			o.Token = merge.Token
		}
		if merge.Platform != "" {
			o.Platform = merge.Platform
		}
	}

	if o.UserID == "" {
		user := InsertUser(t, pool)
		o.UserID = user.ID
	}

	dt := &domain.DeviceToken{}
	err := pool.QueryRow(context.Background(),
		`INSERT INTO device_tokens (user_id, token, platform)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, token, platform, is_active, created_at, updated_at`,
		o.UserID, o.Token, o.Platform,
	).Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.IsActive, &dt.CreatedAt, &dt.UpdatedAt)
	if err != nil {
		t.Fatalf("dbtest: insert device token: %v", err)
	}
	return dt
}

// BoolPtr returns a pointer to a bool value (convenience for opts structs).
func BoolPtr(v bool) *bool { return &v }

// IntPtr returns a pointer to an int value (convenience for opts structs).
func IntPtr(v int) *int { return &v }

func randomSuffix() string {
	b, _ := json.Marshal(struct{ T int64 }{time.Now().UnixNano()})
	return string(b[5 : len(b)-1]) // extract the numeric part
}
