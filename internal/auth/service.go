package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zaminflow/zaminflow-api/internal/config"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidQuery       = errors.New("invalid query parameter")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrNotFound           = errors.New("not found")
	ErrRateLimited        = errors.New("rate limited")
	ErrAccountUnapproved  = errors.New("account not approved")
	ErrAccountRejected    = errors.New("account rejected")
	ErrAccountSuspended   = errors.New("account suspended")
)

type Service struct {
	pool      *pgxpool.Pool
	config    config.Config
	logger    *slog.Logger
	dummyHash string
}

type User struct {
	ID               string    `json:"id"`
	Role             string    `json:"role"`
	AccountStatus    string    `json:"accountStatus"`
	Email            string    `json:"email"`
	EmailVerified    bool      `json:"emailVerified"`
	FullName         string    `json:"fullName"`
	Phone            *string   `json:"phone,omitempty"`
	OrganizationName *string   `json:"organizationName,omitempty"`
	RegionCode       *string   `json:"regionCode,omitempty"`
	STIR             *string   `json:"stir,omitempty"`
	RejectionReason  *string   `json:"rejectionReason,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Principal struct {
	User
	SessionID      string
	EnrollmentOnly bool
	MFAEnabled     bool
}

type SignUpInput struct {
	FullName         string `json:"fullName"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Role             string `json:"role"`
	Phone            string `json:"phone"`
	OrganizationName string `json:"organizationName"`
	RegionCode       string `json:"regionCode"`
	STIR             string `json:"stir"`
}

type SignInInput struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	TOTPCode     string `json:"totpCode"`
	RecoveryCode string `json:"recoveryCode"`
}

type ProfileUpdate struct {
	FullName         *string `json:"fullName"`
	Phone            *string `json:"phone"`
	OrganizationName *string `json:"organizationName"`
	RegionCode       *string `json:"regionCode"`
	STIR             *string `json:"stir"`
}

type AuthResponse struct {
	AccessToken      string    `json:"accessToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	User             User      `json:"user"`
	SessionID        string    `json:"-"`
	RefreshToken     string    `json:"-"`
	RefreshExpiresAt time.Time `json:"-"`
}

type page struct {
	Items    []User `json:"items"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Total    int64  `json:"total"`
}

type AccountDeletionRequest struct {
	ID           string     `json:"id"`
	User         User       `json:"user"`
	Status       string     `json:"status"`
	RequestedAt  time.Time  `json:"requestedAt"`
	ReviewedBy   *string    `json:"reviewedBy,omitempty"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	ReviewReason *string    `json:"reviewReason,omitempty"`
}

type OwnDeletionRequest struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requestedAt"`
	ReviewedAt  *time.Time `json:"reviewedAt,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
}

func NewService(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	dummyHash, err := hashPassword("not-a-real-user-password")
	if err != nil {
		return nil, fmt.Errorf("initialize password timing guard: %w", err)
	}
	return &Service{pool: pool, config: cfg, logger: logger, dummyHash: dummyHash}, nil
}

func (service *Service) CheckRateLimit(ctx context.Context, scope, value string, limit int, window time.Duration) error {
	if limit < 1 || window <= 0 {
		return ErrInvalidInput
	}
	mac := hmac.New(sha256.New, deriveKey(service.config.AppSecret, "request-rate-limit"))
	_, _ = mac.Write([]byte(scope + ":" + value))
	key := hex.EncodeToString(mac.Sum(nil))
	var blockedUntil sql.NullTime
	err := service.pool.QueryRow(ctx, `
		INSERT INTO request_rate_limits (bucket_key,window_started_at,request_count,blocked_until)
		VALUES ($1,now(),1,NULL)
		ON CONFLICT (bucket_key) DO UPDATE SET
			window_started_at=CASE
				WHEN request_rate_limits.window_started_at+$3::interval <= now() THEN now()
				ELSE request_rate_limits.window_started_at END,
			request_count=CASE
				WHEN request_rate_limits.window_started_at+$3::interval <= now() THEN 1
				ELSE request_rate_limits.request_count+1 END,
			blocked_until=CASE
				WHEN request_rate_limits.blocked_until>now()
				 AND request_rate_limits.window_started_at+$3::interval>now() THEN request_rate_limits.blocked_until
				WHEN request_rate_limits.window_started_at+$3::interval<=now() THEN NULL
				WHEN request_rate_limits.request_count+1 > $2 THEN now()+$3::interval
				ELSE NULL END
		RETURNING blocked_until`, key, limit, window.String()).Scan(&blockedUntil)
	if err != nil {
		return fmt.Errorf("check request rate limit: %w", err)
	}
	if blockedUntil.Valid && blockedUntil.Time.After(time.Now()) {
		return ErrRateLimited
	}
	return nil
}

func (service *Service) RunMaintenance(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if _, err := service.pool.Exec(ctx, `
			DELETE FROM request_rate_limits
			WHERE window_started_at < now()-interval '48 hours'`); err != nil && ctx.Err() == nil {
			service.logger.ErrorContext(ctx, "clean expired rate-limit buckets", "error", err)
		}
		if _, err := service.pool.Exec(ctx, `
			DELETE FROM one_time_tokens
			WHERE expires_at < now()-interval '7 days'
			   OR consumed_at < now()-interval '7 days'`); err != nil && ctx.Err() == nil {
			service.logger.ErrorContext(ctx, "clean expired one-time tokens", "error", err)
		}
		if _, err := service.pool.Exec(ctx, `
			DELETE FROM sessions WHERE expires_at < now()-interval '90 days'`); err != nil && ctx.Err() == nil {
			service.logger.ErrorContext(ctx, "clean expired sessions", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (service *Service) SignUp(ctx context.Context, input SignUpInput, requestID string) (User, error) {
	normalized, err := validateSignUp(input)
	if err != nil {
		return User{}, err
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	userID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	verificationToken, err := randomToken(32)
	if err != nil {
		return User{}, err
	}
	verificationID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin registration: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO users (
			id, role, account_status, email, password_hash, full_name,
			phone, organization_name, region_code, stir
		) VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, role, account_status, email, (email_verified_at IS NOT NULL),
		          full_name, phone, organization_name, region_code, stir,
		          rejection_reason, created_at, updated_at`,
		userID, normalized.Role, normalized.Email, passwordHash, normalized.FullName,
		normalized.Phone, normalized.OrganizationName, normalized.RegionCode, normalized.STIR)
	user, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("create account: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO one_time_tokens (id, user_id, token_hash, purpose, expires_at)
		VALUES ($1, $2, $3, 'email_verification', now() + interval '24 hours')`,
		verificationID, userID, tokenHash(verificationToken)); err != nil {
		return User{}, fmt.Errorf("create email verification token: %w", err)
	}
	link := service.config.WebBaseURL + "/verify-email?token=" + url.QueryEscape(verificationToken)
	if err := enqueueMail(ctx, tx, user.Email, "Verify your ZaminFlow email", "Verify your email address using this link:\n\n"+link); err != nil {
		return User{}, err
	}
	auditID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id, actor_id, action, target_type, target_id, request_id)
		VALUES ($1, $2, 'account.sign_up', 'user', $2, $3)`, auditID, userID, requestID); err != nil {
		return User{}, fmt.Errorf("record sign-up audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit registration: %w", err)
	}
	return user, nil
}

