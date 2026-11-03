package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type ListingInput struct {
	RegionCode       string    `json:"regionCode"`
	AvailableLiters  int64     `json:"availableLiters"`
	FatPercent       float64   `json:"fatPercent"`
	PricePerLiterUZS int64     `json:"pricePerLiterUzs"`
	AvailableFrom    time.Time `json:"availableFrom"`
	AvailableUntil   time.Time `json:"availableUntil"`
}

type ListingUpdate struct {
	AvailableLiters  *int64     `json:"availableLiters"`
	FatPercent       *float64   `json:"fatPercent"`
	PricePerLiterUZS *int64     `json:"pricePerLiterUzs"`
	AvailableFrom    *time.Time `json:"availableFrom"`
	AvailableUntil   *time.Time `json:"availableUntil"`
	Status           *string    `json:"status"`
}

type Listing struct {
	ID               string    `json:"id"`
	FarmerID         string    `json:"farmerId"`
	RegionCode       string    `json:"regionCode"`
	AvailableLiters  int64     `json:"availableLiters"`
	FatPercent       float64   `json:"fatPercent"`
	PricePerLiterUZS int64     `json:"pricePerLiterUzs"`
	AvailableFrom    time.Time `json:"availableFrom"`
	AvailableUntil   time.Time `json:"availableUntil"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type OrderInput struct {
	ListingID        string    `json:"listingId"`
	Liters           int64     `json:"liters"`
	ExpectedPickupAt time.Time `json:"expectedPickupAt"`
	Note             string    `json:"note"`
}

type Order struct {
	ID               string             `json:"id"`
	ListingID        string             `json:"listingId"`
	FarmerID         string             `json:"farmerId"`
	ProcessorID      string             `json:"processorId"`
	Liters           int64              `json:"liters"`
	PricePerLiterUZS int64              `json:"pricePerLiterUzs"`
	TotalUZS         int64              `json:"totalUzs"`
	Status           string             `json:"status"`
	ExpectedPickupAt time.Time          `json:"expectedPickupAt"`
	Note             *string            `json:"note,omitempty"`
	QualityOutcome   *string            `json:"qualityOutcome,omitempty"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	CompletedAt      *time.Time         `json:"completedAt,omitempty"`
	History          []OrderHistoryItem `json:"history,omitempty"`
}

