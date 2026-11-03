package auth

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaminflow/zaminflow-api/internal/config"
)

func TestMarketplaceInventoryAndOrderLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	service, err := NewService(ctx, pool, config.Config{
		AppSecret: "marketplace-integration-test-secret-32-characters",
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	farmerID := addTestMarketplaceUser(t, ctx, pool, "farmer")
	processorID := addTestMarketplaceUser(t, ctx, pool, "processor")
	concurrentProcessorID := addTestMarketplaceUser(t, ctx, pool, "processor")
	adminID := addTestMarketplaceUser(t, ctx, pool, "admin")
	t.Cleanup(func() {
		cleanupMarketplaceTestData(t, context.Background(), pool, farmerID, processorID, concurrentProcessorID, adminID)
		pool.Close()
	})
	farmer := Principal{User: User{ID: farmerID, Role: "farmer", AccountStatus: "approved"}}
	processor := Principal{User: User{ID: processorID, Role: "processor", AccountStatus: "approved"}}
	concurrentProcessor := Principal{User: User{ID: concurrentProcessorID, Role: "processor", AccountStatus: "approved"}}
	admin := Principal{User: User{ID: adminID, Role: "admin", AccountStatus: "approved"}, MFAEnabled: true}
	if err := service.RequestDeletion(ctx, farmer, "marketplace-deletion-request"); err != nil {
		t.Fatalf("RequestDeletion() error = %v", err)
	}
	var deletionRequestID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM account_deletion_requests WHERE user_id=$1`, farmerID).Scan(&deletionRequestID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	concurrentListing, err := service.CreateListing(ctx, farmer, ListingInput{
		RegionCode: "andijan", AvailableLiters: 100, FatPercent: 3.5,
		PricePerLiterUZS: 12000, AvailableFrom: now.Add(-time.Minute), AvailableUntil: now.Add(3 * time.Hour),
	}, "test-concurrent-listing")
	if err != nil {
		t.Fatalf("CreateListing() for concurrent reservation test: %v", err)
	}
	const concurrentOrders = 10
	startOrders := make(chan struct{})
	orderResults := make(chan error, concurrentOrders)
	var workers sync.WaitGroup
	for index := 0; index < concurrentOrders; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-startOrders
			_, err := service.CreateOrder(ctx, processor, OrderInput{
				ListingID: concurrentListing.ID, Liters: 20, ExpectedPickupAt: now.Add(time.Hour),
			}, "test-concurrent-order")
			orderResults <- err
		}()
	}
	close(startOrders)
	workers.Wait()
	close(orderResults)
	var createdOrders, rejectedOrders int
	for err := range orderResults {
		switch {
		case err == nil:
			createdOrders++
		case errors.Is(err, ErrConflict):
			rejectedOrders++
		default:
			t.Fatalf("concurrent CreateOrder() error = %v, want success or ErrConflict", err)
		}
	}
	if createdOrders != 5 || rejectedOrders != 5 {
		t.Fatalf("concurrent reservations: created=%d rejected=%d, want 5 each", createdOrders, rejectedOrders)
	}
	listing, err := service.CreateListing(ctx, farmer, ListingInput{
		RegionCode: "andijan", AvailableLiters: 100, FatPercent: 3.5,
		PricePerLiterUZS: 12000, AvailableFrom: now.Add(-time.Minute), AvailableUntil: now.Add(3 * time.Hour),
	}, "test-listing")
	if err != nil {
		t.Fatalf("CreateListing() error = %v", err)
	}
	first, err := service.CreateOrder(ctx, processor, OrderInput{
		ListingID: listing.ID, Liters: 70, ExpectedPickupAt: now.Add(time.Hour), Note: "first order",
	}, "test-order")
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if first.TotalUZS != 840000 || first.Status != "placed" {
		t.Fatalf("first order = %#v", first)
	}
	dashboard, err := service.FarmerDashboard(ctx, farmer, nil)
	if err != nil {
		t.Fatalf("FarmerDashboard() error = %v", err)
	}
	dashboardData, ok := dashboard.(map[string]any)
	if !ok {
		t.Fatalf("FarmerDashboard() = %T, want map[string]any", dashboard)
	}
	dashboardKPIs, ok := dashboardData["kpis"].(map[string]any)
	if !ok {
		t.Fatalf("FarmerDashboard().kpis = %T, want map[string]any", dashboardData["kpis"])
	}
	awaitingResponse, ok := dashboardKPIs["ordersAwaitingResponse"].(map[string]any)
	if !ok {
		t.Fatalf("FarmerDashboard().kpis.ordersAwaitingResponse = %T, want map[string]any",
			dashboardKPIs["ordersAwaitingResponse"])
	}
	wantAwaitingResponse := int64(createdOrders + 1)
	if got, ok := awaitingResponse["value"].(int64); !ok || got != wantAwaitingResponse {
		t.Fatalf("ordersAwaitingResponse.value = %v, want %d", awaitingResponse["value"], wantAwaitingResponse)
	}
	if _, err := service.ReviewDeletionRequest(ctx, admin, deletionRequestID, "approved", "", "marketplace-deletion-approval"); !errors.Is(err, ErrConflict) {
		t.Fatalf("approve deletion request with active order error = %v, want ErrConflict", err)
	}
	if _, err := service.ReviewDeletionRequest(ctx, admin, deletionRequestID, "rejected", "Complete the open order first.", "marketplace-deletion-rejection"); err != nil {
		t.Fatalf("reject deletion request with active order: %v", err)
	}
	if _, err := service.CreateOrder(ctx, processor, OrderInput{
		ListingID: listing.ID, Liters: 31, ExpectedPickupAt: now.Add(time.Hour),
	}, "test-over-order"); !errors.Is(err, ErrConflict) {
		t.Fatalf("oversized order error = %v, want ErrConflict", err)
	}
	if _, err := service.TransitionOrder(ctx, processor, first.ID, "accepted", "", "test-wrong-actor"); !errors.Is(err, ErrConflict) {
		t.Fatalf("processor accept error = %v, want ErrConflict", err)
	}
	for _, status := range []string{"accepted", "ready_for_pickup", "picked_up", "delivered"} {
		first, err = service.TransitionOrder(ctx, farmer, first.ID, status, "", "test-transition")
		if err != nil {
			t.Fatalf("farmer transition to %s: %v", status, err)
		}
	}
	if err := service.CreateQualityReport(ctx, farmer, first.ID, QualityReportInput{
		Measurements: map[string]float64{"fatPercent": 3.5, "temperatureC": 4.0},
		Outcome:      "accepted", Note: "quality checked",
	}, "test-quality"); err != nil {
		t.Fatalf("CreateQualityReport() error = %v", err)
	}
	first, err = service.TransitionOrder(ctx, processor, first.ID, "completed", "", "test-complete")
	if err != nil {
		t.Fatalf("complete order: %v", err)
	}
	if first.Status != "completed" || first.QualityOutcome == nil || *first.QualityOutcome != "accepted" || len(first.History) != 6 {
		t.Fatalf("completed order = %#v", first)
	}
	listing, err = service.GetListing(ctx, farmer, listing.ID)
	if err != nil {
		t.Fatalf("GetListing() error = %v", err)
	}
	if listing.AvailableLiters != 30 {
		t.Fatalf("available liters after completion = %d, want 30", listing.AvailableLiters)
	}
	if _, err := service.CreateOrder(ctx, processor, OrderInput{
		ListingID: listing.ID, Liters: 30, ExpectedPickupAt: now.Add(time.Hour),
	}, "test-second-order"); err != nil {
		t.Fatalf("place order from released inventory: %v", err)
	}
	if err := service.RequestDeletion(ctx, concurrentProcessor, "concurrent-deletion-request"); err != nil {
		t.Fatalf("request deletion for concurrent review test: %v", err)
	}
	var concurrentDeletionRequestID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM account_deletion_requests WHERE user_id=$1 AND status='requested'`,
		concurrentProcessorID).Scan(&concurrentDeletionRequestID); err != nil {
		t.Fatal(err)
	}
	concurrentDeletionListing, err := service.CreateListing(ctx, farmer, ListingInput{
		RegionCode: "andijan", AvailableLiters: 10, FatPercent: 3.5,
		PricePerLiterUZS: 12000, AvailableFrom: now.Add(-time.Minute), AvailableUntil: now.Add(3 * time.Hour),
	}, "test-concurrent-deletion-listing")
	if err != nil {
		t.Fatalf("create listing for concurrent deletion review test: %v", err)
	}
	lockTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer lockTx.Rollback(context.Background())
	if _, err := lockTx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, concurrentProcessorID); err != nil {
		t.Fatal(err)
	}
	raceOrderResults := make(chan error, 1)
	go func() {
		_, err := service.CreateOrder(ctx, concurrentProcessor, OrderInput{
			ListingID: concurrentDeletionListing.ID, Liters: 5, ExpectedPickupAt: now.Add(time.Hour),
		}, "test-order-during-deletion-review")
		raceOrderResults <- err
	}()
	waitForDatabaseLock(t, ctx, pool, "%SELECT id::text FROM users WHERE id IN%")
	raceReviewResults := make(chan error, 1)
	go func() {
		_, err := service.ReviewDeletionRequest(ctx, admin, concurrentDeletionRequestID, "approved", "", "concurrent-deletion-approval")
		raceReviewResults <- err
	}()
	waitForDatabaseLock(t, ctx, pool, "%SELECT id::text FROM users WHERE id=$1 FOR UPDATE%")
	if err := lockTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-raceOrderResults; err != nil {
		t.Fatalf("order serialized before deletion review: %v", err)
	}
	if err := <-raceReviewResults; !errors.Is(err, ErrConflict) {
		t.Fatalf("deletion approval serialized after active order error = %v, want ErrConflict", err)
	}
	if _, err := service.FarmerDashboard(ctx, farmer, nil); err != nil {
		t.Fatalf("FarmerDashboard() error = %v", err)
	}
	if _, err := service.ProcessorDashboard(ctx, processor, nil); err != nil {
		t.Fatalf("ProcessorDashboard() error = %v", err)
	}
	if _, err := service.AdminDashboard(ctx, nil); err != nil {
		t.Fatalf("AdminDashboard() error = %v", err)
	}
}

