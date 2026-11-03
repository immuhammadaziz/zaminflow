package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zaminflow/zaminflow-api/internal/config"
)

type handler struct {
	service *Service
	config  config.Config
	logger  *slog.Logger
}

type apiErrorResponse struct {
	Error struct {
		Code      string          `json:"code"`
		Message   string          `json:"message"`
		RequestID string          `json:"requestId"`
		Fields    []apiFieldError `json:"fieldErrors,omitempty"`
	} `json:"error"`
}

type apiFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

var bearerPattern = regexp.MustCompile(`^Bearer ([A-Za-z0-9._-]+)$`)

func RegisterRoutes(router chi.Router, service *Service, cfg config.Config) {
	api := &handler{service: service, config: cfg, logger: service.logger}
	router.Get("/meta/roles", api.roles)
	router.Get("/meta/regions", api.regions)
	router.Post("/auth/sign-up", api.signUp)
	router.Post("/auth/sign-in", api.signIn)
	router.Post("/auth/refresh", api.refresh)
	router.Post("/auth/sign-out", api.withAuth(api.signOut, false, false))
	router.Post("/auth/email-verification/request", api.requestVerification)
	router.Post("/auth/email-verification/confirm", api.confirmVerification)
	router.Post("/auth/password-reset/request", api.requestPasswordReset)
	router.Post("/auth/password-reset/confirm", api.confirmPasswordReset)
	router.Get("/auth/sessions", api.withAuth(api.listSessions, false, false))
	router.Delete("/auth/sessions/{sessionId}", api.withAuth(api.revokeSession, false, false))
	router.Post("/auth/change-password", api.withAuth(api.changePassword, false, false))
	router.Post("/auth/totp/setup", api.withAuth(api.beginTOTP, true, true))
	router.Post("/auth/totp/confirm", api.withAuth(api.confirmTOTP, true, true))
	router.Post("/auth/totp/recovery-codes/rotate", api.withAuth(api.rotateRecoveryCodes, true, false))
	router.Get("/me", api.withAuth(api.me, false, false))
	router.Get("/me/deletion-request", api.withAuth(api.getOwnDeletionRequest, false, false))
	router.Patch("/me", api.withAuth(api.updateMe, false, false))
	router.Post("/me/resubmit", api.withAuth(api.resubmit, false, false))
	router.Delete("/me", api.withAuth(api.deleteMe, false, false))
	router.Get("/admin/deletion-requests", api.withAuth(api.listDeletionRequests, true, false))
	router.Post("/admin/deletion-requests/{requestId}/approve", api.withAuth(api.approveDeletionRequest, true, false))
	router.Post("/admin/deletion-requests/{requestId}/reject", api.withAuth(api.rejectDeletionRequest, true, false))
	router.Get("/admin/applications", api.withAuth(api.listApplications, true, false))
	router.Get("/admin/applications/{userId}", api.withAuth(api.getApplication, true, false))
	router.Post("/admin/applications/{userId}/approve", api.withAuth(api.approve, true, false))
	router.Post("/admin/applications/{userId}/reject", api.withAuth(api.reject, true, false))
	router.Post("/admin/users/{userId}/suspend", api.withAuth(api.suspend, true, false))
	router.Post("/admin/users/{userId}/reinstate", api.withAuth(api.reinstate, true, false))
	router.Get("/admin/audit-events", api.withAuth(api.auditEvents, true, false))
	RegisterMarketplaceRoutes(router, service)
}

func (api *handler) withAuth(next func(http.ResponseWriter, *http.Request, Principal), requireAdmin, allowEnrollment bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		match := bearerPattern.FindStringSubmatch(strings.TrimSpace(r.Header.Get("Authorization")))
		if len(match) != 2 {
			writeAPIError(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
			return
		}
		principal, err := api.service.Authenticate(r.Context(), match[1])
		if err != nil {
			api.writeServiceError(w, r, err)
			return
		}
		if principal.EnrollmentOnly && (!allowEnrollment || principal.MFAEnabled) {
			writeAPIError(w, r, http.StatusForbidden, "admin_mfa_required", "Complete administrator MFA enrollment before continuing.")
			return
		}
		if requireAdmin {
			if principal.Role != "admin" {
				writeAPIError(w, r, http.StatusForbidden, "forbidden", "This action is not permitted.")
				return
			}
			if !principal.MFAEnabled && !allowEnrollment {
				writeAPIError(w, r, http.StatusForbidden, "admin_mfa_required", "Complete administrator MFA enrollment before continuing.")
				return
			}
		}
		next(w, r, principal)
	}
}