type OrderHistoryItem struct {
	Status    string    `json:"status"`
	ActorID   string    `json:"actorId"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type QualityReportInput struct {
	Measurements map[string]float64 `json:"measurements"`
	Outcome      string             `json:"outcome"`
	Note         string             `json:"note"`
}

func RegisterMarketplaceRoutes(router chi.Router, service *Service) {
	api := &handler{service: service, logger: service.logger}
	router.Get("/listings", api.withMarketplace(api.listListings, "farmer", "processor"))
	router.Post("/listings", api.withMarketplace(api.createListing, "farmer"))
	router.Get("/listings/{listingId}", api.withMarketplace(api.getListing, "farmer", "processor"))
	router.Patch("/listings/{listingId}", api.withMarketplace(api.updateListing, "farmer"))
	router.Delete("/listings/{listingId}", api.withMarketplace(api.archiveListing, "farmer"))
	router.Get("/orders", api.withMarketplace(api.listOrders, "farmer", "processor"))
	router.Post("/orders", api.withMarketplace(api.createOrder, "processor"))
	router.Get("/orders/{orderId}", api.withMarketplace(api.getOrder, "farmer", "processor"))
	router.Post("/orders/{orderId}/accept", api.withMarketplace(api.acceptOrder, "farmer"))
	router.Post("/orders/{orderId}/reject", api.withMarketplace(api.rejectOrder, "farmer"))
	router.Post("/orders/{orderId}/cancel", api.withMarketplace(api.cancelOrder, "processor"))
	router.Post("/orders/{orderId}/status", api.withMarketplace(api.transitionOrder, "farmer", "processor"))
	router.Post("/orders/{orderId}/disputes", api.withMarketplace(api.openDispute, "farmer", "processor"))
	router.Post("/orders/{orderId}/quality-reports", api.withMarketplace(api.createQualityReport, "farmer", "processor"))
	router.Get("/dashboards/farmer", api.withMarketplace(api.farmerDashboard, "farmer"))
	router.Get("/dashboards/processor", api.withMarketplace(api.processorDashboard, "processor"))
	router.Get("/dashboards/admin", api.withAuth(api.adminDashboard, true, false))
}

func (api *handler) withMarketplace(next func(http.ResponseWriter, *http.Request, Principal), roles ...string) http.HandlerFunc {
	return api.withAuth(func(w http.ResponseWriter, r *http.Request, principal Principal) {
		switch principal.AccountStatus {
		case "pending":
			api.writeServiceError(w, r, ErrAccountUnapproved)
			return
		case "rejected":
			api.writeServiceError(w, r, ErrAccountRejected)
			return
		case "suspended":
			api.writeServiceError(w, r, ErrAccountSuspended)
			return
		case "approved":
		default:
			api.writeServiceError(w, r, ErrForbidden)
			return
		}
		allowed := false
		for _, role := range roles {
			if principal.Role == role {
				allowed = true
				break
			}
		}
		if !allowed {
			api.writeServiceError(w, r, ErrForbidden)
			return
		}
		next(w, r, principal)
	}, false, false)
}

func (api *handler) listListings(w http.ResponseWriter, r *http.Request, principal Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	minLiters := int64(0)
	if raw := r.URL.Query().Get("minLiters"); raw != "" {
		minLiters, err = parsePositiveInt64(raw)
		if err != nil {
			api.writeServiceError(w, r, ErrInvalidQuery)
			return
		}
	}
	result, err := api.service.ListListings(r.Context(), principal, strings.TrimSpace(r.URL.Query().Get("regionCode")),
		minLiters, r.URL.Query().Get("sort"), pageNumber, pageSize)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) createListing(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input ListingInput
	if !decodeBody(w, r, &input) {
		return
	}
	listing, err := api.service.CreateListing(r.Context(), principal, input, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]Listing{"listing": listing})
}

func (api *handler) getListing(w http.ResponseWriter, r *http.Request, principal Principal) {
	listing, err := api.service.GetListing(r.Context(), principal, chi.URLParam(r, "listingId"))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (api *handler) updateListing(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input ListingUpdate
	if !decodeBody(w, r, &input) {
		return
	}
	listing, err := api.service.UpdateListing(r.Context(), principal, chi.URLParam(r, "listingId"), input, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (api *handler) archiveListing(w http.ResponseWriter, r *http.Request, principal Principal) {
	if err := api.service.ArchiveListing(r.Context(), principal, chi.URLParam(r, "listingId"), requestID(r)); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *handler) listOrders(w http.ResponseWriter, r *http.Request, principal Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	result, err := api.service.ListOrders(r.Context(), principal, r.URL.Query().Get("status"), pageNumber, pageSize)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) createOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input OrderInput
	if !decodeBody(w, r, &input) {
		return
	}
	order, err := api.service.CreateOrder(r.Context(), principal, input, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]Order{"order": order})
}

func (api *handler) getOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	order, err := api.service.GetOrder(r.Context(), principal, chi.URLParam(r, "orderId"))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (api *handler) acceptOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	api.orderDecision(w, r, principal, "accepted")
}

func (api *handler) rejectOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	api.orderDecision(w, r, principal, "rejected")
}

func (api *handler) cancelOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	api.orderDecision(w, r, principal, "cancelled")
}

func (api *handler) orderDecision(w http.ResponseWriter, r *http.Request, principal Principal, status string) {
	var input struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 && !decodeBody(w, r, &input) {
		return
	}
	order, err := api.service.TransitionOrder(r.Context(), principal, chi.URLParam(r, "orderId"), status, input.Reason, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (api *handler) transitionOrder(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	order, err := api.service.TransitionOrder(r.Context(), principal, chi.URLParam(r, "orderId"), input.Status, input.Note, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (api *handler) openDispute(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if err := api.service.OpenDispute(r.Context(), principal, chi.URLParam(r, "orderId"), input.Reason, requestID(r)); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "disputed"})
}

func (api *handler) createQualityReport(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input QualityReportInput
	if !decodeBody(w, r, &input) {
		return
	}
	if err := api.service.CreateQualityReport(r.Context(), principal, chi.URLParam(r, "orderId"), input, requestID(r)); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

func (api *handler) farmerDashboard(w http.ResponseWriter, r *http.Request, principal Principal) {
	result, err := api.service.FarmerDashboard(r.Context(), principal, r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) processorDashboard(w http.ResponseWriter, r *http.Request, principal Principal) {
	result, err := api.service.ProcessorDashboard(r.Context(), principal, r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) adminDashboard(w http.ResponseWriter, r *http.Request, principal Principal) {
	result, err := api.service.AdminDashboard(r.Context(), r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parsePositiveInt64(raw string) (int64, error) {
	var value int64
	if _, err := fmt.Sscan(raw, &value); err != nil || value < 1 || fmt.Sprint(value) != raw {
		return 0, ErrInvalidQuery
	}
	return value, nil
}

func (service *Service) ListListings(ctx context.Context, principal Principal, regionCode string, minLiters int64, sort string, pageNumber, pageSize int) (any, error) {
	if len(regionCode) > 32 {
		return nil, ErrInvalidQuery
	}
	switch sort {
	case "", "newest", "oldest", "price_asc", "price_desc":
	default:
		return nil, ErrInvalidQuery
	}
	var total int64
	const where = `($2='' OR region_code=$2) AND (available_liters-reserved_liters >= $3)`
	queryFilter := `farmer_id=$1 AND ` + where
	if principal.Role == "processor" {
		queryFilter = `status='active' AND available_until>now() AND available_from<=now()
			AND available_liters>reserved_liters AND ` + where
	}
	if err := service.pool.QueryRow(ctx, `SELECT count(*) FROM milk_listings WHERE `+queryFilter,
		principal.ID, regionCode, minLiters).Scan(&total); err != nil {
		return nil, fmt.Errorf("count visible listings: %w", err)
	}
	query := `
		SELECT id::text,farmer_id::text,region_code,available_liters-reserved_liters,
		       fat_percent::double precision,price_per_liter_uzs,available_from,available_until,
		       status,created_at,updated_at
		FROM milk_listings WHERE ` + queryFilter + `
		ORDER BY
			CASE WHEN $4='oldest' THEN created_at END ASC,
			CASE WHEN $4='newest' OR $4='' THEN created_at END DESC,
			CASE WHEN $4='price_asc' THEN price_per_liter_uzs END ASC,
			CASE WHEN $4='price_desc' THEN price_per_liter_uzs END DESC,
			id LIMIT $5 OFFSET $6`
	rows, err := service.pool.Query(ctx, query, principal.ID, regionCode, minLiters, sort, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list visible listings: %w", err)
	}
	defer rows.Close()
	items := make([]Listing, 0)
	for rows.Next() {
		listing, err := scanListing(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		items = append(items, listing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listings: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

func (service *Service) GetListing(ctx context.Context, principal Principal, listingID string) (Listing, error) {
	if !isUUID(listingID) {
		return Listing{}, ErrInvalidQuery
	}
	var listing Listing
	err := service.pool.QueryRow(ctx, listingSelect+`
		WHERE id=$1 AND (farmer_id=$2 OR ($3='processor' AND status='active' AND available_until>now()))`,
		listingID, principal.ID, principal.Role).Scan(
		&listing.ID, &listing.FarmerID, &listing.RegionCode, &listing.AvailableLiters, &listing.FatPercent,
		&listing.PricePerLiterUZS, &listing.AvailableFrom, &listing.AvailableUntil, &listing.Status,
		&listing.CreatedAt, &listing.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Listing{}, ErrNotFound
	}
	if err != nil {
		return Listing{}, fmt.Errorf("load listing: %w", err)
	}
	return listing, nil
}

const listingSelect = `
	SELECT id::text,farmer_id::text,region_code,available_liters-reserved_liters,
	       fat_percent::double precision,price_per_liter_uzs,available_from,available_until,
	       status,created_at,updated_at FROM milk_listings`

func scanListing(row scanner) (Listing, error) {
	var listing Listing
	err := row.Scan(&listing.ID, &listing.FarmerID, &listing.RegionCode, &listing.AvailableLiters,
		&listing.FatPercent, &listing.PricePerLiterUZS, &listing.AvailableFrom, &listing.AvailableUntil,
		&listing.Status, &listing.CreatedAt, &listing.UpdatedAt)
	return listing, err
}

func (service *Service) CreateListing(ctx context.Context, principal Principal, input ListingInput, requestID string) (Listing, error) {
	if err := validateListingInput(input); err != nil {
		return Listing{}, err
	}
	id, err := newUUID()
	if err != nil {
		return Listing{}, err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Listing{}, fmt.Errorf("begin listing creation: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		INSERT INTO milk_listings
			(id,farmer_id,region_code,available_liters,fat_percent,price_per_liter_uzs,available_from,available_until,status)
		SELECT $1,$2,r.code,$3,$4,$5,$6,$7,'active'
		FROM regions r WHERE r.code=$8 AND r.active`,
		id, principal.ID, input.AvailableLiters, input.FatPercent, input.PricePerLiterUZS,
		input.AvailableFrom.UTC(), input.AvailableUntil.UTC(), input.RegionCode)
	if err != nil {
		return Listing{}, fmt.Errorf("insert listing: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return Listing{}, ErrInvalidInput
	}
	listing, err := queryListingTx(ctx, tx, id)
	if err != nil {
		return Listing{}, err
	}
	if err := insertAudit(ctx, tx, principal.ID, "listing.created", "listing", id, requestID, nil); err != nil {
		return Listing{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Listing{}, fmt.Errorf("commit listing creation: %w", err)
	}
	return listing, nil
}

func validateListingInput(input ListingInput) error {
	now := time.Now()
	if input.AvailableLiters < 1 || input.AvailableLiters > 1_000_000_000 ||
		input.PricePerLiterUZS < 1 || input.PricePerLiterUZS > 1_000_000_000_000 ||
		math.IsNaN(input.FatPercent) || math.IsInf(input.FatPercent, 0) ||
		input.FatPercent < 0 || input.FatPercent > 100 ||
		!validText(input.RegionCode, 1, 32) ||
		input.AvailableFrom.IsZero() || input.AvailableUntil.IsZero() ||
		!input.AvailableUntil.After(input.AvailableFrom) ||
		input.AvailableFrom.Before(now.Add(-time.Hour)) ||
		input.AvailableUntil.After(now.AddDate(1, 0, 0)) {
		return ErrInvalidInput
	}
	return nil
}

func validateListingUpdate(regionCode string, liters int64, fatPercent float64, price int64, from, until time.Time) error {
	now := time.Now()
	if liters < 1 || liters > 1_000_000_000 || price < 1 || price > 1_000_000_000_000 ||
		math.IsNaN(fatPercent) || math.IsInf(fatPercent, 0) || fatPercent < 0 || fatPercent > 100 ||
		!validText(regionCode, 1, 32) || from.IsZero() || until.IsZero() ||
		!until.After(from) || !until.After(now) || until.After(now.AddDate(1, 0, 0)) {
		return ErrInvalidInput
	}
	return nil
}

func queryListingTx(ctx context.Context, tx pgx.Tx, id string) (Listing, error) {
	var listing Listing
	err := tx.QueryRow(ctx, listingSelect+` WHERE id=$1`, id).Scan(
		&listing.ID, &listing.FarmerID, &listing.RegionCode, &listing.AvailableLiters, &listing.FatPercent,
		&listing.PricePerLiterUZS, &listing.AvailableFrom, &listing.AvailableUntil, &listing.Status,
		&listing.CreatedAt, &listing.UpdatedAt)
	if err != nil {
		return Listing{}, fmt.Errorf("load listing after mutation: %w", err)
	}
	return listing, nil
}

func (service *Service) UpdateListing(ctx context.Context, principal Principal, listingID string, input ListingUpdate, requestID string) (Listing, error) {
	if !isUUID(listingID) {
		return Listing{}, ErrInvalidQuery
	}
	if input.AvailableLiters == nil && input.FatPercent == nil && input.PricePerLiterUZS == nil &&
		input.AvailableFrom == nil && input.AvailableUntil == nil && input.Status == nil {
		return Listing{}, ErrInvalidInput
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Listing{}, fmt.Errorf("begin listing update: %w", err)
	}
	defer tx.Rollback(ctx)
	var existing Listing
	var availableTotal, reserved int64
	err = tx.QueryRow(ctx, `
		SELECT id::text,farmer_id::text,region_code,available_liters,reserved_liters,
		       fat_percent::double precision,price_per_liter_uzs,available_from,available_until,
		       status,created_at,updated_at
		FROM milk_listings WHERE id=$1 AND farmer_id=$2 FOR UPDATE`, listingID, principal.ID).Scan(
		&existing.ID, &existing.FarmerID, &existing.RegionCode, &availableTotal, &reserved,
		&existing.FatPercent, &existing.PricePerLiterUZS, &existing.AvailableFrom, &existing.AvailableUntil,
		&existing.Status, &existing.CreatedAt, &existing.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Listing{}, ErrNotFound
	}
	if err != nil {
		return Listing{}, fmt.Errorf("lock listing for update: %w", err)
	}
	if existing.Status == "archived" || existing.Status == "expired" || existing.Status == "sold_out" {
		return Listing{}, ErrConflict
	}
	if input.AvailableLiters != nil {
		if *input.AvailableLiters < 1 || *input.AvailableLiters > 1_000_000_000 {
			return Listing{}, ErrInvalidInput
		}
		availableTotal = reserved + *input.AvailableLiters
	}
	if input.FatPercent != nil {
		if math.IsNaN(*input.FatPercent) || math.IsInf(*input.FatPercent, 0) || *input.FatPercent < 0 || *input.FatPercent > 100 {
			return Listing{}, ErrInvalidInput
		}
		existing.FatPercent = *input.FatPercent
	}
	if input.PricePerLiterUZS != nil {
		if *input.PricePerLiterUZS < 1 || *input.PricePerLiterUZS > 1_000_000_000_000 {
			return Listing{}, ErrInvalidInput
		}
		existing.PricePerLiterUZS = *input.PricePerLiterUZS
	}
	if input.AvailableFrom != nil {
		existing.AvailableFrom = *input.AvailableFrom
	}
	if input.AvailableUntil != nil {
		existing.AvailableUntil = *input.AvailableUntil
	}
	if err := validateListingUpdate(existing.RegionCode, availableTotal-reserved, existing.FatPercent,
		existing.PricePerLiterUZS, existing.AvailableFrom, existing.AvailableUntil); err != nil {
		return Listing{}, err
	}
	status := existing.Status
	if input.Status != nil {
		if *input.Status != "active" && *input.Status != "paused" {
			return Listing{}, ErrInvalidInput
		}
		status = *input.Status
	}
	if status == "active" && availableTotal <= reserved {
		return Listing{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		UPDATE milk_listings SET available_liters=$3,fat_percent=$4,price_per_liter_uzs=$5,
			available_from=$6,available_until=$7,status=$8,updated_at=now()
		WHERE id=$1 AND farmer_id=$2`,
		listingID, principal.ID, availableTotal, existing.FatPercent, existing.PricePerLiterUZS,
		existing.AvailableFrom.UTC(), existing.AvailableUntil.UTC(), status); err != nil {
		return Listing{}, fmt.Errorf("update listing: %w", err)
	}
	if err := insertAudit(ctx, tx, principal.ID, "listing.updated", "listing", listingID, requestID, nil); err != nil {
		return Listing{}, err
	}
	listing, err := queryListingTx(ctx, tx, listingID)
	if err != nil {
		return Listing{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Listing{}, fmt.Errorf("commit listing update: %w", err)
	}
	return listing, nil
}

func (service *Service) ArchiveListing(ctx context.Context, principal Principal, listingID, requestID string) error {
	if !isUUID(listingID) {
		return ErrInvalidQuery
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin listing archive: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	var reserved int64
	err = tx.QueryRow(ctx, `
		SELECT status,reserved_liters FROM milk_listings
		WHERE id=$1 AND farmer_id=$2 FOR UPDATE`, listingID, principal.ID).Scan(&status, &reserved)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock listing for archive: %w", err)
	}
	if status == "archived" || status == "expired" || status == "sold_out" {
		return ErrConflict
	}
	if reserved > 0 {
		return ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE milk_listings SET status='archived',updated_at=now() WHERE id=$1`, listingID); err != nil {
		return fmt.Errorf("archive listing: %w", err)
	}
	if err := insertAudit(ctx, tx, principal.ID, "listing.archived", "listing", listingID, requestID, nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit listing archive: %w", err)
	}
	return nil
}

func (service *Service) CreateOrder(ctx context.Context, principal Principal, input OrderInput, requestID string) (Order, error) {
	input.Note = strings.TrimSpace(input.Note)
	if !isUUID(input.ListingID) || input.Liters < 1 || input.Liters > 1_000_000_000 ||
		len(input.Note) > 2000 || input.ExpectedPickupAt.IsZero() ||
		!input.ExpectedPickupAt.After(time.Now()) {
		return Order{}, ErrInvalidInput
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin order placement: %w", err)
	}
	defer tx.Rollback(ctx)
	var farmerID, status string
	var available, reserved, price int64
	var availableFrom, availableUntil time.Time
	err = tx.QueryRow(ctx, `
		SELECT farmer_id::text,status,available_liters,reserved_liters,price_per_liter_uzs,available_from,available_until
		FROM milk_listings WHERE id=$1 FOR UPDATE`, input.ListingID).Scan(
		&farmerID, &status, &available, &reserved, &price, &availableFrom, &availableUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock listing for order placement: %w", err)
	}
	if farmerID == principal.ID {
		return Order{}, ErrForbidden
	}
	participantRows, err := tx.Query(ctx, `
		SELECT id::text FROM users WHERE id IN ($1,$2) ORDER BY id FOR UPDATE`,
		farmerID, principal.ID)
	if err != nil {
		return Order{}, fmt.Errorf("lock order participants: %w", err)
	}
	var lockedParticipants int
	for participantRows.Next() {
		var participantID string
		if err := participantRows.Scan(&participantID); err != nil {
			participantRows.Close()
			return Order{}, fmt.Errorf("scan locked order participant: %w", err)
		}
		lockedParticipants++
	}
	if err := participantRows.Err(); err != nil {
		participantRows.Close()
		return Order{}, fmt.Errorf("iterate locked order participants: %w", err)
	}
	participantRows.Close()
	if lockedParticipants != 2 {
		return Order{}, ErrNotFound
	}
	if status != "active" || availableUntil.Before(time.Now()) || availableFrom.After(time.Now()) ||
		input.ExpectedPickupAt.Before(availableFrom) || input.ExpectedPickupAt.After(availableUntil) {
		return Order{}, ErrConflict
	}
	if input.Liters > available-reserved {
		return Order{}, ErrConflict
	}
	if input.Liters > 9_223_372_036_854_775_807/price {
		return Order{}, ErrInvalidInput
	}
	orderID, err := newUUID()
	if err != nil {
		return Order{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE milk_listings SET reserved_liters=reserved_liters+$2,updated_at=now() WHERE id=$1`,
		input.ListingID, input.Liters); err != nil {
		return Order{}, fmt.Errorf("reserve listing inventory: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO orders
			(id,listing_id,farmer_id,processor_id,liters,price_per_liter_uzs,total_uzs,status,expected_pickup_at,note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'placed',$8,NULLIF($9,''))`,
		orderID, input.ListingID, farmerID, principal.ID, input.Liters, price, input.Liters*price,
		input.ExpectedPickupAt.UTC(), input.Note)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	if err := insertOrderHistory(ctx, tx, orderID, principal.ID, "", "placed", input.Note); err != nil {
		return Order{}, err
	}
	if err := insertAudit(ctx, tx, principal.ID, "order.placed", "order", orderID, requestID, map[string]any{
		"listingId": input.ListingID, "liters": input.Liters, "totalUzs": input.Liters * price,
	}); err != nil {
		return Order{}, err
	}
	order, err := queryOrderTx(ctx, tx, orderID, true)
	if err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit order placement: %w", err)
	}
	return order, nil
}

func (service *Service) ListOrders(ctx context.Context, principal Principal, status string, pageNumber, pageSize int) (any, error) {
	if status != "" && !validOrderStatus(status) {
		return nil, ErrInvalidQuery
	}
	var total int64
	const filter = `(($2='farmer' AND farmer_id=$1) OR ($2='processor' AND processor_id=$1)) AND ($3='' OR status=$3)`
	if err := service.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE `+filter,
		principal.ID, principal.Role, status).Scan(&total); err != nil {
		return nil, fmt.Errorf("count caller orders: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT `+orderColumns+` FROM orders WHERE `+filter+`
		ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`,
		principal.ID, principal.Role, status, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list caller orders: %w", err)
	}
	defer rows.Close()
	items := make([]Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		items = append(items, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate caller orders: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

const orderColumns = `id::text,listing_id::text,farmer_id::text,processor_id::text,liters,
	price_per_liter_uzs,total_uzs,status,expected_pickup_at,note,quality_outcome,created_at,updated_at,completed_at`

func scanOrder(row scanner) (Order, error) {
	var order Order
	err := row.Scan(&order.ID, &order.ListingID, &order.FarmerID, &order.ProcessorID, &order.Liters,
		&order.PricePerLiterUZS, &order.TotalUZS, &order.Status, &order.ExpectedPickupAt, &order.Note,
		&order.QualityOutcome, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt)
	return order, err
}

func queryOrderTx(ctx context.Context, tx pgx.Tx, orderID string, withHistory bool) (Order, error) {
	order, err := scanOrder(tx.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1`, orderID))
	if err != nil {
		return Order{}, fmt.Errorf("load order after mutation: %w", err)
	}
	if withHistory {
		rows, err := tx.Query(ctx, `
			SELECT new_status,actor_id::text,note,created_at
			FROM order_status_history WHERE order_id=$1 ORDER BY created_at,id`, orderID)
		if err != nil {
			return Order{}, fmt.Errorf("load order history: %w", err)
		}
		order.History = make([]OrderHistoryItem, 0)
		for rows.Next() {
			var item OrderHistoryItem
			if err := rows.Scan(&item.Status, &item.ActorID, &item.Note, &item.CreatedAt); err != nil {
				rows.Close()
				return Order{}, fmt.Errorf("scan order history: %w", err)
			}
			order.History = append(order.History, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return Order{}, fmt.Errorf("iterate order history: %w", err)
		}
		rows.Close()
	}
	return order, nil
}

func (service *Service) GetOrder(ctx context.Context, principal Principal, orderID string) (Order, error) {
	if !isUUID(orderID) {
		return Order{}, ErrInvalidQuery
	}
	order, err := scanOrder(service.pool.QueryRow(ctx, `
		SELECT `+orderColumns+` FROM orders WHERE id=$1
		  AND (($2='farmer' AND farmer_id=$3) OR ($2='processor' AND processor_id=$3))`,
		orderID, principal.Role, principal.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("load caller order: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT new_status,actor_id::text,note,created_at
		FROM order_status_history WHERE order_id=$1 ORDER BY created_at,id`, orderID)
	if err != nil {
		return Order{}, fmt.Errorf("load order history: %w", err)
	}
	defer rows.Close()
	order.History = make([]OrderHistoryItem, 0)
	for rows.Next() {
		var item OrderHistoryItem
		if err := rows.Scan(&item.Status, &item.ActorID, &item.Note, &item.CreatedAt); err != nil {
			return Order{}, fmt.Errorf("scan order history: %w", err)
		}
		order.History = append(order.History, item)
	}
	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("iterate order history: %w", err)
	}
	return order, nil
}

func (service *Service) TransitionOrder(ctx context.Context, principal Principal, orderID, nextStatus, note, requestID string) (Order, error) {
	if !isUUID(orderID) || len(note) > 2000 {
		if !isUUID(orderID) {
			return Order{}, ErrInvalidQuery
		}
		return Order{}, ErrInvalidInput
	}
	note = strings.TrimSpace(note)
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin order transition: %w", err)
	}
	defer tx.Rollback(ctx)
	var order Order
	err = tx.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(
		&order.ID, &order.ListingID, &order.FarmerID, &order.ProcessorID, &order.Liters,
		&order.PricePerLiterUZS, &order.TotalUZS, &order.Status, &order.ExpectedPickupAt,
		&order.Note, &order.QualityOutcome, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock order for transition: %w", err)
	}
	if !isParty(principal, order) {
		return Order{}, ErrNotFound
	}
	var allowed bool
	switch nextStatus {
	case "accepted":
		allowed = principal.Role == "farmer" && order.Status == "placed"
	case "rejected":
		allowed = principal.Role == "farmer" && order.Status == "placed"
	case "cancelled":
		allowed = principal.Role == "processor" && order.Status == "placed"
	case "ready_for_pickup":
		allowed = principal.Role == "farmer" && order.Status == "accepted"
	case "picked_up":
		allowed = principal.Role == "farmer" && order.Status == "ready_for_pickup"
	case "delivered":
		allowed = principal.Role == "farmer" && order.Status == "picked_up"
	case "completed":
		allowed = principal.Role == "processor" && order.Status == "delivered"
	default:
		return Order{}, ErrInvalidInput
	}
	if !allowed {
		return Order{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status=$2,updated_at=now(),completed_at=CASE WHEN $2='completed' THEN now() ELSE completed_at END WHERE id=$1`,
		orderID, nextStatus); err != nil {
		return Order{}, fmt.Errorf("update order status: %w", err)
	}
	if nextStatus == "rejected" || nextStatus == "cancelled" {
		if _, err := tx.Exec(ctx, `
			UPDATE milk_listings SET reserved_liters=reserved_liters-$2,updated_at=now() WHERE id=$1`,
			order.ListingID, order.Liters); err != nil {
			return Order{}, fmt.Errorf("release order inventory: %w", err)
		}
	}
	if nextStatus == "completed" {
		if _, err := tx.Exec(ctx, `
			UPDATE milk_listings
			SET available_liters=available_liters-$2,reserved_liters=reserved_liters-$2,
			    status=CASE WHEN available_liters-$2=0 THEN 'sold_out' ELSE status END,updated_at=now()
			WHERE id=$1`, order.ListingID, order.Liters); err != nil {
			return Order{}, fmt.Errorf("consume completed order inventory: %w", err)
		}
	}
	if err := insertOrderHistory(ctx, tx, orderID, principal.ID, order.Status, nextStatus, note); err != nil {
		return Order{}, err
	}
	if err := insertAudit(ctx, tx, principal.ID, "order."+nextStatus, "order", orderID, requestID, map[string]any{"note": note}); err != nil {
		return Order{}, err
	}
	result, err := queryOrderTx(ctx, tx, orderID, true)
	if err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit order transition: %w", err)
	}
	return result, nil
}

func (service *Service) OpenDispute(ctx context.Context, principal Principal, orderID, reason, requestID string) error {
	reason = strings.TrimSpace(reason)
	if !isUUID(orderID) || !validText(reason, 1, 2000) {
		if !isUUID(orderID) {
			return ErrInvalidQuery
		}
		return ErrInvalidInput
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dispute creation: %w", err)
	}
	defer tx.Rollback(ctx)
	var order Order
	err = tx.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(
		&order.ID, &order.ListingID, &order.FarmerID, &order.ProcessorID, &order.Liters,
		&order.PricePerLiterUZS, &order.TotalUZS, &order.Status, &order.ExpectedPickupAt,
		&order.Note, &order.QualityOutcome, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock order for dispute: %w", err)
	}
	if !isParty(principal, order) {
		return ErrNotFound
	}
	if order.Status == "completed" || order.Status == "rejected" || order.Status == "cancelled" || order.Status == "disputed" {
		return ErrConflict
	}
	disputeID, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO order_disputes (id,order_id,opened_by,reason) VALUES ($1,$2,$3,$4)`,
		disputeID, orderID, principal.ID, reason); err != nil {
		return fmt.Errorf("create order dispute: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status='disputed',updated_at=now() WHERE id=$1`, orderID); err != nil {
		return fmt.Errorf("mark order disputed: %w", err)
	}
	if err := insertOrderHistory(ctx, tx, orderID, principal.ID, order.Status, "disputed", reason); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, principal.ID, "order.disputed", "order", orderID, requestID, map[string]any{"reason": reason}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit dispute creation: %w", err)
	}
	return nil
}

func (service *Service) CreateQualityReport(ctx context.Context, principal Principal, orderID string, input QualityReportInput, requestID string) error {
	input.Note = strings.TrimSpace(input.Note)
	if !isUUID(orderID) || (input.Outcome != "accepted" && input.Outcome != "rejected") ||
		len(input.Measurements) == 0 || len(input.Measurements) > 16 || len(input.Note) > 2000 {
		if !isUUID(orderID) {
			return ErrInvalidQuery
		}
		return ErrInvalidInput
	}
	for metric, value := range input.Measurements {
		if !validText(metric, 1, 32) || math.IsNaN(value) || math.IsInf(value, 0) || value < -1000 || value > 1000 {
			return ErrInvalidInput
		}
	}
	measurements, err := json.Marshal(input.Measurements)
	if err != nil {
		return fmt.Errorf("encode quality measurements: %w", err)
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin quality report: %w", err)
	}
	defer tx.Rollback(ctx)
	var order Order
	err = tx.QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1 FOR UPDATE`, orderID).Scan(
		&order.ID, &order.ListingID, &order.FarmerID, &order.ProcessorID, &order.Liters,
		&order.PricePerLiterUZS, &order.TotalUZS, &order.Status, &order.ExpectedPickupAt,
		&order.Note, &order.QualityOutcome, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock order for quality report: %w", err)
	}
	if !isParty(principal, order) {
		return ErrNotFound
	}
	if order.Status != "accepted" && order.Status != "ready_for_pickup" && order.Status != "picked_up" && order.Status != "delivered" {
		return ErrConflict
	}
	reportID, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO quality_reports (id,order_id,reporter_id,measurements,outcome,note)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''))`,
		reportID, orderID, principal.ID, measurements, input.Outcome, input.Note); err != nil {
		return fmt.Errorf("insert quality report: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET quality_outcome=$2,updated_at=now() WHERE id=$1`, orderID, input.Outcome); err != nil {
		return fmt.Errorf("update order quality outcome: %w", err)
	}
	if err := insertAudit(ctx, tx, principal.ID, "order.quality_reported", "order", orderID, requestID,
		map[string]any{"outcome": input.Outcome, "measurements": input.Measurements}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit quality report: %w", err)
	}
	return nil
}

func isParty(principal Principal, order Order) bool {
	return principal.Role == "farmer" && principal.ID == order.FarmerID ||
		principal.Role == "processor" && principal.ID == order.ProcessorID
}

func validOrderStatus(status string) bool {
	switch status {
	case "placed", "accepted", "rejected", "cancelled", "ready_for_pickup", "picked_up", "delivered", "completed", "disputed":
		return true
	default:
		return false
	}
}

func insertOrderHistory(ctx context.Context, tx pgx.Tx, orderID, actorID, oldStatus, nextStatus, note string) error {
	id, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_status_history (id,order_id,actor_id,old_status,new_status,note)
		VALUES ($1,$2,$3,NULLIF($4,''),$5,NULLIF($6,''))`,
		id, orderID, actorID, oldStatus, nextStatus, note); err != nil {
		return fmt.Errorf("append order history: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, actorID, action, targetType, targetID, requestID string, metadata any) error {
	id, err := newUUID()
	if err != nil {
		return err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id,actor_id,action,target_type,target_id,request_id,metadata)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7)`,
		id, actorID, action, targetType, targetID, requestID, encoded); err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}
	return nil
}

func (service *Service) dashboardPeriod(query url.Values) (time.Time, time.Time, string, error) {
	now := time.Now().UTC()
	fromRaw, toRaw := strings.TrimSpace(query.Get("from")), strings.TrimSpace(query.Get("to"))
	if fromRaw != "" || toRaw != "" {
		from, fromErr := time.Parse(time.RFC3339, fromRaw)
		to, toErr := time.Parse(time.RFC3339, toRaw)
		if fromErr != nil || toErr != nil || !to.After(from) || to.Sub(from) > 366*24*time.Hour || to.After(now.Add(time.Minute)) {
			return time.Time{}, time.Time{}, "", ErrInvalidQuery
		}
		bucket := "day"
		if to.Sub(from) > 90*24*time.Hour {
			bucket = "month"
		}
		return from.UTC(), to.UTC(), bucket, nil
	}
	switch query.Get("period") {
	case "", "30d":
		return now.AddDate(0, 0, -30), now, "day", nil
	case "7d":
		return now.AddDate(0, 0, -7), now, "day", nil
	case "90d":
		return now.AddDate(0, 0, -90), now, "day", nil
	case "12m":
		return now.AddDate(-1, 0, 0), now, "month", nil
	default:
		return time.Time{}, time.Time{}, "", ErrInvalidQuery
	}
}

func (service *Service) FarmerDashboard(ctx context.Context, principal Principal, query url.Values) (any, error) {
	from, to, bucket, err := service.dashboardPeriod(query)
	if err != nil {
		return nil, err
	}
	return service.partyDashboard(ctx, principal, from, to, bucket, true)
}

func (service *Service) ProcessorDashboard(ctx context.Context, principal Principal, query url.Values) (any, error) {
	from, to, bucket, err := service.dashboardPeriod(query)
	if err != nil {
		return nil, err
	}
	return service.partyDashboard(ctx, principal, from, to, bucket, false)
}

func (service *Service) AdminDashboard(ctx context.Context, query url.Values) (any, error) {
	from, to, bucket, err := service.dashboardPeriod(query)
	if err != nil {
		return nil, err
	}
	return service.adminDashboardData(ctx, from, to, bucket)
}

func (service *Service) partyDashboard(ctx context.Context, principal Principal, from, to time.Time, bucket string, farmer bool) (any, error) {
	roleColumn, partyID := "processor_id", principal.ID
	if farmer {
		roleColumn = "farmer_id"
	}
	var volume, value int64
	var awaitingCount, suppliers int64
	var ownAverage float64
	var ownAverageValid bool
	if err := service.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(liters),0),COALESCE(sum(total_uzs),0),count(DISTINCT farmer_id),
		       COALESCE(sum(total_uzs)::double precision/NULLIF(sum(liters),0),0),
		       COALESCE(sum(liters)>0,false)
		FROM orders WHERE `+roleColumn+`=$1 AND status='completed' AND completed_at >= $2 AND completed_at < $3`,
		partyID, from, to).Scan(&volume, &value, &suppliers, &ownAverage, &ownAverageValid); err != nil {
		return nil, fmt.Errorf("calculate party dashboard KPIs: %w", err)
	}
	var activeListings int64
	if farmer {
		if err := service.pool.QueryRow(ctx, `
			SELECT count(*) FROM orders WHERE farmer_id=$1 AND status='placed'`,
			partyID).Scan(&awaitingCount); err != nil {
			return nil, fmt.Errorf("count farmer orders awaiting response: %w", err)
		}
		if err := service.pool.QueryRow(ctx, `
			SELECT count(*) FROM milk_listings
			WHERE farmer_id=$1 AND status='active' AND available_liters>reserved_liters AND available_until>now()`,
			partyID).Scan(&activeListings); err != nil {
			return nil, fmt.Errorf("count farmer active listings: %w", err)
		}
	} else {
		if err := service.pool.QueryRow(ctx, `
			SELECT count(*) FROM orders WHERE processor_id=$1
			  AND status NOT IN ('completed','rejected','cancelled','disputed')`,
			partyID).Scan(&awaitingCount); err != nil {
			return nil, fmt.Errorf("count processor open orders: %w", err)
		}
	}
	var marketAverage float64
	var marketAverageValid bool
	if farmer {
		if err := service.pool.QueryRow(ctx, `
			SELECT COALESCE(sum(total_uzs)::double precision/NULLIF(sum(liters),0),0),
			       COALESCE(sum(liters)>0,false)
			FROM orders WHERE status='completed' AND completed_at >= $1 AND completed_at < $2`,
			from, to).Scan(&marketAverage, &marketAverageValid); err != nil {
			return nil, fmt.Errorf("calculate platform average price: %w", err)
		}
	}
	var qualityAccepted, qualityTotal int64
	if err := service.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE quality_outcome='accepted'),count(*)
		FROM orders WHERE `+roleColumn+`=$1 AND status='completed'
		  AND completed_at >= $2 AND completed_at < $3 AND quality_outcome IS NOT NULL`,
		partyID, from, to).Scan(&qualityAccepted, &qualityTotal); err != nil {
		return nil, fmt.Errorf("calculate quality acceptance rate: %w", err)
	}
	chartRows, err := service.pool.Query(ctx, `
		SELECT date_trunc($4,completed_at) AS bucket,COALESCE(sum(liters),0),COALESCE(sum(total_uzs),0),count(*)
		FROM orders WHERE `+roleColumn+`=$1 AND status='completed'
		  AND completed_at >= $2 AND completed_at < $3
		GROUP BY 1 ORDER BY 1`, partyID, from, to, bucket)
	if err != nil {
		return nil, fmt.Errorf("load dashboard period chart: %w", err)
	}
	volumeSeries := make([]map[string]any, 0)
	for chartRows.Next() {
		var date time.Time
		var liters, totalUZS, count int64
		if err := chartRows.Scan(&date, &liters, &totalUZS, &count); err != nil {
			chartRows.Close()
			return nil, fmt.Errorf("scan dashboard period chart: %w", err)
		}
		volumeSeries = append(volumeSeries, map[string]any{"period": date.UTC(), "liters": liters, "totalUzs": totalUZS, "orders": count})
	}
	if err := chartRows.Err(); err != nil {
		chartRows.Close()
		return nil, fmt.Errorf("iterate dashboard period chart: %w", err)
	}
	chartRows.Close()

	var marketPrice any
	if marketAverageValid {
		marketPrice = marketAverage
	}
	var ownPrice any
	if ownAverageValid {
		ownPrice = ownAverage
	}
	kpis := map[string]any{}
	if farmer {
		kpis = map[string]any{
			"milkSoldLiters":               map[string]any{"value": volume, "unit": "liters"},
			"revenueUzs":                   map[string]any{"value": value, "unit": "UZS"},
			"activeListings":               map[string]any{"value": activeListings, "unit": "listings"},
			"ordersAwaitingResponse":       map[string]any{"value": awaitingCount, "unit": "orders"},
			"ownAveragePricePerLiter":      map[string]any{"value": ownPrice, "unit": "UZS/liter"},
			"platformAveragePricePerLiter": map[string]any{"value": marketPrice, "unit": "UZS/liter", "source": "completed platform orders"},
		}
	} else {
		kpis = map[string]any{
			"volumeSourcedLiters":   map[string]any{"value": volume, "unit": "liters"},
			"spendUzs":              map[string]any{"value": value, "unit": "UZS"},
			"activeSuppliers":       map[string]any{"value": suppliers, "unit": "suppliers"},
			"averagePricePerLiter":  map[string]any{"value": ownPrice, "unit": "UZS/liter"},
			"openOrders":            map[string]any{"value": awaitingCount, "unit": "orders"},
			"qualityAcceptanceRate": map[string]any{"value": ratio(qualityAccepted, qualityTotal), "numerator": qualityAccepted, "denominator": qualityTotal},
		}
	}
	upcoming, err := service.upcomingOrders(ctx, principal, farmer)
	if err != nil {
		return nil, err
	}
	tables := map[string]any{"upcomingOrders": upcoming}
	charts := map[string]any{"completedVolume": volumeSeries}
	if farmer {
		qualityTrend, err := service.farmerQualityTrend(ctx, partyID, from, to, bucket)
		if err != nil {
			return nil, err
		}
		activity, err := service.farmerRecentActivity(ctx, partyID)
		if err != nil {
			return nil, err
		}
		charts["qualityTrend"] = qualityTrend
		tables["recentActivity"] = activity
	} else {
		supply, err := service.processorSupplyByRegion(ctx, partyID, from, to)
		if err != nil {
			return nil, err
		}
		charts["supplyByRegion"] = supply
	}
	return map[string]any{
		"period": map[string]any{"from": from, "to": to, "bucket": bucket},
		"kpis":   kpis, "charts": charts, "tables": tables,
	}, nil
}

