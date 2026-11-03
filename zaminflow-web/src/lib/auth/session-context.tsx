"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { components } from "@/lib/api/schema";
import {
  createApiClient,
  getCsrfToken,
  restoreAccessToken,
  type ApiClient,
} from "@/lib/api/client";
import { ApiError, ensureApiSuccess, getApiData } from "@/lib/api/errors";
import {
  broadcastSessionAvailable,
  broadcastSignedOut,
  clearAccessToken,
  setAccessToken,
  subscribeAuthExpired,
  subscribeCrossTabSession,
} from "@/lib/auth/token-store";

type User = components["schemas"]["User"];
type SignInRequest = components["schemas"]["SignInRequest"];
type AuthResponse = components["schemas"]["AuthResponse"];
export type SessionStatus =
  "checking" | "anonymous" | "authenticated" | "admin_enrollment" | "error";

type SessionContextValue = Readonly<{
  api: ApiClient;
  status: SessionStatus;
  user: User | null;
  error: ApiError | null;
  signIn: (credentials: SignInRequest) => Promise<User>;
  signOut: () => Promise<void>;
  updateUser: (user: User) => void;
  refreshSession: () => Promise<void>;
}>;

const SessionContext = createContext<SessionContextValue | null>(null);

function normalizeSessionError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError("internal_error");
}

async function loadSession(
  api: ApiClient,
  apiBaseUrl: string,
): Promise<
  Readonly<{
    status: "anonymous" | "authenticated" | "admin_enrollment";
    user: User | null;
  }>
> {
  let authResponse: AuthResponse | null;
  try {
    authResponse = await restoreAccessToken(apiBaseUrl);
  } catch (error) {
    if (error instanceof ApiError && error.code === "csrf_validation_failed") {
      return { status: "anonymous", user: null };
    }
    throw error;
  }
  if (!authResponse) return { status: "anonymous", user: null };

  try {
    const user = getApiData(await api.GET("/me")).user;
    return { status: "authenticated", user };
  } catch (error) {
    if (
      authResponse.user.role === "admin" &&
      error instanceof ApiError &&
      error.code === "admin_mfa_required"
    ) {
      return { status: "admin_enrollment", user: authResponse.user };
    }
    throw error;
  }
}

export function SessionProvider({
  apiBaseUrl,
  children,
}: Readonly<{ apiBaseUrl: string; children: ReactNode }>) {
  const api = useMemo(() => createApiClient(apiBaseUrl), [apiBaseUrl]);
  const [status, setStatus] = useState<SessionStatus>("checking");
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<ApiError | null>(null);

  const refreshSession = useCallback(async () => {
    setStatus("checking");
    setError(null);
    try {
      const session = await loadSession(api, apiBaseUrl);
      setStatus(session.status);
      setUser(session.user);
    } catch (cause) {
      setStatus("error");
      setError(normalizeSessionError(cause));
    }
  }, [api, apiBaseUrl]);

  useEffect(() => {
    let active = true;
    const unsubscribeExpired = subscribeAuthExpired(() => {
      setStatus("anonymous");
      setUser(null);
      setError(null);
    });
    const unsubscribeCrossTab = subscribeCrossTabSession((event) => {
      if (event === "signed_out") {
        setStatus("anonymous");
        setUser(null);
        setError(null);
      } else {
        void refreshSession();
      }
    });
    void loadSession(api, apiBaseUrl)
      .then((session) => {
        if (!active) return;
        setStatus(session.status);
        setUser(session.user);
      })
      .catch((cause: unknown) => {
        if (!active) return;
        setStatus("error");
        setError(normalizeSessionError(cause));
      });
    return () => {
      active = false;
      unsubscribeExpired();
      unsubscribeCrossTab();
    };
  }, [api, apiBaseUrl, refreshSession]);

  const signIn = useCallback(
    async (credentials: SignInRequest) => {
      setStatus("checking");
      setError(null);
      let authResponse: AuthResponse;
      try {
        authResponse = getApiData(
          await api.POST("/auth/sign-in", { body: credentials }),
        );
      } catch (cause) {
        const normalized = normalizeSessionError(cause);
        setError(normalized);
        setStatus(cause instanceof ApiError ? "anonymous" : "error");
        throw normalized;
      }

      setAccessToken(authResponse.accessToken);
      broadcastSessionAvailable();
      try {
        const currentUser = getApiData(await api.GET("/me")).user;
        setUser(currentUser);
        setStatus("authenticated");
        return currentUser;
      } catch (cause) {
        if (
          authResponse.user.role === "admin" &&
          !credentials.totpCode &&
          !credentials.recoveryCode &&
          cause instanceof ApiError &&
          cause.code === "admin_mfa_required"
        ) {
          setUser(authResponse.user);
          setStatus("admin_enrollment");
          return authResponse.user;
        }
        setUser(authResponse.user);
        setStatus("error");
        const normalized = normalizeSessionError(cause);
        setError(normalized);
        throw normalized;
      }
    },
    [api],
  );

  const signOut = useCallback(async () => {
    try {
      ensureApiSuccess(
        await api.POST("/auth/sign-out", {
          params: {
            header: {
              Origin: window.location.origin,
              "X-CSRF-Token": getCsrfToken(),
            },
          },
        }),
      );
    } finally {
      clearAccessToken();
      broadcastSignedOut();
      setStatus("anonymous");
      setUser(null);
      setError(null);
    }
  }, [api]);

  const updateUser = useCallback((updatedUser: User) => {
    setUser(updatedUser);
    setStatus("authenticated");
    setError(null);
  }, []);

  const value = useMemo(
    () => ({
      api,
      status,
      user,
      error,
      signIn,
      signOut,
      updateUser,
      refreshSession,
    }),
    [api, status, user, error, signIn, signOut, updateUser, refreshSession],
  );

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error("useSession must be used within a SessionProvider.");
  }
  return context;
}