func (api *handler) roles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": []map[string]string{
		{"value": "farmer", "label": "Dairy farmer or producer"},
		{"value": "processor", "label": "Dairy industry processor"},
	}})
}

func (api *handler) regions(w http.ResponseWriter, r *http.Request) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	var total int64
	if err := api.service.pool.QueryRow(r.Context(), `SELECT count(*) FROM regions WHERE active`).Scan(&total); err != nil {
		api.writeInternalError(w, r, "count regions", err)
		return
	}
	rows, err := api.service.pool.Query(r.Context(), `
		SELECT code,name_uz,name_ru,name_en FROM regions WHERE active
		ORDER BY code LIMIT $1 OFFSET $2`, pageSize, (pageNumber-1)*pageSize)
	if err != nil {
		api.writeInternalError(w, r, "list regions", err)
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var code, uz, ru, en string
		if err := rows.Scan(&code, &uz, &ru, &en); err != nil {
			api.writeInternalError(w, r, "scan region", err)
			return
		}
		items = append(items, map[string]string{"code": code, "nameUz": uz, "nameRu": ru, "nameEn": en})
	}
	if err := rows.Err(); err != nil {
		api.writeInternalError(w, r, "iterate regions", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": pageNumber, "pageSize": pageSize, "total": total})
}

func (api *handler) signUp(w http.ResponseWriter, r *http.Request) {
	ip := api.requestIP(r)
	if !api.allowRequests(w, r, "sign-up-ip", ip, 20, time.Hour) {
		return
	}
	var input SignUpInput
	if !decodeBody(w, r, &input) {
		return
	}
	user, err := api.service.SignUp(r.Context(), input, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user": user, "message": "Check your email to verify your account before administrator review.",
	})
}

func (api *handler) signIn(w http.ResponseWriter, r *http.Request) {
	ip := api.requestIP(r)
	var input SignInInput
	if !decodeBody(w, r, &input) {
		return
	}
	if !api.allowRequests(w, r, "sign-in-ip", ip, 30, 15*time.Minute) ||
		!api.allowRequests(w, r, "sign-in-email", normalizeEmail(input.Email), 10, 15*time.Minute) {
		return
	}
	response, err := api.service.SignIn(r.Context(), input, parseMetadata(r.UserAgent(), 256), ip, requestID(r))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
			return
		}
		api.writeServiceError(w, r, err)
		return
	}
	if err := api.setRefreshCookie(w, response.RefreshToken, response.RefreshExpiresAt); err != nil {
		api.writeInternalError(w, r, "set refresh cookie", err)
		return
	}
	writeJSON(w, http.StatusOK, authBody(response))
}

func (api *handler) refresh(w http.ResponseWriter, r *http.Request) {
	ip := api.requestIP(r)
	if !api.allowRequests(w, r, "refresh-ip", ip, 60, time.Minute) {
		return
	}
	if !api.validCSRF(w, r) {
		return
	}
	cookie, err := r.Cookie("zf_refresh")
	if err != nil {
		writeAPIError(w, r, http.StatusUnauthorized, "unauthenticated", "A valid refresh session is required.")
		return
	}
	response, err := api.service.Refresh(r.Context(), cookie.Value, parseMetadata(r.UserAgent(), 256), ip, requestID(r))
	if err != nil {
		if errors.Is(err, ErrConflict) {
			api.clearRefreshCookie(w)
			writeAPIError(w, r, http.StatusConflict, "refresh_token_reuse_detected", "The refresh session was revoked. Sign in again.")
			return
		}
		api.writeServiceError(w, r, err)
		return
	}
	if err := api.setRefreshCookie(w, response.RefreshToken, response.RefreshExpiresAt); err != nil {
		api.writeInternalError(w, r, "rotate refresh cookie", err)
		return
	}
	writeJSON(w, http.StatusOK, authBody(response))
}

