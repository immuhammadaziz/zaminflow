import type { components } from "@/lib/api/schema";

export type ApiFieldError = components["schemas"]["FieldError"];

export class ApiError extends Error {
  readonly code: string;
  readonly requestId: string | undefined;
  readonly fieldErrors: readonly ApiFieldError[];
  readonly retryAfterSeconds: number | undefined;

  constructor(
    code: string,
    options: Readonly<{
      requestId?: string | undefined;
      fieldErrors?: readonly ApiFieldError[];
      retryAfterSeconds?: number | undefined;
    }> = {},
  ) {
    super(code);
    this.name = "ApiError";
    this.code = code;
    this.requestId = options.requestId;
    this.fieldErrors = options.fieldErrors ?? [];
    this.retryAfterSeconds = options.retryAfterSeconds;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isFieldError(value: unknown): value is ApiFieldError {
  return (
    isRecord(value) &&
    typeof value.field === "string" &&
    typeof value.code === "string" &&
    typeof value.message === "string"
  );
}

function parseRetryAfter(response: Response): number | undefined {
  const value = response.headers.get("Retry-After");
  if (!value) return undefined;
  const seconds = Number(value);
  if (Number.isFinite(seconds) && seconds >= 0) return Math.ceil(seconds);

  const date = Date.parse(value);
  if (!Number.isFinite(date)) return undefined;
  return Math.max(0, Math.ceil((date - Date.now()) / 1000));
}

export function toApiError(value: unknown, response: Response): ApiError {
  const top = isRecord(value) ? value.error : undefined;
  const error = isRecord(top) ? top : undefined;
  const fieldErrors = Array.isArray(error?.fieldErrors)
    ? error.fieldErrors.filter(isFieldError)
    : [];
  const requestId =
    (typeof error?.requestId === "string" && error.requestId) ||
    response.headers.get("X-Request-ID") ||
    undefined;
  const code = typeof error?.code === "string" ? error.code : "internal_error";

  return new ApiError(code, {
    requestId,
    fieldErrors,
    retryAfterSeconds: parseRetryAfter(response),
  });
}

export function getApiData<T>(result: {
  data?: T;
  error?: unknown;
  response: Response;
}): T {
  if (result.error !== undefined) {
    throw toApiError(result.error, result.response);
  }
  if (result.data === undefined) {
    throw toApiError(undefined, result.response);
  }
  return result.data;
}

export function ensureApiSuccess(result: {
  data?: unknown;
  error?: unknown;
  response: Response;
}): void {
  if (result.error !== undefined || !result.response.ok) {
    throw toApiError(result.error, result.response);
  }
}