func (service *Service) SignIn(ctx context.Context, input SignInInput, userAgent, ipAddress, requestID string) (AuthResponse, error) {
	email := normalizeEmail(input.Email)
	var userID, role, status, encoded, rejection string
	var verified, mfaRequired, mfaEnabled bool
	var lockedUntil sql.NullTime
	err := service.pool.QueryRow(ctx, `
		SELECT id::text, role, account_status, password_hash,
		       (email_verified_at IS NOT NULL), COALESCE(rejection_reason, ''),
		       admin_mfa_required, (admin_mfa_enabled_at IS NOT NULL), locked_until
		FROM users WHERE email = $1`, email).Scan(
		&userID, &role, &status, &encoded, &verified, &rejection, &mfaRequired, &mfaEnabled, &lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, hashErr := verifyPassword(service.dummyHash, input.Password); hashErr != nil {
			return AuthResponse{}, fmt.Errorf("verify timing-guard password: %w", hashErr)
		}
		return AuthResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResponse{}, fmt.Errorf("load sign-in account: %w", err)
	}
	if lockedUntil.Valid && lockedUntil.Time.After(time.Now()) {
		return AuthResponse{}, ErrRateLimited
	}
	passwordOK, err := verifyPassword(encoded, input.Password)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("verify password: %w", err)
	}
	if !passwordOK {
		if err := service.recordFailedLogin(ctx, userID); err != nil {
			return AuthResponse{}, err
		}
		return AuthResponse{}, ErrInvalidCredentials
	}
	if status == "suspended" {
		return AuthResponse{}, ErrAccountSuspended
	}

	mfaVerified := false
	if role == "admin" && mfaRequired {
		if !mfaEnabled {
			mfaVerified = false
		} else if input.TOTPCode != "" {
			var encrypted []byte
			if err := service.pool.QueryRow(ctx, `SELECT admin_totp_secret FROM users WHERE id=$1`, userID).Scan(&encrypted); err != nil {
				return AuthResponse{}, fmt.Errorf("load admin TOTP secret: %w", err)
			}
			secret, err := decryptSecret(service.config.AppSecret, encrypted)
			if err != nil {
				return AuthResponse{}, fmt.Errorf("decrypt admin TOTP secret: %w", err)
			}
			if !verifyTOTP(secret, input.TOTPCode, time.Now()) {
				if err := service.recordFailedLogin(ctx, userID); err != nil {
					return AuthResponse{}, err
				}
				return AuthResponse{}, ErrInvalidCredentials
			}
			mfaVerified = true
		} else if input.RecoveryCode != "" {
			valid, err := service.consumeRecoveryCode(ctx, userID, input.RecoveryCode)
			if err != nil {
				return AuthResponse{}, err
			}
			if !valid {
				if err := service.recordFailedLogin(ctx, userID); err != nil {
					return AuthResponse{}, err
				}
				return AuthResponse{}, ErrInvalidCredentials
			}
			mfaVerified = true
		} else {
			return AuthResponse{}, ErrInvalidCredentials
		}
	}
	if _, err := service.pool.Exec(ctx, `
		UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id=$1`, userID); err != nil {
		return AuthResponse{}, fmt.Errorf("clear sign-in lockout: %w", err)
	}
	return service.createSession(ctx, userID, mfaRequired && !mfaVerified, userAgent, ipAddress, requestID)
}

func (service *Service) recordFailedLogin(ctx context.Context, userID string) error {
	if _, err := service.pool.Exec(ctx, `
		UPDATE users SET failed_login_attempts=failed_login_attempts+1,
			locked_until=CASE WHEN failed_login_attempts+1>=5
				THEN now()+interval '15 minutes' ELSE locked_until END
		WHERE id=$1`, userID); err != nil {
		return fmt.Errorf("record failed sign-in: %w", err)
	}
	return nil
}