func (api *handler) requestIP(r *http.Request) string {
	if api.config.AppEnv == "production" {
		forwardedIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
		if ip := net.ParseIP(forwardedIP); ip != nil {
			return ip.String()
		}
	}
	return clientIP(r.RemoteAddr)
}

func (api *handler) signOut(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !api.validCSRF(w, r) {
		return
	}
	if err := api.service.RevokeSession(r.Context(), principal); err != nil {
		api.writeInternalError(w, r, "revoke session", err)
		return
	}
	api.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (api *handler) requestVerification(w http.ResponseWriter, r *http.Request) {
	if !api.allowRequests(w, r, "verification-ip", api.requestIP(r), 20, time.Hour) {
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if !validEmail(normalizeEmail(input.Email)) {
		writeFieldError(w, r, "email", "invalid_format", "Enter a valid email address.")
		return
	}
	if !api.allowRequests(w, r, "verification-email", normalizeEmail(input.Email), 3, time.Hour) {
		return
	}
	if err := api.service.RequestEmailVerification(r.Context(), input.Email); err != nil {
		api.writeInternalError(w, r, "request email verification", err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "If the account needs verification, an email will be sent."})
}

func (api *handler) confirmVerification(w http.ResponseWriter, r *http.Request) {
	if !api.allowRequests(w, r, "verification-confirm-ip", api.requestIP(r), 20, 15*time.Minute) {
		return
	}
	var input struct {
		Token string `json:"token"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if err := api.service.VerifyEmail(r.Context(), input.Token); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"emailVerified": true})
}

func (api *handler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if !api.allowRequests(w, r, "password-reset-ip", api.requestIP(r), 20, time.Hour) {
		return
	}
	var input struct {
		Email string `json:"email"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if !validEmail(normalizeEmail(input.Email)) {
		writeFieldError(w, r, "email", "invalid_format", "Enter a valid email address.")
		return
	}
	if !api.allowRequests(w, r, "password-reset-email", normalizeEmail(input.Email), 3, time.Hour) {
		return
	}
	if err := api.service.RequestPasswordReset(r.Context(), input.Email); err != nil {
		api.writeInternalError(w, r, "request password reset", err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "If the account exists, password reset instructions will be sent."})
}

func (api *handler) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	if !api.allowRequests(w, r, "password-reset-confirm-ip", api.requestIP(r), 20, time.Hour) {
		return
	}
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if err := api.service.ConfirmPasswordReset(r.Context(), input.Token, input.NewPassword); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *handler) listSessions(w http.ResponseWriter, r *http.Request, principal Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	result, err := api.service.ListSessions(r.Context(), principal, pageNumber, pageSize)
	if err != nil {
		api.writeInternalError(w, r, "list sessions", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) revokeSession(w http.ResponseWriter, r *http.Request, principal Principal) {
	id := chi.URLParam(r, "sessionId")
	if !isUUID(id) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "The session ID is invalid.")
		return
	}
	if err := api.service.RevokeOwnSession(r.Context(), principal, id); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *handler) changePassword(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !api.allowRequests(w, r, "password-change", principal.ID, 5, time.Hour) {
		return
	}
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	if err := api.service.UpdatePassword(r.Context(), principal, input.CurrentPassword, input.NewPassword); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *handler) beginTOTP(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !api.allowRequests(w, r, "totp-setup", principal.ID, 3, time.Hour) {
		return
	}
	result, err := api.service.BeginTOTP(r.Context(), principal)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) confirmTOTP(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !api.allowRequests(w, r, "totp-confirm", principal.ID, 5, 15*time.Minute) {
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	codes, err := api.service.ConfirmTOTP(r.Context(), principal, input.Code)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"recoveryCodes": codes})
}

func (api *handler) rotateRecoveryCodes(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !api.allowRequests(w, r, "recovery-code-rotation", principal.ID, 5, time.Hour) {
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	codes, err := api.service.RotateRecoveryCodes(r.Context(), principal, input.Code)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"recoveryCodes": codes})
}

func (api *handler) me(w http.ResponseWriter, r *http.Request, principal Principal) {
	user, err := api.service.Me(r.Context(), principal)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}

func (api *handler) updateMe(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input ProfileUpdate
	if !decodeBody(w, r, &input) {
		return
	}
	user, err := api.service.UpdateMe(r.Context(), principal, input)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]User{"user": user})
}

