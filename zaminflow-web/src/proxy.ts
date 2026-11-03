import { randomBytes } from "node:crypto";
import { NextResponse, type NextRequest } from "next/server";
import { routing } from "@/i18n/routing";
import { getPublicRuntimeConfig } from "@/lib/config/runtime";

function createContentSecurityPolicy(nonce: string, apiOrigin: string) {
  const connectSources = [
    "'self'",
    apiOrigin,
    ...(process.env.NODE_ENV === "development"
      ? ["ws://127.0.0.1:3000", "ws://localhost:3000"]
      : []),
  ];

  return [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
    `style-src 'self' 'nonce-${nonce}'`,
    `connect-src ${connectSources.join(" ")}`,
    "img-src 'self' data: blob:",
    "font-src 'self'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "frame-src 'none'",
    "worker-src 'self' blob:",
    "manifest-src 'self'",
    ...(process.env.NODE_ENV === "production"
      ? ["upgrade-insecure-requests"]
      : []),
  ].join("; ");
}

function applySecurityHeaders(response: NextResponse, policy: string) {
  response.headers.set("Content-Security-Policy", policy);
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  response.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(), geolocation=()",
  );
  response.headers.set("Cross-Origin-Opener-Policy", "same-origin");
  if (process.env.NODE_ENV === "production") {
    response.headers.set(
      "Strict-Transport-Security",
      "max-age=63072000; includeSubDomains; preload",
    );
  }
}

export function proxy(request: NextRequest) {
  const config = getPublicRuntimeConfig();
  const apiOrigin = new URL(config.apiBaseUrl).origin;
  const nonce = randomBytes(16).toString("base64");
  const policy = createContentSecurityPolicy(nonce, apiOrigin);
  const pathLocale = request.nextUrl.pathname.split("/")[1];
  const locale = routing.locales.find((candidate) => candidate === pathLocale);

  if (!locale) {
    const target = request.nextUrl.clone();
    target.pathname = `/${routing.defaultLocale}${target.pathname === "/" ? "" : target.pathname}`;
    const response = NextResponse.redirect(target);
    applySecurityHeaders(response, policy);
    return response;
  }

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("x-locale", locale);
  requestHeaders.set("Content-Security-Policy", policy);

  const response = NextResponse.next({
    request: {
      headers: requestHeaders,
    },
  });
  applySecurityHeaders(response, policy);
  return response;
}

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
