package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaminflow/zaminflow-api/internal/config"
)

func TestEmailVerificationAndAccountReviewLifecycle(t *testing.T) {
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
		AppSecret:  "approval-integration-test-secret-32-characters",
		WebBaseURL: "https://example.invalid",
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	email := "applicant-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "") + "@integration.invalid"
	adminEmail := "reviewer-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "") + "@integration.invalid"
	adminPassword := "an-admin-password-that-is-unique-937!"
	if err := service.CreateAdmin(ctx, adminEmail, adminPassword, "Integration Reviewer"); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	var adminID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM users WHERE email=$1`, adminEmail).Scan(&adminID); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupApprovalTestData(t, context.Background(), pool, email, adminEmail)
		pool.Close()
	})
	enrollment, err := service.SignIn(ctx, SignInInput{Email: adminEmail, Password: adminPassword}, "", "", "mfa-enrollment")
	if err != nil {
		t.Fatalf("admin sign-in for MFA enrollment: %v", err)
	}
	enrollmentPrincipal, err := service.Authenticate(ctx, enrollment.AccessToken)
	if err != nil || !enrollmentPrincipal.EnrollmentOnly || enrollmentPrincipal.MFAEnabled {
		t.Fatalf("initial admin principal = %#v, error = %v", enrollmentPrincipal, err)
	}
	totpSetup, err := service.BeginTOTP(ctx, enrollmentPrincipal)
	if err != nil {
		t.Fatalf("BeginTOTP() error = %v", err)
	}
	code, err := totpCode(totpSetup["secret"], time.Now())
	if err != nil {
		t.Fatal(err)
	}
	recoveryCodes, err := service.ConfirmTOTP(ctx, enrollmentPrincipal, code)
	if err != nil || len(recoveryCodes) != 10 {
		t.Fatalf("ConfirmTOTP() codes=%d error=%v", len(recoveryCodes), err)
	}
	if _, err := service.Authenticate(ctx, enrollment.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("pre-enrollment session remained valid: %v", err)
	}
	authenticated, err := service.SignIn(ctx, SignInInput{Email: adminEmail, Password: adminPassword, TOTPCode: code}, "", "", "mfa-signin")
	if err != nil {
		t.Fatalf("admin sign-in with TOTP: %v", err)
	}
	adminPrincipal, err := service.Authenticate(ctx, authenticated.AccessToken)
	if err != nil || adminPrincipal.EnrollmentOnly || !adminPrincipal.MFAEnabled {
		t.Fatalf("MFA admin principal = %#v, error = %v", adminPrincipal, err)
	}
	applicant, err := service.SignUp(ctx, SignUpInput{
		FullName: "Integration Applicant", Email: email, Password: "integration-password-that-is-unique-482!",
		Role: "farmer", Phone: "+998901234568", OrganizationName: "Integration Dairy",
		RegionCode: "andijan", STIR: "987654321",
	}, "approval-test-signup")
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	if applicant.AccountStatus != "pending" || applicant.EmailVerified {
		t.Fatalf("new applicant state = %#v", applicant)
	}
	rateLimitKey := "rate-limit-test-" + email
	if err := service.CheckRateLimit(ctx, "integration", rateLimitKey, 1, time.Hour); err != nil {
		t.Fatalf("first rate-limited request: %v", err)
	}
	if err := service.CheckRateLimit(ctx, "integration", rateLimitKey, 1, time.Hour); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("second rate-limited request error = %v, want ErrRateLimited", err)
	}
	var messageBody string
	if err := pool.QueryRow(ctx, `
		SELECT body FROM mail_outbox WHERE recipient=$1 AND subject='Verify your ZaminFlow email'
		ORDER BY created_at DESC LIMIT 1`, email).Scan(&messageBody); err != nil {
		t.Fatal(err)
	}
	verificationURL, err := url.Parse(strings.TrimSpace(strings.TrimPrefix(messageBody, "Verify your email address using this link:\n\n")))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyEmail(ctx, verificationURL.Query().Get("token")); err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}
	admin := adminPrincipal
	if _, err := service.GetOwnDeletionRequest(ctx, admin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin own deletion request status error = %v, want ErrForbidden", err)
	}
	applications, err := service.ListApplications(ctx, "farmer", "andijan", 1, 20)
	if err != nil {
		t.Fatalf("ListApplications() error = %v", err)
	}
	if !strings.Contains(toJSON(t, applications), applicant.ID) {
		t.Fatal("verified applicant was not in the administrator review queue")
	}
	if _, err := service.ReviewAccount(ctx, admin, applicant.ID, "rejected", "Please correct the registration details.", "approval-test-reject"); err != nil {
		t.Fatalf("reject applicant: %v", err)
	}
	rejected, err := service.Me(ctx, Principal{User: User{ID: applicant.ID}})
	if err != nil || rejected.AccountStatus != "rejected" || rejected.RejectionReason == nil {
		t.Fatalf("rejected account = %#v, error = %v", rejected, err)
	}
	resubmitted, err := service.Resubmit(ctx, Principal{User: User{ID: applicant.ID}}, "approval-test-resubmit")
	if err != nil || resubmitted.AccountStatus != "pending" || resubmitted.RejectionReason != nil {
		t.Fatalf("resubmitted account = %#v, error = %v", resubmitted, err)
	}
	approved, err := service.ReviewAccount(ctx, admin, applicant.ID, "approved", "", "approval-test-approve")
	if err != nil || approved.AccountStatus != "approved" {
		t.Fatalf("approved account = %#v, error = %v", approved, err)
	}
	suspended, err := service.ReviewAccount(ctx, admin, applicant.ID, "suspended", "Policy review.", "approval-test-suspend")
	if err != nil || suspended.AccountStatus != "suspended" {
		t.Fatalf("suspended account = %#v, error = %v", suspended, err)
	}
	reinstated, err := service.ReviewAccount(ctx, admin, applicant.ID, "reinstated", "", "approval-test-reinstate")
	if err != nil || reinstated.AccountStatus != "approved" {
		t.Fatalf("reinstated account = %#v, error = %v", reinstated, err)
	}
	if err := service.RequestDeletion(ctx, Principal{User: reinstated}, "deletion-test-reject-request"); err != nil {
		t.Fatalf("request account deletion: %v", err)
	}
	requests, err := service.ListDeletionRequests(ctx, "", 1, 20)
	if err != nil || !strings.Contains(toJSON(t, requests), applicant.ID) {
		t.Fatalf("ListDeletionRequests() = %s, error = %v", toJSON(t, requests), err)
	}
	router := chi.NewRouter()
	RegisterRoutes(router, service, service.config)
	adminQueueRequest := httptest.NewRequest(http.MethodGet, "/admin/deletion-requests", nil)
	adminQueueRequest.Header.Set("Authorization", "Bearer "+authenticated.AccessToken)
	adminQueueResponse := httptest.NewRecorder()
	router.ServeHTTP(adminQueueResponse, adminQueueRequest)
	if adminQueueResponse.Code != http.StatusOK || !strings.Contains(adminQueueResponse.Body.String(), applicant.ID) {
		t.Fatalf("admin deletion queue response = %d %s", adminQueueResponse.Code, adminQueueResponse.Body.String())
	}
	applicantAuth, err := service.SignIn(ctx, SignInInput{
		Email: email, Password: "integration-password-that-is-unique-482!",
	}, "", "", "deletion-test-applicant-sign-in")
	if err != nil {
		t.Fatalf("sign in applicant for deletion request status: %v", err)
	}
	ownRequest := httptest.NewRequest(http.MethodGet, "/me/deletion-request", nil)
	ownRequest.Header.Set("Authorization", "Bearer "+applicantAuth.AccessToken)
	ownResponse := httptest.NewRecorder()
	router.ServeHTTP(ownResponse, ownRequest)
	if ownResponse.Code != http.StatusOK || !strings.Contains(ownResponse.Body.String(), `"status":"requested"`) {
		t.Fatalf("own deletion request response = %d %s", ownResponse.Code, ownResponse.Body.String())
	}
	if _, err := service.ListDeletionRequests(ctx, "invalid", 1, 20); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("invalid deletion request status error = %v, want ErrInvalidQuery", err)
	}
	if _, err := service.ReviewDeletionRequest(ctx, Principal{User: applicant}, "00000000-0000-4000-8000-000000000001", "rejected", "Not an administrator.", "deletion-test-forbidden"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin deletion review error = %v, want ErrForbidden", err)
	}
	var rejectedRequestID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM account_deletion_requests WHERE user_id=$1 AND status='requested'`,
		applicant.ID).Scan(&rejectedRequestID); err != nil {
		t.Fatal(err)
	}
	rejectRequest := httptest.NewRequest(http.MethodPost, "/admin/deletion-requests/"+rejectedRequestID+"/reject", strings.NewReader(`{"reason":"Trading records must be retained."}`))
	rejectRequest.Header.Set("Authorization", "Bearer "+authenticated.AccessToken)
	rejectRequest.Header.Set("Content-Type", "application/json")
	rejectRequest.Header.Set("X-Request-ID", "deletion-test-reject")
	rejectResponse := httptest.NewRecorder()
	router.ServeHTTP(rejectResponse, rejectRequest)
	if rejectResponse.Code != http.StatusOK || !strings.Contains(rejectResponse.Body.String(), `"status":"rejected"`) ||
		!strings.Contains(rejectResponse.Body.String(), admin.ID) {
		t.Fatalf("reject deletion request response = %d %s", rejectResponse.Code, rejectResponse.Body.String())
	}
	nonAdminQueueRequest := httptest.NewRequest(http.MethodGet, "/admin/deletion-requests", nil)
	nonAdminQueueRequest.Header.Set("Authorization", "Bearer "+applicantAuth.AccessToken)
	nonAdminQueueResponse := httptest.NewRecorder()
	router.ServeHTTP(nonAdminQueueResponse, nonAdminQueueRequest)
	if nonAdminQueueResponse.Code != http.StatusForbidden {
		t.Fatalf("non-admin deletion queue status = %d, want %d", nonAdminQueueResponse.Code, http.StatusForbidden)
	}
	ownRequest = httptest.NewRequest(http.MethodGet, "/me/deletion-request", nil)
	ownRequest.Header.Set("Authorization", "Bearer "+applicantAuth.AccessToken)
	ownResponse = httptest.NewRecorder()
	router.ServeHTTP(ownResponse, ownRequest)
	if ownResponse.Code != http.StatusOK || !strings.Contains(ownResponse.Body.String(), `"status":"rejected"`) ||
		!strings.Contains(ownResponse.Body.String(), "Trading records must be retained.") {
		t.Fatalf("own reviewed deletion request response = %d %s", ownResponse.Code, ownResponse.Body.String())
	}
	var accountStatus string
	if err := pool.QueryRow(ctx, `SELECT account_status FROM users WHERE id=$1`, applicant.ID).Scan(&accountStatus); err != nil || accountStatus != "approved" {
		t.Fatalf("account status after deletion review = %q, error = %v", accountStatus, err)
	}
	if _, err := service.ReviewDeletionRequest(ctx, admin, rejectedRequestID, "approved", "", "deletion-test-repeat"); !errors.Is(err, ErrConflict) {
		t.Fatalf("re-reviewing a resolved deletion request error = %v, want ErrConflict", err)
	}
	if err := service.RequestDeletion(ctx, Principal{User: reinstated}, "deletion-test-approve-request"); err != nil {
		t.Fatalf("submit a new request after rejection: %v", err)
	}
	var approvedRequestID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM account_deletion_requests WHERE user_id=$1 AND status='requested'`,
		applicant.ID).Scan(&approvedRequestID); err != nil {
		t.Fatal(err)
	}
	approvedRequest, err := service.ReviewDeletionRequest(ctx, admin, approvedRequestID, "approved", "Refer to the approved retention workflow.", "deletion-test-approve")
	if err != nil || approvedRequest.Status != "approved" {
		t.Fatalf("approved deletion request = %#v, error = %v", approvedRequest, err)
	}
	if err := service.RequestDeletion(ctx, Principal{User: reinstated}, "deletion-test-pending-duplicate"); !errors.Is(err, ErrConflict) {
		t.Fatalf("request while manual deletion workflow is pending error = %v, want ErrConflict", err)
	}
	var deletionAuditCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_events WHERE target_type='account_deletion_request'
		  AND target_id IN ($1,$2)`, rejectedRequestID, approvedRequestID).Scan(&deletionAuditCount); err != nil || deletionAuditCount != 2 {
		t.Fatalf("deletion review audit count = %d, error = %v", deletionAuditCount, err)
	}
	if _, err := service.ReviewAccount(ctx, admin, adminID, "suspended", "self-protection", "approval-test-self"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("reviewing an admin error = %v, want ErrForbidden", err)
	}
	if _, err := service.GetApplication(ctx, applicant.ID); err != nil {
		t.Fatalf("GetApplication() error = %v", err)
	}
	if _, err := service.ListAuditEvents(ctx, url.Values{}, 1, 20); err != nil {
		t.Fatalf("ListAuditEvents() error = %v", err)
	}
}

func toJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func cleanupApprovalTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, emails ...string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		DELETE FROM account_deletion_requests WHERE user_id IN (SELECT id FROM users WHERE email=ANY($1::citext[]))
		   OR reviewed_by IN (SELECT id FROM users WHERE email=ANY($1::citext[]))`, emails)
	if err != nil {
		t.Errorf("delete approval test deletion requests: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM mail_outbox WHERE recipient=ANY($1::text[])`, emails)
	if err != nil {
		t.Errorf("delete approval test email: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM audit_events WHERE actor_id IN (SELECT id FROM users WHERE email=ANY($1::citext[]))
		   OR target_id IN (SELECT id FROM users WHERE email=ANY($1::citext[]))
		   OR metadata->>'userId' IN (SELECT id::text FROM users WHERE email=ANY($1::citext[]))`, emails)
	if err != nil {
		t.Errorf("delete approval test audit events: %v", err)
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM account_reviews WHERE user_id IN (SELECT id FROM users WHERE email=ANY($1::citext[]))
		   OR reviewer_id IN (SELECT id FROM users WHERE email=ANY($1::citext[]))`, emails)
	if err != nil {
		t.Errorf("delete approval test review history: %v", err)
	}
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE email=ANY($1::citext[])`, emails)
	if err != nil {
		t.Errorf("delete approval test users: %v", err)
	}
}