func (service *Service) upcomingOrders(ctx context.Context, principal Principal, farmer bool) ([]Order, error) {
	column := "processor_id"
	if farmer {
		column = "farmer_id"
	}
	rows, err := service.pool.Query(ctx, `
		SELECT `+orderColumns+` FROM orders WHERE `+column+`=$1
		  AND status IN ('accepted','ready_for_pickup','picked_up','delivered')
		  AND expected_pickup_at>=now()
		ORDER BY expected_pickup_at,id LIMIT 10`, principal.ID)
	if err != nil {
		return nil, fmt.Errorf("load upcoming orders: %w", err)
	}
	defer rows.Close()
	items := make([]Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan upcoming order: %w", err)
		}
		items = append(items, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate upcoming orders: %w", err)
	}
	return items, nil
}

func (service *Service) farmerQualityTrend(ctx context.Context, farmerID string, from, to time.Time, bucket string) ([]map[string]any, error) {
	rows, err := service.pool.Query(ctx, `
		SELECT date_trunc($4,qr.created_at),metric.key,avg(metric.value::numeric)
		FROM quality_reports qr JOIN orders o ON o.id=qr.order_id
		CROSS JOIN LATERAL jsonb_each_text(qr.measurements) AS metric(key,value)
		WHERE o.farmer_id=$1 AND qr.created_at >= $2 AND qr.created_at < $3
		GROUP BY 1,2 ORDER BY 1,2`, farmerID, from, to, bucket)
	if err != nil {
		return nil, fmt.Errorf("load farmer quality trend: %w", err)
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var date time.Time
		var metric string
		var average float64
		if err := rows.Scan(&date, &metric, &average); err != nil {
			return nil, fmt.Errorf("scan farmer quality trend: %w", err)
		}
		items = append(items, map[string]any{"period": date.UTC(), "metric": metric, "average": average})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate farmer quality trend: %w", err)
	}
	return items, nil
}