func (service *Service) createSession(
	ctx context.Context,
	userID string,
	enrollmentOnly bool,
	userAgent, ipAddress, requestID string,
) (AuthResponse, error) {
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("begin session creation: %w", err)
	}
	defer tx.Rollback(ctx)
	sessionID, err := newUUID()
	if err != nil {
		return AuthResponse{}, err
	}
	familyID, err := newUUID()
	if err != nil {
		return AuthResponse{}, err
	}
	refresh, err := randomToken(32)
	if err != nil {
		return AuthResponse{}, err
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	query := `
		INSERT INTO sessions (id, user_id, family_id, refresh_token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, '')::inet)`
	if _, err := tx.Exec(ctx, query, sessionID, userID, familyID, tokenHash(refresh), expiresAt, userAgent, ipAddress); err != nil {
		return AuthResponse{}, fmt.Errorf("create refresh session: %w", err)
	}
	accessExpires := time.Now().UTC().Add(15 * time.Minute)
	access, err := signAccessToken(deriveKey(service.config.AppSecret, "access-token"), accessClaims{
		Subject: userID, Session: sessionID, Issuer: "zaminflow-api", Audience: "zaminflow-clients",
		IssuedAt: time.Now().UTC().Unix(), Expires: accessExpires.Unix(), EnrollmentOnly: enrollmentOnly,
	})
	if err != nil {
		return AuthResponse{}, err
	}
	user, err := queryUserTx(ctx, tx, userID)
	if err != nil {
		return AuthResponse{}, err
	}
	if requestID != "" {
		auditID, err := newUUID()
		if err != nil {
			return AuthResponse{}, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_events (id, actor_id, action, target_type, target_id, request_id)
			VALUES ($1, $2, 'auth.sign_in', 'user', $2, $3)`, auditID, userID, requestID); err != nil {
			return AuthResponse{}, fmt.Errorf("record sign-in audit event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AuthResponse{}, fmt.Errorf("commit new session: %w", err)
	}
	return AuthResponse{
		AccessToken: access, ExpiresAt: accessExpires, User: user,
		SessionID: sessionID, RefreshToken: refresh, RefreshExpiresAt: expiresAt,
	}, nil
}

func (service *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	claims, err := verifyAccessToken(deriveKey(service.config.AppSecret, "access-token"), token,
		"zaminflow-api", "zaminflow-clients", time.Now())
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	var user User
	var mfaEnabled bool
	var sessionID string
	err = service.pool.QueryRow(ctx, `
		SELECT u.id::text, u.role, u.account_status, u.email,
		       (u.email_verified_at IS NOT NULL), u.full_name, u.phone,
		       u.organization_name, u.region_code, u.stir, u.rejection_reason,
		       u.created_at, u.updated_at, (u.admin_mfa_enabled_at IS NOT NULL),
		       s.id::text
		FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.id=$1 AND u.id=$2 AND s.revoked_at IS NULL AND s.expires_at > now()`,
		claims.Session, claims.Subject).Scan(
		&user.ID, &user.Role, &user.AccountStatus, &user.Email, &user.EmailVerified,
		&user.FullName, &user.Phone, &user.OrganizationName, &user.RegionCode, &user.STIR,
		&user.RejectionReason, &user.CreatedAt, &user.UpdatedAt, &mfaEnabled, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrUnauthorized
	}
	if err != nil {
		return Principal{}, fmt.Errorf("authenticate access token: %w", err)
	}
	if user.AccountStatus == "suspended" {
		return Principal{}, ErrAccountSuspended
	}
	return Principal{User: user, SessionID: sessionID, EnrollmentOnly: claims.EnrollmentOnly, MFAEnabled: mfaEnabled}, nil
}

func (service *Service) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress, requestID string) (AuthResponse, error) {
	if refreshToken == "" {
		return AuthResponse{}, ErrUnauthorized
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("begin session refresh: %w", err)
	}
	defer tx.Rollback(ctx)
	var sessionID, userID, familyID, status string
	var expired, revoked bool
	var mfaRequired, mfaEnabled bool
	err = tx.QueryRow(ctx, `
		SELECT s.id::text, s.user_id::text, s.family_id::text,
		       (s.expires_at <= now()), (s.revoked_at IS NOT NULL),
		       u.account_status, u.admin_mfa_required,
		       (u.admin_mfa_enabled_at IS NOT NULL)
		FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.refresh_token_hash=$1 FOR UPDATE OF s`, tokenHash(refreshToken)).Scan(
		&sessionID, &userID, &familyID, &expired, &revoked, &status,
		&mfaRequired, &mfaEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthResponse{}, ErrUnauthorized
	}
	if err != nil {
		return AuthResponse{}, fmt.Errorf("load refresh session: %w", err)
	}
	if revoked {
		if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at, now()) WHERE family_id=$1`, familyID); err != nil {
			return AuthResponse{}, fmt.Errorf("revoke reused refresh-token family: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return AuthResponse{}, fmt.Errorf("commit refresh-token reuse response: %w", err)
		}
		return AuthResponse{}, ErrConflict
	}
	if expired {
		return AuthResponse{}, ErrUnauthorized
	}
	if status == "suspended" {
		_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at, now()) WHERE user_id=$1`, userID)
		if err != nil {
			return AuthResponse{}, fmt.Errorf("revoke suspended user's sessions: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return AuthResponse{}, fmt.Errorf("commit session revocation: %w", err)
		}
		return AuthResponse{}, ErrAccountSuspended
	}

	newSessionID, err := newUUID()
	if err != nil {
		return AuthResponse{}, err
	}
	nextRefresh, err := randomToken(32)
	if err != nil {
		return AuthResponse{}, err
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	if _, err := tx.Exec(ctx, `
		INSERT INTO sessions (id, user_id, family_id, refresh_token_hash, expires_at, user_agent, ip_address)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,'')::inet)`,
		newSessionID, userID, familyID, tokenHash(nextRefresh), expiresAt, userAgent, ipAddress); err != nil {
		return AuthResponse{}, fmt.Errorf("create rotated refresh session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at=now(), replaced_by=$2, last_used_at=now() WHERE id=$1`,
		sessionID, newSessionID); err != nil {
		return AuthResponse{}, fmt.Errorf("revoke rotated refresh session: %w", err)
	}
	auditID, err := newUUID()
	if err != nil {
		return AuthResponse{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id, actor_id, action, target_type, target_id, request_id)
		VALUES ($1,$2,'auth.refresh','user',$2,$3)`, auditID, userID, requestID); err != nil {
		return AuthResponse{}, fmt.Errorf("record refresh audit event: %w", err)
	}
	accessExpires := time.Now().UTC().Add(15 * time.Minute)
	access, err := signAccessToken(deriveKey(service.config.AppSecret, "access-token"), accessClaims{
		Subject: userID, Session: newSessionID, Issuer: "zaminflow-api", Audience: "zaminflow-clients",
		IssuedAt: time.Now().UTC().Unix(), Expires: accessExpires.Unix(), EnrollmentOnly: mfaRequired && !mfaEnabled,
	})
	if err != nil {
		return AuthResponse{}, err
	}
	user, err := queryUserTx(ctx, tx, userID)
	if err != nil {
		return AuthResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AuthResponse{}, fmt.Errorf("commit rotated refresh session: %w", err)
	}
	return AuthResponse{
		AccessToken: access, ExpiresAt: accessExpires, User: user,
		SessionID: newSessionID, RefreshToken: nextRefresh, RefreshExpiresAt: expiresAt,
	}, nil
}

func (service *Service) RevokeSession(ctx context.Context, principal Principal) error {
	if _, err := service.pool.Exec(ctx, `
		UPDATE sessions s SET revoked_at=COALESCE(revoked_at,now())
		WHERE s.family_id=(SELECT family_id FROM sessions WHERE id=$1 AND user_id=$2)`,
		principal.SessionID, principal.ID); err != nil {
		return fmt.Errorf("revoke session family: %w", err)
	}
	return nil
}

func (service *Service) VerifyEmail(ctx context.Context, token string) error {
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email verification: %w", err)
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE one_time_tokens SET consumed_at=now()
		WHERE token_hash=$1 AND purpose='email_verification' AND consumed_at IS NULL AND expires_at>now()
		RETURNING user_id::text`, tokenHash(token)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return fmt.Errorf("consume email verification token: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET email_verified_at=COALESCE(email_verified_at,now()), updated_at=now() WHERE id=$1`, userID); err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	var applicant User
	if err := tx.QueryRow(ctx, `
		SELECT id::text,role,account_status,email,(email_verified_at IS NOT NULL),
		       full_name,phone,organization_name,region_code,stir,rejection_reason,created_at,updated_at
		FROM users WHERE id=$1`, userID).Scan(
		&applicant.ID, &applicant.Role, &applicant.AccountStatus, &applicant.Email,
		&applicant.EmailVerified, &applicant.FullName, &applicant.Phone, &applicant.OrganizationName,
		&applicant.RegionCode, &applicant.STIR, &applicant.RejectionReason, &applicant.CreatedAt, &applicant.UpdatedAt); err != nil {
		return fmt.Errorf("load verified applicant: %w", err)
	}
	if err := notifyAdmins(ctx, tx, applicant, service.config.WebBaseURL); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) RequestEmailVerification(ctx context.Context, email string) error {
	normalized := normalizeEmail(email)
	var userID, token string
	err := service.pool.QueryRow(ctx, `
		SELECT id::text FROM users WHERE email=$1 AND email_verified_at IS NULL`, normalized).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find email verification account: %w", err)
	}
	token, err = randomToken(32)
	if err != nil {
		return err
	}
	tokenID, err := newUUID()
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email verification request: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE one_time_tokens SET consumed_at=now() WHERE user_id=$1 AND purpose='email_verification' AND consumed_at IS NULL`, userID); err != nil {
		return fmt.Errorf("invalidate previous verification tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO one_time_tokens (id,user_id,token_hash,purpose,expires_at)
		VALUES ($1,$2,$3,'email_verification',now()+interval '24 hours')`, tokenID, userID, tokenHash(token)); err != nil {
		return fmt.Errorf("store email verification token: %w", err)
	}
	var destination string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&destination); err != nil {
		return fmt.Errorf("load verification recipient: %w", err)
	}
	link := service.config.WebBaseURL + "/verify-email?token=" + url.QueryEscape(token)
	if err := enqueueMail(ctx, tx, destination, "Verify your ZaminFlow email", "Verify your email address using this link:\n\n"+link); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) RequestPasswordReset(ctx context.Context, email string) error {
	var userID, destination string
	err := service.pool.QueryRow(ctx, `SELECT id::text,email FROM users WHERE email=$1`, normalizeEmail(email)).Scan(&userID, &destination)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find password reset account: %w", err)
	}
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	tokenID, err := newUUID()
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password reset: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE one_time_tokens SET consumed_at=now() WHERE user_id=$1 AND purpose='password_reset' AND consumed_at IS NULL`, userID); err != nil {
		return fmt.Errorf("invalidate previous reset tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO one_time_tokens (id,user_id,token_hash,purpose,expires_at)
		VALUES ($1,$2,$3,'password_reset',now()+interval '30 minutes')`, tokenID, userID, tokenHash(token)); err != nil {
		return fmt.Errorf("store password reset token: %w", err)
	}
	link := service.config.WebBaseURL + "/reset-password?token=" + url.QueryEscape(token)
	if err := enqueueMail(ctx, tx, destination, "Reset your ZaminFlow password", "Use this link to reset your password:\n\n"+link); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (service *Service) ConfirmPasswordReset(ctx context.Context, token, password string) error {
	if err := validatePassword(password); err != nil {
		return ErrInvalidInput
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password reset confirmation: %w", err)
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `
		UPDATE one_time_tokens SET consumed_at=now()
		WHERE token_hash=$1 AND purpose='password_reset' AND consumed_at IS NULL AND expires_at>now()
		RETURNING user_id::text`, tokenHash(token)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return fmt.Errorf("consume password reset token: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, failed_login_attempts=0, locked_until=NULL, updated_at=now() WHERE id=$1`, userID, hash); err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, userID); err != nil {
		return fmt.Errorf("revoke sessions after password reset: %w", err)
	}
	return tx.Commit(ctx)
}

func (service *Service) UpdatePassword(ctx context.Context, principal Principal, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return ErrInvalidInput
	}
	var encoded string
	if err := service.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, principal.ID).Scan(&encoded); err != nil {
		return fmt.Errorf("load current password hash: %w", err)
	}
	valid, err := verifyPassword(encoded, oldPassword)
	if err != nil {
		return fmt.Errorf("verify current password: %w", err)
	}
	if !valid {
		return ErrInvalidCredentials
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password change: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, principal.ID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1 AND id<>$2`, principal.ID, principal.SessionID); err != nil {
		return fmt.Errorf("revoke other sessions after password change: %w", err)
	}
	return tx.Commit(ctx)
}

func (service *Service) Me(ctx context.Context, principal Principal) (User, error) {
	return service.loadUser(ctx, principal.ID)
}

func (service *Service) UpdateMe(ctx context.Context, principal Principal, input ProfileUpdate) (User, error) {
	current, err := service.loadUser(ctx, principal.ID)
	if err != nil {
		return User{}, err
	}
	fullName := current.FullName
	phone := valueOrEmpty(current.Phone)
	organization := valueOrEmpty(current.OrganizationName)
	region := valueOrEmpty(current.RegionCode)
	stir := valueOrEmpty(current.STIR)
	if input.FullName != nil {
		fullName = strings.TrimSpace(*input.FullName)
	}
	if input.Phone != nil {
		phone = strings.TrimSpace(*input.Phone)
	}
	if input.OrganizationName != nil {
		organization = strings.TrimSpace(*input.OrganizationName)
	}
	if input.RegionCode != nil {
		region = strings.TrimSpace(*input.RegionCode)
	}
	if input.STIR != nil {
		stir = strings.TrimSpace(*input.STIR)
	}
	if !validText(fullName, 2, 160) || !validPhone(phone) ||
		!validText(organization, 2, 200) || !validSTIR(stir) || !validText(region, 1, 32) {
		return User{}, ErrInvalidInput
	}
	if _, err := service.pool.Exec(ctx, `
		UPDATE users SET full_name=$2, phone=$3, organization_name=$4, region_code=$5, stir=$6, updated_at=now()
		WHERE id=$1`, principal.ID, fullName, phone, organization, region, stir); err != nil {
		if isForeignKeyViolation(err) {
			return User{}, ErrInvalidInput
		}
		return User{}, fmt.Errorf("update profile: %w", err)
	}
	return service.loadUser(ctx, principal.ID)
}

func (service *Service) Resubmit(ctx context.Context, principal Principal, requestID string) (User, error) {
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin application resubmission: %w", err)
	}
	defer tx.Rollback(ctx)
	var oldStatus string
	if err := tx.QueryRow(ctx, `SELECT account_status FROM users WHERE id=$1 FOR UPDATE`, principal.ID).Scan(&oldStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("load applicant status: %w", err)
	}
	if oldStatus != "rejected" {
		return User{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET account_status='pending', rejection_reason=NULL, updated_at=now() WHERE id=$1`, principal.ID); err != nil {
		return User{}, fmt.Errorf("reset applicant status: %w", err)
	}
	var adminID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE role='admin' ORDER BY id LIMIT 1`).Scan(&adminID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, fmt.Errorf("cannot resubmit application: no admin account exists")
		}
		return User{}, fmt.Errorf("find admin for resubmission record: %w", err)
	}
	reviewID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_reviews (id,user_id,reviewer_id,decision,reason)
		VALUES ($1,$2,$3,'resubmitted','Applicant corrected details and requested a new review')`,
		reviewID, principal.ID, adminID); err != nil {
		return User{}, fmt.Errorf("record resubmission: %w", err)
	}
	applicant, err := queryUserTx(ctx, tx, principal.ID)
	if err != nil {
		return User{}, err
	}
	if applicant.EmailVerified {
		if err := notifyAdmins(ctx, tx, applicant, service.config.WebBaseURL); err != nil {
			return User{}, err
		}
	}
	auditID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id,actor_id,action,target_type,target_id,request_id)
		VALUES ($1,$2,'account.resubmitted','user',$2,$3)`, auditID, principal.ID, requestID); err != nil {
		return User{}, fmt.Errorf("record resubmission audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit resubmission: %w", err)
	}
	return service.loadUser(ctx, principal.ID)
}

func queryUserTx(ctx context.Context, tx pgx.Tx, userID string) (User, error) {
	return scanUser(tx.QueryRow(ctx, `
		SELECT id::text,role,account_status,email,(email_verified_at IS NOT NULL),
		       full_name,phone,organization_name,region_code,stir,rejection_reason,created_at,updated_at
		FROM users WHERE id=$1`, userID))
}

type applicationReview struct {
	ID         string    `json:"id"`
	ReviewerID string    `json:"reviewerId"`
	Decision   string    `json:"decision"`
	Reason     *string   `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (service *Service) RequestDeletion(ctx context.Context, principal Principal, requestID string) error {
	if principal.Role == "admin" {
		return ErrForbidden
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin account deletion request: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT account_status FROM users WHERE id=$1 FOR UPDATE`, principal.ID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load account for deletion request: %w", err)
	}
	var activeOrders bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM orders
			WHERE (farmer_id=$1 OR processor_id=$1)
			  AND status IN ('placed','accepted','ready_for_pickup','picked_up','delivered','disputed')
		)`, principal.ID).Scan(&activeOrders); err != nil {
		return fmt.Errorf("check active orders before deletion request: %w", err)
	}
	if activeOrders {
		return ErrConflict
	}
	var pending bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM account_deletion_requests WHERE user_id=$1 AND status IN ('requested','approved'))`,
		principal.ID).Scan(&pending); err != nil {
		return fmt.Errorf("check pending deletion request: %w", err)
	}
	if pending {
		return ErrConflict
	}
	id, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO account_deletion_requests (id,user_id) VALUES ($1,$2)`, id, principal.ID); err != nil {
		return fmt.Errorf("create account deletion request: %w", err)
	}
	auditID, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id,actor_id,action,target_type,target_id,request_id,metadata)
		VALUES ($1,$2,'account.deletion_requested','user',$2,$3,jsonb_build_object('accountStatus',$4::text))`,
		auditID, principal.ID, requestID, status); err != nil {
		return fmt.Errorf("record account deletion request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account deletion request: %w", err)
	}
	return nil
}

func (service *Service) GetOwnDeletionRequest(ctx context.Context, principal Principal) (OwnDeletionRequest, error) {
	if principal.Role != "farmer" && principal.Role != "processor" {
		return OwnDeletionRequest{}, ErrForbidden
	}
	var request OwnDeletionRequest
	err := service.pool.QueryRow(ctx, `
		SELECT id::text,status,requested_at,reviewed_at,reason
		FROM account_deletion_requests WHERE user_id=$1
		ORDER BY requested_at DESC,id DESC LIMIT 1`, principal.ID).Scan(
		&request.ID, &request.Status, &request.RequestedAt, &request.ReviewedAt, &request.Reason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return OwnDeletionRequest{}, ErrNotFound
	}
	if err != nil {
		return OwnDeletionRequest{}, fmt.Errorf("load own account deletion request: %w", err)
	}
	return request, nil
}

func (service *Service) ListDeletionRequests(ctx context.Context, status string, pageNumber, pageSize int) (any, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "":
		status = "requested"
	case "all":
		status = ""
	case "requested", "approved", "completed", "rejected":
	default:
		return nil, ErrInvalidQuery
	}
	if _, _, err := boundedPage(pageNumber, pageSize); err != nil {
		return nil, err
	}
	const condition = "($1='' OR status=$1)"
	var total int64
	if err := service.pool.QueryRow(ctx,
		`SELECT count(*) FROM account_deletion_requests WHERE `+condition, status).Scan(&total); err != nil {
		return nil, fmt.Errorf("count account deletion requests: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT d.id::text,d.status,d.requested_at,d.reviewed_by::text,d.reviewed_at,d.reason,
		       u.id::text,u.role,u.account_status,u.email,(u.email_verified_at IS NOT NULL),
		       u.full_name,u.phone,u.organization_name,u.region_code,u.stir,u.rejection_reason,
		       u.created_at,u.updated_at
		FROM account_deletion_requests d
		JOIN users u ON u.id=d.user_id
		WHERE ($1='' OR d.status=$1)
		ORDER BY d.requested_at ASC,d.id ASC LIMIT $2 OFFSET $3`,
		status, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list account deletion requests: %w", err)
	}
	defer rows.Close()
	items := make([]AccountDeletionRequest, 0)
	for rows.Next() {
		var item AccountDeletionRequest
		if err := rows.Scan(
			&item.ID, &item.Status, &item.RequestedAt, &item.ReviewedBy, &item.ReviewedAt, &item.ReviewReason,
			&item.User.ID, &item.User.Role, &item.User.AccountStatus, &item.User.Email, &item.User.EmailVerified,
			&item.User.FullName, &item.User.Phone, &item.User.OrganizationName, &item.User.RegionCode, &item.User.STIR,
			&item.User.RejectionReason, &item.User.CreatedAt, &item.User.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account deletion request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account deletion requests: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

func (service *Service) ReviewDeletionRequest(ctx context.Context, reviewer Principal, requestID, decision, reason, auditRequestID string) (AccountDeletionRequest, error) {
	if reviewer.Role != "admin" || !reviewer.MFAEnabled || reviewer.EnrollmentOnly {
		return AccountDeletionRequest{}, ErrForbidden
	}
	if !isUUID(requestID) {
		return AccountDeletionRequest{}, ErrInvalidQuery
	}
	reason = strings.TrimSpace(reason)
	if decision != "approved" && decision != "rejected" {
		return AccountDeletionRequest{}, ErrInvalidInput
	}
	if decision == "rejected" && !validText(reason, 1, 2000) {
		return AccountDeletionRequest{}, ErrInvalidInput
	}
	if reason != "" && !validText(reason, 1, 2000) {
		return AccountDeletionRequest{}, ErrInvalidInput
	}

	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return AccountDeletionRequest{}, fmt.Errorf("begin account deletion review: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID, status string
	if err := tx.QueryRow(ctx, `
		SELECT user_id::text,status FROM account_deletion_requests WHERE id=$1 FOR UPDATE`,
		requestID).Scan(&userID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountDeletionRequest{}, ErrNotFound
		}
		return AccountDeletionRequest{}, fmt.Errorf("load account deletion request for review: %w", err)
	}
	if status != "requested" {
		return AccountDeletionRequest{}, ErrConflict
	}
	var lockedUserID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&lockedUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountDeletionRequest{}, ErrNotFound
		}
		return AccountDeletionRequest{}, fmt.Errorf("lock account for deletion review: %w", err)
	}
	if decision == "approved" {
		var activeOrders bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM orders
				WHERE (farmer_id=$1 OR processor_id=$1)
				  AND status IN ('placed','accepted','ready_for_pickup','picked_up','delivered','disputed')
			)`, userID).Scan(&activeOrders); err != nil {
			return AccountDeletionRequest{}, fmt.Errorf("check active orders before approving deletion request: %w", err)
		}
		if activeOrders {
			return AccountDeletionRequest{}, ErrConflict
		}
	}
	var item AccountDeletionRequest
	if err := tx.QueryRow(ctx, `
		UPDATE account_deletion_requests
		SET status=$2,reviewed_by=$3,reviewed_at=now(),reason=NULLIF($4,'')
		WHERE id=$1
		RETURNING id::text,status,requested_at,reviewed_by::text,reviewed_at,reason`,
		requestID, decision, reviewer.ID, reason).Scan(
		&item.ID, &item.Status, &item.RequestedAt, &item.ReviewedBy, &item.ReviewedAt, &item.ReviewReason,
	); err != nil {
		return AccountDeletionRequest{}, fmt.Errorf("record account deletion review: %w", err)
	}
	item.User, err = queryUserTx(ctx, tx, userID)
	if err != nil {
		return AccountDeletionRequest{}, err
	}
	auditID, err := newUUID()
	if err != nil {
		return AccountDeletionRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id,actor_id,action,target_type,target_id,request_id,metadata)
		VALUES ($1,$2,$3,'account_deletion_request',$4,$5,
			jsonb_build_object('userId',$6::text,'decision',$7::text,'reason',NULLIF($8::text,'')))`,
		auditID, reviewer.ID, "account.deletion_"+decision, requestID, auditRequestID, userID, decision, reason); err != nil {
		return AccountDeletionRequest{}, fmt.Errorf("record account deletion review audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return AccountDeletionRequest{}, fmt.Errorf("commit account deletion review: %w", err)
	}
	return item, nil
}

