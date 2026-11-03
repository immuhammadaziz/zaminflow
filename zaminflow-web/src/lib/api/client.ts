import createClient from "openapi-fetch";
import type { paths } from "@/lib/api/schema";
import { ApiError, toApiError } from "@/lib/api/errors";
import {
  broadcastSignedOut,
  getAccessToken,
  setAccessToken,
  withCrossTabRefreshLock,
} from "@/lib/auth/token-store";
import type { components } from "@/lib/api/schema";

type AuthResponse = components["schemas"]["AuthResponse"];
let refreshInFlight: Promise<AuthResponse | null> | null = null;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isAuthResponse(value: unknown): value is AuthResponse {
  if (!isRecord(value) || !isRecord(value.user)) return false;
  const user = value.user;
  return (
    typeof value.accessToken === "string" &&
    value.accessToken.length > 0 &&
    typeof value.expiresAt === "string" &&
    typeof user.id === "string" &&
    (user.role === "farmer" ||
      user.role === "processor" ||
      user.role === "admin") &&
    (user.accountStatus === "pending" ||
      user.accountStatus === "approved" ||
      user.accountStatus === "rejected" ||
      user.accountStatus === "suspended") &&
    typeof user.email === "string" &&
    typeof user.emailVerified === "boolean" &&
    typeof user.fullName === "string" &&
    typeof user.createdAt === "string"
  );
}

function readCsrfCookie(): string | undefined {
  if (typeof document === "undefined") return undefined;
  const cookie = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith("zf_csrf="));
  return cookie?.slice("zf_csrf=".length);
}

export function getCsrfToken(): string {
  const token = readCsrfCookie();
  if (!token) throw new ApiError("csrf_validation_failed");
  return token;
}

async function readError(response: Response): Promise<ApiError> {
  let body: unknown;
  try {
    body = await response.json();
  } catch (error) {
    if (error instanceof SyntaxError) return toApiError(undefined, response);
    throw error;
  }
  return toApiError(body, response);
}

async function refreshSession(
  apiBaseUrl: string,
): Promise<AuthResponse | null> {
  if (refreshInFlight) return refreshInFlight;

  const refreshRequest = withCrossTabRefreshLock(async () => {
    const csrf = readCsrfCookie();
    if (!csrf) {
      throw new ApiError("csrf_validation_failed");
    }

    const response = await fetch(`${apiBaseUrl}/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "X-CSRF-Token": csrf },
    });

    if (response.status === 401) {
      broadcastSignedOut();
      return null;
    }

    if (!response.ok) {
      const error = await readError(response);
      if (error.code === "refresh_token_reuse_detected") {
        broadcastSignedOut();
        return null;
      }
      throw error;
    }

    let payload: unknown;
    try {
      payload = await response.json();
    } catch (error) {
      if (!(error instanceof SyntaxError)) throw error;
      throw new ApiError("internal_error", {
        requestId: response.headers.get("X-Request-ID") ?? undefined,
      });
    }
    if (!isAuthResponse(payload)) {
      throw new ApiError("internal_error", {
        requestId: response.headers.get("X-Request-ID") ?? undefined,
      });
    }

    setAccessToken(payload.accessToken);
    return payload;
  });
  refreshInFlight = refreshRequest;
  try {
    return await refreshRequest;
  } finally {
    if (refreshInFlight === refreshRequest) refreshInFlight = null;
  }
}

function csrfProtectedPath(path: string): boolean {
  return path.endsWith("/auth/refresh") || path.endsWith("/auth/sign-out");
}

function publicAuthPath(path: string): boolean {
  return (
    path.endsWith("/auth/sign-in") ||
    path.endsWith("/auth/sign-up") ||
    path.includes("/auth/email-verification/") ||
    path.includes("/auth/password-reset/")
  );
}

function withAuthorization(request: Request, token: string | null): Request {
  const headers = new Headers(request.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  else headers.delete("Authorization");
  return new Request(request, { headers });
}

function createAuthenticatedFetch(apiBaseUrl: string): typeof fetch {
  return async (input, init) => {
    const original = new Request(input, { ...init, credentials: "include" });
    const url = new URL(original.url);
    const path = url.pathname;
    let request = original;
    const currentToken = getAccessToken();

    if (!publicAuthPath(path) && !path.endsWith("/auth/refresh")) {
      request = withAuthorization(original, currentToken);
    }

    if (csrfProtectedPath(path)) {
      const csrf = readCsrfCookie();
      if (!csrf) throw new ApiError("csrf_validation_failed");
      const headers = new Headers(request.headers);
      headers.set("X-CSRF-Token", csrf);
      request = new Request(request, { headers });
    }

    const retryTemplate = request.clone();
    const response = await fetch(request);
    if (
      response.status !== 401 ||
      !currentToken ||
      publicAuthPath(path) ||
      path.endsWith("/auth/refresh") ||
      path.endsWith("/auth/sign-out")
    ) {
      return response;
    }

    const refreshedSession = await refreshSession(apiBaseUrl);
    if (!refreshedSession) return response;
    return fetch(
      withAuthorization(retryTemplate, refreshedSession.accessToken),
    );
  };
}

export function createApiClient(apiBaseUrl: string) {
  return createClient<paths>({
    baseUrl: apiBaseUrl,
    fetch: createAuthenticatedFetch(apiBaseUrl),
  });
}

export type ApiClient = ReturnType<typeof createApiClient>;

export async function restoreAccessToken(
  apiBaseUrl: string,
): Promise<AuthResponse | null> {
  return refreshSession(apiBaseUrl);
}

export function readApiError(value: unknown, response: Response): ApiError {
  return toApiError(value, response);
}