func (service *Service) farmerRecentActivity(ctx context.Context, farmerID string) ([]map[string]any, error) {
	rows, err := service.pool.Query(ctx, `
		(SELECT h.created_at,'order.'||h.new_status AS action,o.id::text AS target_id,h.actor_id::text AS actor_id,h.note
		 FROM order_status_history h JOIN orders o ON o.id=h.order_id WHERE o.farmer_id=$1)
		UNION ALL
		(SELECT l.updated_at,'listing.'||l.status AS action,l.id::text AS target_id,l.farmer_id::text AS actor_id,NULL::text AS note
		 FROM milk_listings l WHERE l.farmer_id=$1)
		ORDER BY created_at DESC LIMIT 10`, farmerID)
	if err != nil {
		return nil, fmt.Errorf("load farmer recent activity: %w", err)
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var createdAt time.Time
		var action, targetID, actorID string
		var note *string
		if err := rows.Scan(&createdAt, &action, &targetID, &actorID, &note); err != nil {
			return nil, fmt.Errorf("scan farmer recent activity: %w", err)
		}
		items = append(items, map[string]any{"createdAt": createdAt, "action": action, "targetId": targetID, "actorId": actorID, "note": note})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate farmer recent activity: %w", err)
	}
	return items, nil
}

func (service *Service) processorSupplyByRegion(ctx context.Context, processorID string, from, to time.Time) ([]map[string]any, error) {
	rows, err := service.pool.Query(ctx, `
		SELECT r.code,r.name_en,COALESCE(sum(o.liters),0)
		FROM orders o JOIN users farmer ON farmer.id=o.farmer_id
		JOIN regions r ON r.code=farmer.region_code
		WHERE o.processor_id=$1 AND o.status='completed' AND o.completed_at >= $2 AND o.completed_at < $3
		GROUP BY r.code,r.name_en ORDER BY sum(o.liters) DESC,r.code`, processorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load processor supply by region: %w", err)
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var code, name string
		var liters int64
		if err := rows.Scan(&code, &name, &liters); err != nil {
			return nil, fmt.Errorf("scan processor region supply: %w", err)
		}
		items = append(items, map[string]any{"regionCode": code, "regionName": name, "liters": liters})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate processor region supply: %w", err)
	}
	return items, nil
}

