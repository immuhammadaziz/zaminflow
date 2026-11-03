import { describe, expect, it } from "vitest";
import {
  ApiError,
  ensureApiSuccess,
  getApiData,
  toApiError,
} from "@/lib/api/errors";

describe("API error handling", () => {
  it("preserves backend code, request ID, field errors and retry delay", () => {
    const error = toApiError(
      {
        error: {
          code: "validation_failed",
          requestId: "request-123",
          fieldErrors: [
            {
              field: "email",
              code: "invalid_format",
              message: "Invalid email",
            },
            {
              field: 12,
              code: "invalid",
              message: "Discard this malformed entry",
            },
          ],
        },
      },
      new Response(null, {
        status: 429,
        headers: { "Retry-After": "30" },
      }),
    );

    expect(error.code).toBe("validation_failed");
    expect(error.requestId).toBe("request-123");
    expect(error.fieldErrors).toEqual([
      { field: "email", code: "invalid_format", message: "Invalid email" },
    ]);
    expect(error.retryAfterSeconds).toBe(30);
  });

  it("uses the response request ID when the error payload omits one", () => {
    const error = toApiError(
      { error: { code: "forbidden" } },
      new Response(null, {
        status: 403,
        headers: { "X-Request-ID": "response-request-id" },
      }),
    );

    expect(error.requestId).toBe("response-request-id");
  });

  it("throws the parsed API error when a result contains an error", () => {
    expect(() =>
      getApiData({
        error: { error: { code: "conflict" } },
        response: new Response(null, { status: 409 }),
      }),
    ).toThrowError(new ApiError("conflict"));
  });

  it("treats a successful response without data as an invalid response", () => {
    expect(() =>
      getApiData({
        response: new Response(null, { status: 200 }),
      }),
    ).toThrowError(new ApiError("internal_error"));
  });

  it("accepts a successful empty response", () => {
    expect(() =>
      ensureApiSuccess({ response: new Response(null, { status: 204 }) }),
    ).not.toThrow();
  });

  it("rejects unsuccessful empty responses", () => {
    expect(() =>
      ensureApiSuccess({
        response: new Response(null, { status: 403 }),
      }),
    ).toThrowError(new ApiError("internal_error"));
  });
});