func (api *handler) resubmit(w http.ResponseWriter, r *http.Request, principal Principal) {
	user, err := api.service.Resubmit(r.Context(), principal, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]User{"user": user})
}

func (api *handler) deleteMe(w http.ResponseWriter, r *http.Request, principal Principal) {
	if err := api.service.RequestDeletion(r.Context(), principal, requestID(r)); err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "requested"})
}

func (api *handler) getOwnDeletionRequest(w http.ResponseWriter, r *http.Request, principal Principal) {
	result, err := api.service.GetOwnDeletionRequest(r.Context(), principal)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]OwnDeletionRequest{"request": result})
}

func (api *handler) listApplications(w http.ResponseWriter, r *http.Request, principal Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	result, err := api.service.ListApplications(r.Context(), r.URL.Query().Get("role"), r.URL.Query().Get("regionCode"), pageNumber, pageSize)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) listDeletionRequests(w http.ResponseWriter, r *http.Request, _ Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	result, err := api.service.ListDeletionRequests(r.Context(), r.URL.Query().Get("status"), pageNumber, pageSize)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) approveDeletionRequest(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 && !decodeBody(w, r, &input) {
		return
	}
	api.reviewDeletionRequest(w, r, principal, "approved", input.Reason)
}

func (api *handler) rejectDeletionRequest(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	api.reviewDeletionRequest(w, r, principal, "rejected", input.Reason)
}

func (api *handler) reviewDeletionRequest(w http.ResponseWriter, r *http.Request, principal Principal, decision, reason string) {
	result, err := api.service.ReviewDeletionRequest(
		r.Context(), principal, chi.URLParam(r, "requestId"), decision, reason, requestID(r),
	)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]AccountDeletionRequest{"request": result})
}

func (api *handler) getApplication(w http.ResponseWriter, r *http.Request, _ Principal) {
	result, err := api.service.GetApplication(r.Context(), chi.URLParam(r, "userId"))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) approve(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 && !decodeBody(w, r, &input) {
		return
	}
	result, err := api.service.ReviewAccount(r.Context(), principal, chi.URLParam(r, "userId"), "approved", input.Reason, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) reject(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	result, err := api.service.ReviewAccount(r.Context(), principal, chi.URLParam(r, "userId"), "rejected", input.Reason, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) suspend(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decodeBody(w, r, &input) {
		return
	}
	result, err := api.service.ReviewAccount(r.Context(), principal, chi.URLParam(r, "userId"), "suspended", input.Reason, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) reinstate(w http.ResponseWriter, r *http.Request, principal Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 && !decodeBody(w, r, &input) {
		return
	}
	result, err := api.service.ReviewAccount(r.Context(), principal, chi.URLParam(r, "userId"), "reinstated", input.Reason, requestID(r))
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *handler) auditEvents(w http.ResponseWriter, r *http.Request, _ Principal) {
	pageNumber, pageSize, err := parsePage(r.URL.Query())
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	result, err := api.service.ListAuditEvents(r.Context(), r.URL.Query(), pageNumber, pageSize)
	if err != nil {
		api.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func authBody(response AuthResponse) map[string]any {
	return map[string]any{"accessToken": response.AccessToken, "expiresAt": response.ExpiresAt, "user": response.User}
}

func (api *handler) setRefreshCookie(w http.ResponseWriter, value string, expires time.Time) error {
	csrfToken, err := newCSRFToken(api.config.AppSecret)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name: "zf_refresh", Value: value, Path: "/api/v1/auth", HttpOnly: true,
		Secure: api.config.AppEnv == "production", SameSite: http.SameSiteLaxMode,
		Expires: expires.UTC(), MaxAge: int(time.Until(expires).Seconds()),
	}
	if api.config.RefreshCookieDomain != "" {
		cookie.Domain = api.config.RefreshCookieDomain
	}
	http.SetCookie(w, cookie)
	csrfCookie := &http.Cookie{
		Name: "zf_csrf", Value: csrfToken, Path: "/", Secure: api.config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode, Expires: expires.UTC(), MaxAge: int(time.Until(expires).Seconds()),
	}
	if api.config.RefreshCookieDomain != "" {
		csrfCookie.Domain = api.config.RefreshCookieDomain
	}
	http.SetCookie(w, csrfCookie)
	return nil
}

func (api *handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "zf_refresh", Value: "", Path: "/api/v1/auth", HttpOnly: true,
		Secure: api.config.AppEnv == "production", SameSite: http.SameSiteLaxMode,
		MaxAge: -1, Expires: time.Unix(1, 0).UTC(), Domain: api.config.RefreshCookieDomain,
	})
	http.SetCookie(w, &http.Cookie{
		Name: "zf_csrf", Value: "", Path: "/", Secure: api.config.AppEnv == "production",
		SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0).UTC(),
		Domain: api.config.RefreshCookieDomain,
	})
}

func (api *handler) validCSRF(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	allowed := false
	for _, candidate := range api.config.CORSAllowedOrigins {
		if origin == candidate {
			allowed = true
			break
		}
	}
	csrfCookie, err := r.Cookie("zf_csrf")
	provided := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if !allowed || err != nil || provided == "" ||
		subtle.ConstantTimeCompare([]byte(provided), []byte(csrfCookie.Value)) != 1 ||
		!validCSRFToken(api.config.AppSecret, provided) {
		writeAPIError(w, r, http.StatusForbidden, "csrf_validation_failed", "The request origin or CSRF token is invalid.")
		return false
	}
	return true
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "A JSON request body is required.")
		return false
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "The JSON request body is malformed or contains unknown fields.")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "The JSON request body must contain exactly one value.")
		return false
	}
	return true
}