func (service *Service) ListApplications(ctx context.Context, role, regionCode string, pageNumber, pageSize int) (any, error) {
	if role != "" && role != "farmer" && role != "processor" {
		return nil, ErrInvalidQuery
	}
	role = strings.TrimSpace(role)
	regionCode = strings.TrimSpace(regionCode)
	if len(regionCode) > 32 {
		return nil, ErrInvalidQuery
	}
	var total int64
	if err := service.pool.QueryRow(ctx, `
		SELECT count(*) FROM users
		WHERE account_status='pending' AND email_verified_at IS NOT NULL
		  AND ($1='' OR role=$1) AND ($2='' OR region_code=$2)`,
		role, regionCode).Scan(&total); err != nil {
		return nil, fmt.Errorf("count pending applications: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT id::text,role,account_status,email,(email_verified_at IS NOT NULL),
		       full_name,phone,organization_name,region_code,stir,rejection_reason,created_at,updated_at
		FROM users
		WHERE account_status='pending' AND email_verified_at IS NOT NULL
		  AND ($1='' OR role=$1) AND ($2='' OR region_code=$2)
		ORDER BY created_at ASC,id ASC LIMIT $3 OFFSET $4`,
		role, regionCode, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list pending applications: %w", err)
	}
	defer rows.Close()
	items := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending application: %w", err)
		}
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending applications: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

func (service *Service) GetApplication(ctx context.Context, userID string) (any, error) {
	if !isUUID(userID) {
		return nil, ErrInvalidQuery
	}
	user, err := service.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Role == "admin" {
		return nil, ErrNotFound
	}
	rows, err := service.pool.Query(ctx, `
		SELECT id::text,reviewer_id::text,decision,reason,created_at
		FROM account_reviews WHERE user_id=$1 ORDER BY created_at DESC,id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("load application review history: %w", err)
	}
	defer rows.Close()
	reviews := make([]applicationReview, 0)
	for rows.Next() {
		var review applicationReview
		if err := rows.Scan(&review.ID, &review.ReviewerID, &review.Decision, &review.Reason, &review.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan application review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application review history: %w", err)
	}
	return map[string]any{"user": user, "reviews": reviews}, nil
}

func (service *Service) ReviewAccount(ctx context.Context, reviewer Principal, userID, decision, reason, requestID string) (User, error) {
	if !isUUID(userID) {
		return User{}, ErrInvalidQuery
	}
	reason = strings.TrimSpace(reason)
	if (decision == "rejected" || decision == "suspended") && !validText(reason, 1, 2000) {
		return User{}, ErrInvalidInput
	}
	if reason != "" && !validText(reason, 1, 2000) {
		return User{}, ErrInvalidInput
	}
	if reviewer.ID == userID {
		return User{}, ErrForbidden
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin account review: %w", err)
	}
	defer tx.Rollback(ctx)
	var target User
	err = tx.QueryRow(ctx, `
		SELECT id::text,role,account_status,email,(email_verified_at IS NOT NULL),
		       full_name,phone,organization_name,region_code,stir,rejection_reason,created_at,updated_at
		FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(
		&target.ID, &target.Role, &target.AccountStatus, &target.Email, &target.EmailVerified,
		&target.FullName, &target.Phone, &target.OrganizationName, &target.RegionCode, &target.STIR,
		&target.RejectionReason, &target.CreatedAt, &target.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load account for review: %w", err)
	}
	if target.Role == "admin" {
		return User{}, ErrForbidden
	}
	var nextStatus string
	switch decision {
	case "approved", "rejected":
		if target.AccountStatus != "pending" || !target.EmailVerified {
			return User{}, ErrConflict
		}
		nextStatus = decision
	case "suspended":
		if target.AccountStatus != "approved" && target.AccountStatus != "pending" {
			return User{}, ErrConflict
		}
		nextStatus = "suspended"
	case "reinstated":
		if target.AccountStatus != "suspended" {
			return User{}, ErrConflict
		}
		nextStatus = "approved"
	default:
		return User{}, ErrInvalidInput
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET account_status=$2,rejection_reason=$3,updated_at=now() WHERE id=$1`,
		userID, nextStatus, nullableRejectionReason(decision, reason)); err != nil {
		return User{}, fmt.Errorf("apply account review decision: %w", err)
	}
	reviewID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_reviews (id,user_id,reviewer_id,decision,reason)
		VALUES ($1,$2,$3,$4,NULLIF($5,''))`, reviewID, userID, reviewer.ID, decision, reason); err != nil {
		return User{}, fmt.Errorf("record account review: %w", err)
	}
	if nextStatus != "approved" {
		if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, userID); err != nil {
			return User{}, fmt.Errorf("revoke sessions after account restriction: %w", err)
		}
	}
	if target.EmailVerified {
		subject := "Your ZaminFlow account status has changed"
		body := fmt.Sprintf("Your account status is now %s.", nextStatus)
		if reason != "" {
			body += "\nReason: " + reason
		}
		if err := enqueueMail(ctx, tx, target.Email, subject, body); err != nil {
			return User{}, err
		}
	}
	auditID, err := newUUID()
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (id,actor_id,action,target_type,target_id,request_id,metadata)
		VALUES ($1,$2,$3,'user',$4,$5,jsonb_build_object('decision',$6::text,'reason',NULLIF($7::text,'')))`,
		auditID, reviewer.ID, "account."+decision, userID, requestID, decision, reason); err != nil {
		return User{}, fmt.Errorf("record account review audit event: %w", err)
	}
	target, err = queryUserTx(ctx, tx, userID)
	if err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit account review: %w", err)
	}
	return target, nil
}

