"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { SessionProvider } from "@/lib/auth/session-context";

export function AppProviders({
  apiBaseUrl,
  children,
}: Readonly<{ apiBaseUrl: string; children: ReactNode }>) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: (failureCount, error) =>
              failureCount < 1 &&
              (!(error instanceof Error) ||
                !("code" in error && typeof error.code === "string")),
            refetchOnWindowFocus: false,
            staleTime: 30_000,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider apiBaseUrl={apiBaseUrl}>{children}</SessionProvider>
    </QueryClientProvider>
  );
}
