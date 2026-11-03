export type PublicRuntimeConfig = Readonly<{
  apiBaseUrl: string;
}>;

export function getPublicRuntimeConfig(): PublicRuntimeConfig {
  const configuredUrl = process.env.API_BASE_URL;
  if (!configuredUrl) {
    throw new Error("API_BASE_URL must be set at application start.");
  }

  let url: URL;
  try {
    url = new URL(configuredUrl);
  } catch {
    throw new Error("API_BASE_URL must be an absolute URL.");
  }

  if (
    (url.protocol !== "https:" && url.protocol !== "http:") ||
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    !url.pathname.replace(/\/+$/, "").endsWith("/api/v1")
  ) {
    throw new Error("API_BASE_URL must be an HTTP(S) URL ending in /api/v1.");
  }

  if (
    process.env.NODE_ENV === "production" &&
    url.protocol !== "https:" &&
    url.hostname !== "localhost"
  ) {
    throw new Error("Production API_BASE_URL must use HTTPS.");
  }

  return { apiBaseUrl: url.toString().replace(/\/+$/, "") };
}