func nullableRejectionReason(decision, reason string) *string {
	if decision == "rejected" {
		return &reason
	}
	return nil
}

func (service *Service) ListAuditEvents(ctx context.Context, filters url.Values, pageNumber, pageSize int) (any, error) {
	actorID, targetID := strings.TrimSpace(filters.Get("actorId")), strings.TrimSpace(filters.Get("targetId"))
	if actorID != "" && !isUUID(actorID) || targetID != "" && !isUUID(targetID) {
		return nil, ErrInvalidQuery
	}
	action, targetType := strings.TrimSpace(filters.Get("action")), strings.TrimSpace(filters.Get("targetType"))
	if len(action) > 80 || len(targetType) > 40 {
		return nil, ErrInvalidQuery
	}
	from, to := strings.TrimSpace(filters.Get("from")), strings.TrimSpace(filters.Get("to"))
	if from != "" {
		if _, err := time.Parse(time.RFC3339, from); err != nil {
			return nil, ErrInvalidQuery
		}
	}
	if to != "" {
		if _, err := time.Parse(time.RFC3339, to); err != nil {
			return nil, ErrInvalidQuery
		}
	}
	const condition = `
		($1='' OR actor_id=$1::uuid) AND ($2='' OR action=$2)
		AND ($3='' OR target_type=$3) AND ($4='' OR target_id=$4::uuid)
		AND ($5='' OR created_at >= $5::timestamptz) AND ($6='' OR created_at < $6::timestamptz)`
	var total int64
	if err := service.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE `+condition,
		actorID, action, targetType, targetID, from, to).Scan(&total); err != nil {
		return nil, fmt.Errorf("count audit events: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT id::text,actor_id::text,action,target_type,target_id::text,request_id,metadata,created_at
		FROM audit_events WHERE `+condition+`
		ORDER BY created_at DESC,id DESC LIMIT $7 OFFSET $8`,
		actorID, action, targetType, targetID, from, to, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, actionName, targetName, reqID string
		var actor, target *string
		var metadata []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &actor, &actionName, &targetName, &target, &reqID, &metadata, &createdAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		var decoded any
		if err := json.Unmarshal(metadata, &decoded); err != nil {
			return nil, fmt.Errorf("decode audit event metadata: %w", err)
		}
		items = append(items, map[string]any{
			"id": id, "actorId": actor, "action": actionName, "targetType": targetName,
			"targetId": target, "requestId": reqID, "metadata": decoded, "createdAt": createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

func (service *Service) ListSessions(ctx context.Context, principal Principal, pageNumber, pageSize int) (any, error) {
	var total int64
	if err := service.pool.QueryRow(ctx, `
		SELECT count(*) FROM sessions WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now()`,
		principal.ID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}
	rows, err := service.pool.Query(ctx, `
		SELECT id::text, created_at, last_used_at FROM sessions
		WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now()
		ORDER BY created_at DESC,id LIMIT $2 OFFSET $3`, principal.ID, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id string
		var createdAt, lastUsedAt time.Time
		if err := rows.Scan(&id, &createdAt, &lastUsedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		items = append(items, map[string]any{"id": id, "createdAt": createdAt, "lastUsedAt": lastUsedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total}, nil
}

func (service *Service) RevokeOwnSession(ctx context.Context, principal Principal, sessionID string) error {
	tag, err := service.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1 AND user_id=$2`,
		sessionID, principal.ID)
	if err != nil {
		return fmt.Errorf("revoke owned session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (service *Service) BeginTOTP(ctx context.Context, principal Principal) (map[string]string, error) {
	if principal.Role != "admin" {
		return nil, ErrForbidden
	}
	secret, err := newTOTPSecret()
	if err != nil {
		return nil, err
	}
	encrypted, err := encryptSecret(service.config.AppSecret, secret)
	if err != nil {
		return nil, err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin TOTP enrollment: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE users SET admin_totp_secret=$2
		WHERE id=$1 AND role='admin' AND admin_mfa_enabled_at IS NULL`, principal.ID, encrypted)
	if err != nil {
		return nil, fmt.Errorf("store pending TOTP secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrConflict
	}
	if err := insertAudit(ctx, tx, principal.ID, "admin.mfa_setup_started", "user", principal.ID, "auth/totp/setup", nil); err != nil {
		return nil, err
	}
	user, err := mailAddress(ctx, service.pool, principal.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit TOTP enrollment setup: %w", err)
	}
	uri := "otpauth://totp/ZaminFlow:" + url.QueryEscape(user) + "?secret=" + secret + "&issuer=ZaminFlow&algorithm=SHA1&digits=6&period=30"
	return map[string]string{"secret": secret, "otpauthUri": uri}, nil
}

func (service *Service) ConfirmTOTP(ctx context.Context, principal Principal, code string) ([]string, error) {
	if principal.Role != "admin" || len(code) != 6 {
		return nil, ErrForbidden
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin TOTP confirmation: %w", err)
	}
	defer tx.Rollback(ctx)
	var ciphertext []byte
	err = tx.QueryRow(ctx, `SELECT admin_totp_secret FROM users WHERE id=$1 AND role='admin' AND admin_mfa_enabled_at IS NULL FOR UPDATE`, principal.ID).Scan(&ciphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("load pending TOTP secret: %w", err)
	}
	secret, err := decryptSecret(service.config.AppSecret, ciphertext)
	if err != nil {
		return nil, err
	}
	if !verifyTOTP(secret, code, time.Now()) {
		return nil, ErrInvalidCredentials
	}
	codes := make([]string, 10)
	for index := range codes {
		raw, err := randomToken(9)
		if err != nil {
			return nil, err
		}
		codes[index] = strings.ToUpper(strings.TrimRight(strings.ReplaceAll(raw, "-", ""), "="))
		codeID, err := newUUID()
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO admin_recovery_codes (id,user_id,code_hash) VALUES ($1,$2,$3)`,
			codeID, principal.ID, recoveryCodeHash(service.config.AppSecret, codes[index])); err != nil {
			return nil, fmt.Errorf("store admin recovery code: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET admin_mfa_enabled_at=now() WHERE id=$1`, principal.ID); err != nil {
		return nil, fmt.Errorf("enable admin TOTP: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, principal.ID); err != nil {
		return nil, fmt.Errorf("revoke pre-enrollment admin sessions: %w", err)
	}
	if err := insertAudit(ctx, tx, principal.ID, "admin.mfa_enabled", "user", principal.ID, "auth/totp/confirm", nil); err != nil {
		return nil, err
	}
	return codes, tx.Commit(ctx)
}

func (service *Service) RotateRecoveryCodes(ctx context.Context, principal Principal, code string) ([]string, error) {
	valid, err := service.verifyAdminTOTP(ctx, principal.ID, code)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin recovery-code rotation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE admin_recovery_codes SET used_at=COALESCE(used_at,now()) WHERE user_id=$1 AND used_at IS NULL`, principal.ID); err != nil {
		return nil, fmt.Errorf("invalidate prior recovery codes: %w", err)
	}
	codes := make([]string, 10)
	for index := range codes {
		raw, err := randomToken(9)
		if err != nil {
			return nil, err
		}
		codes[index] = strings.ToUpper(strings.TrimRight(strings.ReplaceAll(raw, "-", ""), "="))
		codeID, err := newUUID()
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO admin_recovery_codes (id,user_id,code_hash) VALUES ($1,$2,$3)`,
			codeID, principal.ID, recoveryCodeHash(service.config.AppSecret, codes[index])); err != nil {
			return nil, fmt.Errorf("store replacement recovery code: %w", err)
		}
	}
	if err := insertAudit(ctx, tx, principal.ID, "admin.recovery_codes_rotated", "user", principal.ID, "auth/totp/recovery-codes/rotate", nil); err != nil {
		return nil, err
	}
	return codes, tx.Commit(ctx)
}

func (service *Service) verifyAdminTOTP(ctx context.Context, userID, code string) (bool, error) {
	var ciphertext []byte
	err := service.pool.QueryRow(ctx, `SELECT admin_totp_secret FROM users WHERE id=$1 AND admin_mfa_enabled_at IS NOT NULL`, userID).Scan(&ciphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrForbidden
	}
	if err != nil {
		return false, fmt.Errorf("load enabled TOTP secret: %w", err)
	}
	secret, err := decryptSecret(service.config.AppSecret, ciphertext)
	if err != nil {
		return false, err
	}
	return verifyTOTP(secret, code, time.Now()), nil
}

func (service *Service) consumeRecoveryCode(ctx context.Context, userID, code string) (bool, error) {
	tag, err := service.pool.Exec(ctx, `
		UPDATE admin_recovery_codes SET used_at=now()
		WHERE id=(SELECT id FROM admin_recovery_codes
		          WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL
		          ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED)`,
		userID, recoveryCodeHash(service.config.AppSecret, code))
	if err != nil {
		return false, fmt.Errorf("consume admin recovery code: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (service *Service) CreateAdmin(ctx context.Context, email, password, fullName string) error {
	email = normalizeEmail(email)
	if err := validatePassword(password); err != nil {
		return ErrInvalidInput
	}
	if !validEmail(email) || !validText(strings.TrimSpace(fullName), 2, 160) {
		return ErrInvalidInput
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	id, err := newUUID()
	if err != nil {
		return err
	}
	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin admin provisioning: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id,role,account_status,email,password_hash,email_verified_at,full_name,admin_mfa_required)
		VALUES ($1,'admin','approved',$2,$3,now(),$4,true)`, id, email, hash, strings.TrimSpace(fullName))
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create admin account: %w", err)
	}
	if err := insertAudit(ctx, tx, id, "admin.created", "user", id, "cmd/admin", nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit admin provisioning: %w", err)
	}
	return nil
}

func (service *Service) loadUser(ctx context.Context, userID string) (User, error) {
	row := service.pool.QueryRow(ctx, `
		SELECT id::text, role, account_status, email, (email_verified_at IS NOT NULL),
		       full_name, phone, organization_name, region_code, stir, rejection_reason,
		       created_at, updated_at
		FROM users WHERE id=$1`, userID)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load user profile: %w", err)
	}
	return user, nil
}

type scanner interface {
	Scan(...any) error
}

func scanUser(row scanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Role, &user.AccountStatus, &user.Email, &user.EmailVerified,
		&user.FullName, &user.Phone, &user.OrganizationName, &user.RegionCode, &user.STIR,
		&user.RejectionReason, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func validateSignUp(input SignUpInput) (SignUpInput, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Email = normalizeEmail(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	input.OrganizationName = strings.TrimSpace(input.OrganizationName)
	input.RegionCode = strings.TrimSpace(input.RegionCode)
	input.STIR = strings.TrimSpace(input.STIR)
	if input.Role != "farmer" && input.Role != "processor" {
		return SignUpInput{}, ErrInvalidInput
	}
	if !validText(input.FullName, 2, 160) || !validEmail(input.Email) ||
		!validPhone(input.Phone) || !validText(input.OrganizationName, 2, 200) ||
		!validText(input.RegionCode, 1, 32) || !validSTIR(input.STIR) {
		return SignUpInput{}, ErrInvalidInput
	}
	if err := validatePassword(input.Password); err != nil {
		return SignUpInput{}, ErrInvalidInput
	}
	return input, nil
}

func validText(value string, min, max int) bool {
	if !utf8.ValidString(value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	return length >= min && length <= max && strings.TrimSpace(value) == value
}

func validEmail(value string) bool {
	if len(value) > 254 {
		return false
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value && strings.Count(value, "@") == 1
}

func validPhone(value string) bool {
	if len(value) != 13 || !strings.HasPrefix(value, "+998") {
		return false
	}
	for _, char := range value[4:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func validSTIR(value string) bool {
	if len(value) != 9 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23503"
}

func enqueueMail(ctx context.Context, tx pgx.Tx, recipient, subject, body string) error {
	id, err := newUUID()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO mail_outbox (id,recipient,subject,body) VALUES ($1,$2,$3,$4)`,
		id, recipient, subject, body); err != nil {
		return fmt.Errorf("enqueue email notification: %w", err)
	}
	return nil
}

func notifyAdmins(ctx context.Context, tx pgx.Tx, applicant User, webBaseURL string) error {
	rows, err := tx.Query(ctx, `SELECT email FROM users WHERE role='admin' AND admin_mfa_enabled_at IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("find admins for application notification: %w", err)
	}
	emails := make([]string, 0)
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			rows.Close()
			return fmt.Errorf("scan admin notification recipient: %w", err)
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate admin notification recipients: %w", err)
	}
	rows.Close()
	for _, email := range emails {
		body := fmt.Sprintf("A new verified %s application from %s is awaiting review.\nReview queue: %s/admin/applications",
			applicant.Role, applicant.FullName, webBaseURL)
		if err := enqueueMail(ctx, tx, email, "New ZaminFlow application", body); err != nil {
			return err
		}
	}
	return nil
}

func mailAddress(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var email string
	if err := pool.QueryRow(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&email); err != nil {
		return "", fmt.Errorf("load TOTP account email: %w", err)
	}
	return email, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newUUID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func clientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return ""
	}
	return host
}

func boundedPage(pageNumber, pageSize int) (int, int, error) {
	if pageNumber < 1 || pageNumber > 1_000_000 || pageSize < 1 || pageSize > 100 {
		return 0, 0, ErrInvalidQuery
	}
	return pageNumber, pageSize, nil
}

func parsePage(query url.Values) (int, int, error) {
	pageNumber, pageSize := 1, 20
	var err error
	if raw := query.Get("page"); raw != "" {
		pageNumber, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, ErrInvalidQuery
		}
	}
	if raw := query.Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, ErrInvalidQuery
		}
	}
	return boundedPage(pageNumber, pageSize)
}

func parseMetadata(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func auditMetadata(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return encoded
}

func validIP(value string) string {
	if net.ParseIP(value) == nil {
		return ""
	}
	return value
}