func ratio(numerator, denominator int64) any {
	if denominator == 0 {
		return nil
	}
	return float64(numerator) / float64(denominator)
}

func (service *Service) adminDashboardData(ctx context.Context, from, to time.Time, bucket string) (any, error) {
	var pending int64
	if err := service.pool.QueryRow(ctx, `
		SELECT count(*) FROM users
		WHERE role IN ('farmer','processor') AND account_status='pending' AND email_verified_at IS NOT NULL`).Scan(&pending); err != nil {
		return nil, fmt.Errorf("count applications awaiting review: %w", err)
	}
	var oldest User
	var oldestAt *time.Time
	err := service.pool.QueryRow(ctx, `
		SELECT id::text,role,account_status,email,(email_verified_at IS NOT NULL),
		       full_name,phone,organization_name,region_code,stir,rejection_reason,created_at,updated_at
		FROM users WHERE role IN ('farmer','processor')
		  AND account_status='pending' AND email_verified_at IS NOT NULL
		ORDER BY created_at,id LIMIT 1`).Scan(
		&oldest.ID, &oldest.Role, &oldest.AccountStatus, &oldest.Email, &oldest.EmailVerified,
		&oldest.FullName, &oldest.Phone, &oldest.OrganizationName, &oldest.RegionCode, &oldest.STIR,
		&oldest.RejectionReason, &oldest.CreatedAt, &oldest.UpdatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("load oldest pending applicant: %w", err)
	}
	if err == nil {
		oldestAt = &oldest.CreatedAt
	}
	roleRows, err := service.pool.Query(ctx, `SELECT role,account_status,count(*) FROM users GROUP BY role,account_status ORDER BY role,account_status`)
	if err != nil {
		return nil, fmt.Errorf("load account counts: %w", err)
	}
	accountsByRoleStatus := make([]map[string]any, 0)
	for roleRows.Next() {
		var role, status string
		var count int64
		if err := roleRows.Scan(&role, &status, &count); err != nil {
			roleRows.Close()
			return nil, fmt.Errorf("scan account counts: %w", err)
		}
		accountsByRoleStatus = append(accountsByRoleStatus, map[string]any{"role": role, "status": status, "count": count})
	}
	if err := roleRows.Err(); err != nil {
		roleRows.Close()
		return nil, fmt.Errorf("iterate account counts: %w", err)
	}
	roleRows.Close()
	signups, err := service.pool.Query(ctx, `
		SELECT date_trunc($1,created_at),count(*) FROM users
		WHERE created_at >= $2 AND created_at < $3 GROUP BY 1 ORDER BY 1`, bucket, from, to)
	if err != nil {
		return nil, fmt.Errorf("load signup chart: %w", err)
	}
	signupSeries := make([]map[string]any, 0)
	for signups.Next() {
		var date time.Time
		var count int64
		if err := signups.Scan(&date, &count); err != nil {
			signups.Close()
			return nil, fmt.Errorf("scan signup chart: %w", err)
		}
		signupSeries = append(signupSeries, map[string]any{"period": date.UTC(), "signups": count})
	}
	if err := signups.Err(); err != nil {
		signups.Close()
		return nil, fmt.Errorf("iterate signup chart: %w", err)
	}
	signups.Close()
	completed, err := service.pool.Query(ctx, `
		SELECT date_trunc($1,completed_at),COALESCE(sum(liters),0),COALESCE(sum(total_uzs),0)
		FROM orders WHERE status='completed' AND completed_at >= $2 AND completed_at < $3
		GROUP BY 1 ORDER BY 1`, bucket, from, to)
	if err != nil {
		return nil, fmt.Errorf("load platform volume chart: %w", err)
	}
	volumeSeries := make([]map[string]any, 0)
	for completed.Next() {
		var date time.Time
		var liters, totalUZS int64
		if err := completed.Scan(&date, &liters, &totalUZS); err != nil {
			completed.Close()
			return nil, fmt.Errorf("scan platform volume chart: %w", err)
		}
		volumeSeries = append(volumeSeries, map[string]any{"period": date.UTC(), "liters": liters, "totalUzs": totalUZS})
	}
	if err := completed.Err(); err != nil {
		completed.Close()
		return nil, fmt.Errorf("iterate platform volume chart: %w", err)
	}
	completed.Close()
	listings, err := service.pool.Query(ctx, `
		SELECT r.code,r.name_en,count(l.id),COALESCE(sum(l.available_liters-l.reserved_liters),0)
		FROM regions r LEFT JOIN milk_listings l ON l.region_code=r.code
		  AND l.status='active' AND l.available_until>now()
		GROUP BY r.code,r.name_en ORDER BY count(l.id) DESC,r.code`)
	if err != nil {
		return nil, fmt.Errorf("load active listings by region: %w", err)
	}
	activeListings := make([]map[string]any, 0)
	for listings.Next() {
		var code, name string
		var count, liters int64
		if err := listings.Scan(&code, &name, &count, &liters); err != nil {
			listings.Close()
			return nil, fmt.Errorf("scan active listings by region: %w", err)
		}
		activeListings = append(activeListings, map[string]any{"regionCode": code, "regionName": name, "listings": count, "availableLiters": liters})
	}
	if err := listings.Err(); err != nil {
		listings.Close()
		return nil, fmt.Errorf("iterate active listings by region: %w", err)
	}
	listings.Close()
	orderRows, err := service.pool.Query(ctx, `SELECT status,count(*) FROM orders GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, fmt.Errorf("load order counts by status: %w", err)
	}
	ordersByStatus := make([]map[string]any, 0)
	for orderRows.Next() {
		var status string
		var count int64
		if err := orderRows.Scan(&status, &count); err != nil {
			orderRows.Close()
			return nil, fmt.Errorf("scan order status counts: %w", err)
		}
		ordersByStatus = append(ordersByStatus, map[string]any{"status": status, "count": count})
	}
	if err := orderRows.Err(); err != nil {
		orderRows.Close()
		return nil, fmt.Errorf("iterate order status counts: %w", err)
	}
	orderRows.Close()
	auditRows, err := service.pool.Query(ctx, `
		SELECT id::text,actor_id::text,action,target_type,target_id::text,request_id,created_at
		FROM audit_events ORDER BY created_at DESC,id DESC LIMIT 20`)
	if err != nil {
		return nil, fmt.Errorf("load recent audit events for admin dashboard: %w", err)
	}
	audit := make([]map[string]any, 0)
	for auditRows.Next() {
		var id, action, targetType, requestID string
		var actorID, targetID *string
		var createdAt time.Time
		if err := auditRows.Scan(&id, &actorID, &action, &targetType, &targetID, &requestID, &createdAt); err != nil {
			auditRows.Close()
			return nil, fmt.Errorf("scan recent audit event: %w", err)
		}
		audit = append(audit, map[string]any{"id": id, "actorId": actorID, "action": action, "targetType": targetType, "targetId": targetID, "requestId": requestID, "createdAt": createdAt})
	}
	if err := auditRows.Err(); err != nil {
		auditRows.Close()
		return nil, fmt.Errorf("iterate recent audit events: %w", err)
	}
	auditRows.Close()
	return map[string]any{
		"period": map[string]any{"from": from, "to": to, "bucket": bucket},
		"kpis": map[string]any{"accountsAwaitingApproval": map[string]any{
			"value": pending, "unit": "accounts", "oldestApplicant": func() any {
				if oldestAt == nil {
					return nil
				}
				return map[string]any{"user": oldest, "createdAt": *oldestAt}
			}(),
		}},
		"charts": map[string]any{"signups": signupSeries, "completedVolume": volumeSeries},
		"tables": map[string]any{
			"accountsByRoleStatus": accountsByRoleStatus, "activeListingsByRegion": activeListings,
			"ordersByStatus": ordersByStatus, "recentAuditEvents": audit,
		},
	}, nil
}