func waitForDatabaseLock(t *testing.T, ctx context.Context, pool *pgxpool.Pool, queryPattern string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE pid<>pg_backend_pid() AND datname=current_database()
				  AND wait_event_type='Lock' AND query LIKE $1
			)`, queryPattern).Scan(&waiting); err != nil {
			t.Fatalf("check for blocked database operation: %v", err)
		}
		if waiting {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("database operation matching %q did not wait for a lock", queryPattern)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func addTestMarketplaceUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role string) string {
	t.Helper()
	id, err := newUUID()
	if err != nil {
		t.Fatal(err)
	}
	email := id + "@integration.invalid"
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id,role,account_status,email,password_hash,email_verified_at,full_name,phone,organization_name,region_code,stir)
		VALUES ($1,$2,'approved',$3,'unused',now(),'Integration User','+998901234567','Test Organization','andijan','123456789')`,
		id, role, email)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func cleanupMarketplaceTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userIDs ...string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		DELETE FROM account_deletion_requests WHERE user_id=ANY($1::uuid[]) OR reviewed_by=ANY($1::uuid[])`,
		userIDs)
	if err != nil {
		t.Errorf("delete integration deletion requests: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM quality_reports WHERE reporter_id=ANY($1::uuid[])
		   OR order_id IN (SELECT id FROM orders WHERE farmer_id=ANY($1::uuid[]) OR processor_id=ANY($1::uuid[]))`,
		userIDs)
	if err != nil {
		t.Errorf("delete integration quality reports: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM order_disputes WHERE opened_by=ANY($1::uuid[])
		   OR order_id IN (SELECT id FROM orders WHERE farmer_id=ANY($1::uuid[]) OR processor_id=ANY($1::uuid[]))`,
		userIDs)
	if err != nil {
		t.Errorf("delete integration disputes: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM order_status_history WHERE actor_id=ANY($1::uuid[])
		   OR order_id IN (SELECT id FROM orders WHERE farmer_id=ANY($1::uuid[]) OR processor_id=ANY($1::uuid[]))`,
		userIDs)
	if err != nil {
		t.Errorf("delete integration order history: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM orders WHERE farmer_id=ANY($1::uuid[]) OR processor_id=ANY($1::uuid[])`, userIDs)
	if err != nil {
		t.Errorf("delete integration orders: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM milk_listings WHERE farmer_id=ANY($1::uuid[])`, userIDs)
	if err != nil {
		t.Errorf("delete integration listings: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM audit_events WHERE actor_id=ANY($1::uuid[]) OR target_id=ANY($1::uuid[])
		   OR metadata->>'userId'=ANY($1::text[])`, userIDs)
	if err != nil {
		t.Errorf("delete integration audit events: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM account_reviews WHERE user_id=ANY($1::uuid[]) OR reviewer_id=ANY($1::uuid[])`, userIDs)
	if err != nil {
		t.Errorf("delete integration account reviews: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE id=ANY($1::uuid[])`, userIDs)
	if err != nil {
		t.Errorf("delete integration users: %v", err)
	}
}