func (api *handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "Email, password, or verification code is incorrect.")
	case errors.Is(err, ErrUnauthorized):
		writeAPIError(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
	case errors.Is(err, ErrAccountSuspended):
		writeAPIError(w, r, http.StatusForbidden, "account_suspended", "This account is suspended.")
	case errors.Is(err, ErrAccountUnapproved):
		writeAPIError(w, r, http.StatusForbidden, "account_not_approved", "This account must be approved before marketplace access.")
	case errors.Is(err, ErrAccountRejected):
		writeAPIError(w, r, http.StatusForbidden, "account_rejected", "This account was rejected.")
	case errors.Is(err, ErrForbidden):
		writeAPIError(w, r, http.StatusForbidden, "forbidden", "This action is not permitted.")
	case errors.Is(err, ErrNotFound):
		writeAPIError(w, r, http.StatusNotFound, "not_found", "The requested resource was not found.")
	case errors.Is(err, ErrConflict):
		writeAPIError(w, r, http.StatusConflict, "conflict", "The request conflicts with the current resource state.")
	case errors.Is(err, ErrRateLimited):
		writeAPIError(w, r, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
	case errors.Is(err, ErrInvalidInput):
		writeAPIError(w, r, http.StatusUnprocessableEntity, "validation_failed", "One or more input values are invalid.")
	case errors.Is(err, ErrInvalidQuery):
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "One or more query parameters are invalid.")
	default:
		api.writeInternalError(w, r, "request failed", err)
	}
}

func (api *handler) writeInternalError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	api.logger.ErrorContext(r.Context(), operation, "request_id", requestID(r), "error", err)
	writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
}

func (api *handler) allowRequests(w http.ResponseWriter, r *http.Request, scope, value string, limit int, window time.Duration) bool {
	if value == "" {
		value = "unknown"
	}
	if err := api.service.CheckRateLimit(r.Context(), scope, value, limit, window); err != nil {
		api.writeServiceError(w, r, err)
		return false
	}
	return true
}

func writeFieldError(w http.ResponseWriter, r *http.Request, field, code, message string) {
	response := apiErrorResponse{}
	response.Error.Code = "validation_failed"
	response.Error.Message = "One or more fields are invalid."
	response.Error.RequestID = requestID(r)
	response.Error.Fields = []apiFieldError{{Field: field, Code: code, Message: message}}
	writeJSON(w, http.StatusUnprocessableEntity, response)
}

func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	response := apiErrorResponse{}
	response.Error.Code = code
	response.Error.Message = message
	response.Error.RequestID = requestID(r)
	writeJSON(w, status, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func requestID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Request-ID"))
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	_, err := url.ParseRequestURI("/" + value)
	if err != nil {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
		} else if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
