export interface paths {
    "/meta/roles": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List roles selectable at sign-up */
        get: operations["listSignupRoles"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/meta/regions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List Uzbekistan regions */
        get: operations["listRegions"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/sign-up": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Register a pending farmer or processor */
        post: operations["signUp"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/sign-in": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Sign in; admin users must provide TOTP or a recovery code */
        post: operations["signIn"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/email-verification/request": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Request an enumeration-safe verification email */
        post: operations["requestEmailVerification"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/email-verification/confirm": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Verify email with a one-time token */
        post: operations["confirmEmailVerification"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/password-reset/request": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Request password reset without disclosing account existence */
        post: operations["requestPasswordReset"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/password-reset/confirm": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Consume password reset token and revoke existing sessions */
        post: operations["confirmPasswordReset"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/refresh": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Rotate the refresh token and issue an access token
         * @description Browser requests must include an allowlisted Origin and matching zf_csrf cookie and X-CSRF-Token header.
         */
        post: operations["refreshSession"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/sessions": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the current user's sessions */
        get: operations["listSessions"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/sessions/{sessionId}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                sessionId: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /** Revoke one session owned by the caller */
        delete: operations["revokeSession"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/change-password": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Change password and revoke other sessions */
        post: operations["changePassword"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/totp/setup": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Begin mandatory admin TOTP enrollment
         * @description May be called by an enrollment-only session created after the first CLI-provisioned admin sign-in.
         */
        post: operations["setupAdminTotp"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/totp/confirm": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Confirm admin TOTP and issue recovery codes */
        post: operations["confirmAdminTotp"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/totp/recovery-codes/rotate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Rotate one-time admin recovery codes */
        post: operations["rotateAdminRecoveryCodes"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/sign-out": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Revoke current refresh-token family
         * @description Requires an allowlisted Origin and matching zf_csrf cookie and X-CSRF-Token header.
         */
        post: operations["signOut"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/me/resubmit": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Resubmit corrected rejected account for review */
        post: operations["resubmitAccount"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/me": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get the current user and live account status */
        get: operations["getMe"];
        put?: never;
        post?: never;
        /** Submit an account deletion request for administrator review */
        delete: operations["requestAccountDeletion"];
        options?: never;
        head?: never;
        /** Update allowed fields on the current user's profile */
        patch: operations["updateMe"];
        trace?: never;
    };
    "/me/deletion-request": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get the current user's latest account deletion request and review status */
        get: operations["getOwnAccountDeletionRequest"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/deletion-requests": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List account deletion requests for administrator review
         * @description Defaults to pending requests. Approval records a decision for a separate, manually governed retention process; it never erases account data or restricts marketplace activity. Recheck open transactions and applicable legal obligations before any separate manual retention action.
         */
        get: operations["listAccountDeletionRequests"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/deletion-requests/{requestId}/approve": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                requestId: components["parameters"]["DeletionRequestId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Approve a deletion request for separate manual retention review
         * @description Does not erase or change account data/status or restrict marketplace activity. Requests with active orders cannot be approved at review time; recheck open transactions and applicable legal obligations before any separate manual retention action.
         */
        post: operations["approveAccountDeletionRequest"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/deletion-requests/{requestId}/reject": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                requestId: components["parameters"]["DeletionRequestId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Reject an account deletion request with a reason */
        post: operations["rejectAccountDeletionRequest"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/applications": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List verified applicants awaiting review */
        get: operations["listApplications"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/applications/{userId}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        /** Get applicant profile and review history */
        get: operations["getApplication"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/applications/{userId}/approve": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Approve an eligible applicant */
        post: operations["approveApplication"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/applications/{userId}/reject": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Reject an applicant with a reason */
        post: operations["rejectApplication"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/users/{userId}/suspend": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Suspend user, revoke sessions, and audit the decision */
        post: operations["suspendUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/users/{userId}/reinstate": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Reinstate a suspended user */
        post: operations["reinstateUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/admin/audit-events": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Query redacted administrative and security audit events */
        get: operations["listAuditEvents"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/listings": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Search listings as an approved processor, or view own listings as a farmer */
        get: operations["listListings"];
        put?: never;
        /** Publish a milk listing */
        post: operations["createListing"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/listings/{listingId}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                listingId: string;
            };
            cookie?: never;
        };
        /** Get a listing visible to its farmer owner or a processor */
        get: operations["getListing"];
        put?: never;
        post?: never;
        /** Archive an owned listing if no active reservations prevent it */
        delete: operations["archiveListing"];
        options?: never;
        head?: never;
        /** Update an owned listing */
        patch: operations["updateListing"];
        trace?: never;
    };
    "/orders": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List orders in which the caller is a party */
        get: operations["listOrders"];
        put?: never;
        /** Place an order for an active listing */
        post: operations["createOrder"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: string;
            };
            cookie?: never;
        };
        /** Get an order and status history as one of its parties */
        get: operations["getOrder"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/accept": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Accept a placed order as its farmer */
        post: operations["acceptOrder"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/reject": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Reject a placed order as its farmer */
        post: operations["rejectOrder"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/cancel": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Cancel an order before farmer acceptance as its processor */
        post: operations["cancelOrder"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/status": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Record the next actor-authorized fulfillment status */
        post: operations["transitionOrder"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/disputes": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Open a fulfillment dispute before order completion */
        post: operations["createOrderDispute"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/orders/{orderId}/quality-reports": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Record quality measurements for an order */
        post: operations["createQualityReport"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/dashboards/farmer": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get aggregated farmer dashboard data */
        get: operations["getFarmerDashboard"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/dashboards/processor": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get aggregated processor dashboard data */
        get: operations["getProcessorDashboard"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/dashboards/admin": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get aggregated administrator dashboard data */
        get: operations["getAdminDashboard"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/healthz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Liveness probe */
        get: operations["getHealth"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/readyz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Readiness probe including database connectivity */
        get: operations["getReadiness"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/metrics": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Expose Prometheus metrics to internal monitoring only
         * @description This endpoint must not be publicly exposed; production network access is restricted.
         */
        get: operations["getMetrics"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        RoleOption: {
            /** @enum {string} */
            value: "farmer" | "processor";
            label: string;
        };
        SignUpRequest: {
            fullName: string;
            /** Format: email */
            email: string;
            password: string;
            /** @enum {string} */
            role: "farmer" | "processor";
            phone: string;
            organizationName: string;
            regionCode: string;
            stir: string;
        };
        SignInRequest: {
            /** Format: email */
            email: string;
            /** Format: password */
            password: string;
            totpCode?: string;
            recoveryCode?: string;
        };
        ProfileUpdate: {
            fullName?: string;
            phone?: string;
            organizationName?: string;
            regionCode?: string;
            stir?: string;
        };
        User: {
            /** Format: uuid */
            id: string;
            /** @enum {string} */
            role: "farmer" | "processor" | "admin";
            /** @enum {string} */
            accountStatus: "pending" | "approved" | "rejected" | "suspended";
            /** Format: email */
            email: string;
            emailVerified: boolean;
            fullName: string;
            phone?: string;
            organizationName?: string;
            regionCode?: string;
            stir?: string;
            rejectionReason?: string;
            /** Format: date-time */
            createdAt: string;
            /** Format: date-time */
            updatedAt?: string;
        };
        Region: {
            code: string;
            nameUz: string;
            nameRu: string;
            nameEn: string;
        };
        ListingInput: {
            regionCode: string;
            availableLiters: number;
            fatPercent: number;
            pricePerLiterUzs: number;
            /** Format: date-time */
            availableFrom: string;
            /** Format: date-time */
            availableUntil: string;
        };
        Listing: components["schemas"]["ListingInput"] & {
            /** Format: uuid */
            id: string;
            /** Format: uuid */
            farmerId: string;
            /** @enum {string} */
            status: "draft" | "active" | "paused" | "sold_out" | "expired" | "archived";
            /** Format: date-time */
            createdAt: string;
            /** Format: date-time */
            updatedAt?: string;
        };
        OrderInput: {
            /** Format: uuid */
            listingId: string;
            liters: number;
            /** Format: date-time */
            expectedPickupAt: string;
            note?: string;
        };
        /** @enum {string} */
        OrderStatus: "placed" | "accepted" | "rejected" | "cancelled" | "ready_for_pickup" | "picked_up" | "delivered" | "completed" | "disputed";
        Order: {
            /** Format: uuid */
            id: string;
            /** Format: uuid */
            listingId: string;
            /** Format: uuid */
            farmerId: string;
            /** Format: uuid */
            processorId: string;
            liters: number;
            pricePerLiterUzs: number;
            totalUzs: number;
            status: components["schemas"]["OrderStatus"];
            /** Format: date-time */
            expectedPickupAt: string;
            note?: string;
            /** @enum {string} */
            qualityOutcome?: "accepted" | "rejected";
            /** Format: date-time */
            createdAt: string;
            /** Format: date-time */
            updatedAt: string;
            /** Format: date-time */
            completedAt?: string;
            history?: components["schemas"]["OrderHistoryItem"][];
        };
        OrderHistoryItem: {
            status: components["schemas"]["OrderStatus"];
            /** Format: uuid */
            actorId: string;
            note?: string;
            /** Format: date-time */
            createdAt: string;
        };
        Pagination: {
            page: number;
            pageSize: number;
            total: number;
        };
        PaginatedRegions: components["schemas"]["Pagination"] & {
            items: components["schemas"]["Region"][];
        };
        PaginatedUsers: components["schemas"]["Pagination"] & {
            items: components["schemas"]["User"][];
        };
        AccountDeletionRequest: {
            /** Format: uuid */
            id: string;
            user: components["schemas"]["User"];
            /** @enum {string} */
            status: "requested" | "approved" | "completed" | "rejected";
            /** Format: date-time */
            requestedAt: string;
            /** Format: uuid */
            reviewedBy?: string;
            /** Format: date-time */
            reviewedAt?: string;
            reviewReason?: string;
        };
        OwnDeletionRequest: {
            /** Format: uuid */
            id: string;
            /** @enum {string} */
            status: "requested" | "approved" | "completed" | "rejected";
            /** Format: date-time */
            requestedAt: string;
            /** Format: date-time */
            reviewedAt?: string;
            reason?: string;
        };
        DeletionRequestResponse: {
            request: components["schemas"]["AccountDeletionRequest"];
        };
        PaginatedDeletionRequests: components["schemas"]["Pagination"] & {
            items: components["schemas"]["AccountDeletionRequest"][];
        };
        PaginatedListings: components["schemas"]["Pagination"] & {
            items: components["schemas"]["Listing"][];
        };
        PaginatedOrders: components["schemas"]["Pagination"] & {
            items: components["schemas"]["Order"][];
        };
        PaginatedSessions: components["schemas"]["Pagination"] & {
            items: {
                /** Format: uuid */
                id: string;
                /** Format: date-time */
                createdAt: string;
                /** Format: date-time */
                lastUsedAt: string;
            }[];
        };
        PaginatedAuditEvents: components["schemas"]["Pagination"] & {
            items: {
                [key: string]: unknown;
            }[];
        };
        AuthResponse: {
            accessToken: string;
            /** Format: date-time */
            expiresAt: string;
            user: components["schemas"]["User"];
        };
        DecisionRequest: {
            reason?: string;
        };
        RejectRequest: {
            reason: string;
        };
        EmailRequest: {
            /** Format: email */
            email: string;
        };
        TokenRequest: {
            token: string;
        };
        PasswordResetRequest: {
            token: string;
            newPassword: string;
        };
        ChangePasswordRequest: {
            /** Format: password */
            currentPassword: string;
            newPassword: string;
        };
        TotpCodeRequest: {
            code: string;
        };
        ListingUpdate: {
            availableLiters?: number;
            fatPercent?: number;
            pricePerLiterUzs?: number;
            /** Format: date-time */
            availableFrom?: string;
            /** Format: date-time */
            availableUntil?: string;
            /** @enum {string} */
            status?: "active" | "paused";
        };
        OrderStatusRequest: {
            /** @enum {string} */
            status: "ready_for_pickup" | "picked_up" | "delivered" | "completed";
            note?: string;
        };
        QualityReportInput: {
            measurements: {
                [key: string]: number;
            };
            /** @enum {string} */
            outcome: "accepted" | "rejected";
            note?: string;
        };
        DecisionResponse: components["schemas"]["User"];
        AccountReview: {
            /** Format: uuid */
            id: string;
            /** Format: uuid */
            reviewerId: string;
            /** @enum {string} */
            decision: "approved" | "rejected" | "suspended" | "reinstated" | "resubmitted";
            reason?: string;
            /** Format: date-time */
            createdAt: string;
        };
        Dashboard: {
            period: {
                /** Format: date-time */
                from: string;
                /** Format: date-time */
                to: string;
                /** @enum {string} */
                bucket: "day" | "month";
            };
            kpis: {
                [key: string]: unknown;
            };
            charts: {
                [key: string]: unknown;
            };
            tables: {
                [key: string]: unknown;
            };
        };
        FieldError: {
            field: string;
            code: string;
            message: string;
        };
        ErrorResponse: {
            error: {
                code: string;
                message: string;
                requestId: string;
                fieldErrors?: components["schemas"]["FieldError"][];
            };
        };
    };
    responses: {
        /** @description Malformed request */
        BadRequest: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Missing or invalid authentication */
        Unauthorized: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Wrong role, unapproved account, or missing MFA */
        Forbidden: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Resource not found or inaccessible */
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Duplicate or invalid state transition */
        Conflict: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Field validation failed */
        ValidationError: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
        /** @description Rate limit exceeded */
        RateLimited: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ErrorResponse"];
            };
        };
    };
    parameters: {
        Page: number;
        PageSize: number;
        UserId: string;
        DeletionRequestId: string;
        OrderId: string;
        From: string;
        To: string;
        /** @description Defaults to 30d; ignored when both from and to are supplied. */
        DashboardPeriod: "7d" | "30d" | "90d" | "12m";
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    listSignupRoles: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Farmer and processor roles only */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        items: components["schemas"]["RoleOption"][];
                    };
                };
            };
            429: components["responses"]["RateLimited"];
        };
    };
    listRegions: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Region references */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedRegions"];
                };
            };
            400: components["responses"]["BadRequest"];
        };
    };
    signUp: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["SignUpRequest"];
            };
        };
        responses: {
            /** @description Pending account created */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        user: components["schemas"]["User"];
                        message: string;
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    signIn: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["SignInRequest"];
            };
        };
        responses: {
            /** @description Access token and user; refresh and signed CSRF cookies are set */
            200: {
                headers: {
                    /** @description Sets zf_refresh (HttpOnly, Secure in production, SameSite=Lax, scoped to auth path) and zf_csrf (signed double-submit token). */
                    "Set-Cookie"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    requestEmailVerification: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["EmailRequest"];
            };
        };
        responses: {
            /** @description Request accepted whether or not the account exists */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    confirmEmailVerification: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["TokenRequest"];
            };
        };
        responses: {
            /** @description Email verified */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    requestPasswordReset: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["EmailRequest"];
            };
        };
        responses: {
            /** @description Request accepted */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    confirmPasswordReset: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["PasswordResetRequest"];
            };
        };
        responses: {
            /** @description Password changed */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    refreshSession: {
        parameters: {
            query?: never;
            header: {
                Origin: string;
                "X-CSRF-Token": string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Refreshed session; refresh and signed CSRF cookies are set */
            200: {
                headers: {
                    /** @description Sets zf_refresh and zf_csrf cookies. */
                    "Set-Cookie"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["Conflict"];
            429: components["responses"]["RateLimited"];
        };
    };
    listSessions: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Paginated sessions owned by the caller */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedSessions"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    revokeSession: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                sessionId: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Session revoked */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    changePassword: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ChangePasswordRequest"];
            };
        };
        responses: {
            /** @description Password changed */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    setupAdminTotp: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Secret shown once for authenticator enrollment */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        secret: string;
                        otpauthUri: string;
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["Conflict"];
        };
    };
    confirmAdminTotp: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["TotpCodeRequest"];
            };
        };
        responses: {
            /** @description Recovery codes shown once */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        recoveryCodes: string[];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    rotateAdminRecoveryCodes: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["TotpCodeRequest"];
            };
        };
        responses: {
            /** @description New recovery codes shown once */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            422: components["responses"]["ValidationError"];
            429: components["responses"]["RateLimited"];
        };
    };
    signOut: {
        parameters: {
            query?: never;
            header: {
                Origin: string;
                "X-CSRF-Token": string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Session revoked */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    resubmitAccount: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": Record<string, never>;
            };
        };
        responses: {
            /** @description Account returned to pending and review attempt recorded */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["Conflict"];
        };
    };
    getMe: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Current user */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        user: components["schemas"]["User"];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    requestAccountDeletion: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Request recorded for administrator review; no account data is erased by this endpoint */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        /** @enum {string} */
                        status: "requested";
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            409: components["responses"]["Conflict"];
        };
    };
    updateMe: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ProfileUpdate"];
            };
        };
        responses: {
            /** @description Updated profile */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        user: components["schemas"]["User"];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            422: components["responses"]["ValidationError"];
        };
    };
    getOwnAccountDeletionRequest: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The latest request and its review status */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        request: components["schemas"]["OwnDeletionRequest"];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    listAccountDeletionRequests: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
                /** @description Defaults to requested; use all to include every status. */
                status?: "requested" | "approved" | "completed" | "rejected" | "all";
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Paginated deletion requests and associated account profiles */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedDeletionRequests"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    approveAccountDeletionRequest: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                requestId: components["parameters"]["DeletionRequestId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Request decision recorded and audited; account data remains unchanged */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DeletionRequestResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    rejectAccountDeletionRequest: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                requestId: components["parameters"]["DeletionRequestId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RejectRequest"];
            };
        };
        responses: {
            /** @description Request decision recorded and audited; account data remains unchanged */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DeletionRequestResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    listApplications: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
                role?: "farmer" | "processor";
                regionCode?: string;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Review queue */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedUsers"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getApplication: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Applicant and review attempts */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        user: components["schemas"]["User"];
                        reviews: components["schemas"]["AccountReview"][];
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    approveApplication: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Account approved and decision audited */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DecisionResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    rejectApplication: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RejectRequest"];
            };
        };
        responses: {
            /** @description Account rejected and decision audited */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DecisionResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    suspendUser: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RejectRequest"];
            };
        };
        responses: {
            /** @description Account suspended and sessions revoked */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DecisionResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    reinstateUser: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                userId: components["parameters"]["UserId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Account reinstated and decision audited */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DecisionResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    listAuditEvents: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
                actorId?: string;
                action?: string;
                targetId?: string;
                targetType?: string;
                from?: components["parameters"]["From"];
                to?: components["parameters"]["To"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Paginated audit events */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedAuditEvents"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    listListings: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
                regionCode?: string;
                minLiters?: number;
                sort?: "newest" | "oldest" | "price_asc" | "price_desc";
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Listings visible to caller */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedListings"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    createListing: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ListingInput"];
            };
        };
        responses: {
            /** @description Listing created */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        listing: components["schemas"]["Listing"];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            422: components["responses"]["ValidationError"];
        };
    };
    getListing: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                listingId: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Listing */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Listing"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    archiveListing: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                listingId: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Listing archived */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    updateListing: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                listingId: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ListingUpdate"];
            };
        };
        responses: {
            /** @description Updated listing */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Listing"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    listOrders: {
        parameters: {
            query?: {
                page?: components["parameters"]["Page"];
                pageSize?: components["parameters"]["PageSize"];
                status?: components["schemas"]["OrderStatus"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Caller-owned party orders */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["PaginatedOrders"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    createOrder: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["OrderInput"];
            };
        };
        responses: {
            /** @description Order placed with price snapshot */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        order: components["schemas"]["Order"];
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    getOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Order and immutable status history */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Order"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    acceptOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Order accepted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Order"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    rejectOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Order rejected */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Order"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    cancelOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody?: {
            content: {
                "application/json": components["schemas"]["DecisionRequest"];
            };
        };
        responses: {
            /** @description Order cancelled and reserved quantity released */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Order"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    transitionOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["OrderStatusRequest"];
            };
        };
        responses: {
            /** @description Order transitioned and history appended */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Order"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    createOrderDispute: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["RejectRequest"];
            };
        };
        responses: {
            /** @description Dispute recorded */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        /** @enum {string} */
                        status: "disputed";
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    createQualityReport: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                orderId: components["parameters"]["OrderId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["QualityReportInput"];
            };
        };
        responses: {
            /** @description Quality report recorded */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        /** @enum {string} */
                        status: "recorded";
                    };
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
            422: components["responses"]["ValidationError"];
        };
    };
    getFarmerDashboard: {
        parameters: {
            query?: {
                from?: components["parameters"]["From"];
                to?: components["parameters"]["To"];
                /** @description Defaults to 30d; ignored when both from and to are supplied. */
                period?: components["parameters"]["DashboardPeriod"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Database-computed farmer KPIs, charts, and tables */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Dashboard"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getProcessorDashboard: {
        parameters: {
            query?: {
                from?: components["parameters"]["From"];
                to?: components["parameters"]["To"];
                /** @description Defaults to 30d; ignored when both from and to are supplied. */
                period?: components["parameters"]["DashboardPeriod"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Database-computed processor KPIs, charts, and tables */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Dashboard"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getAdminDashboard: {
        parameters: {
            query?: {
                from?: components["parameters"]["From"];
                to?: components["parameters"]["To"];
                /** @description Defaults to 30d; ignored when both from and to are supplied. */
                period?: components["parameters"]["DashboardPeriod"];
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Database-computed admin KPIs, charts, and tables */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Dashboard"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    getHealth: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Process is live */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        /** @enum {string} */
                        status?: "ok";
                    };
                };
            };
        };
    };
    getReadiness: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Dependencies ready */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Dependency unavailable */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getMetrics: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Prometheus exposition format */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
}
