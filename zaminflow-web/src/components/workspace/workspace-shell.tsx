"use client";

import { useTranslations } from "next-intl";
import { useState, type ReactNode } from "react";
import { Link } from "@/i18n/navigation";
import { ApiError } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import type { components } from "@/lib/api/schema";

type Role = components["schemas"]["User"]["role"];
type NavigationItem = Readonly<{ href: string; label: string }>;

export function WorkspaceShell({
  role,
  title,
  children,
}: Readonly<{ role: Role; title: string; children: ReactNode }>) {
  const t = useTranslations("workspace");
  const tCommon = useTranslations("common");
  const { user, signOut } = useSession();
  const [signOutError, setSignOutError] = useState<ApiError | null>(null);

  const items: NavigationItem[] = [
    { href: `/${role}/dashboard`, label: "dashboard" },
    ...(role === "farmer"
      ? [
          { href: "/farmer/listings", label: "myListings" },
          { href: "/farmer/orders", label: "incomingOrders" },
        ]
      : []),
    ...(role === "processor"
      ? [
          { href: "/marketplace", label: "market" },
          { href: "/processor/orders", label: "incomingOrders" },
        ]
      : []),
    ...(role === "admin"
      ? [
          { href: "/admin/applications", label: "applications" },
          { href: "/admin/audit-events", label: "audit" },
          { href: "/admin/deletion-requests", label: "deletionRequests" },
        ]
      : []),
    { href: "/account/profile", label: "profile" },
    { href: "/account/security", label: "security" },
    { href: "/account/deletion", label: "accountDeletion" },
  ];

  return (
    <main id="main-content" className="min-h-screen">
      <header className="border-border bg-surface border-b">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-5 py-4 sm:px-8">
          <Link href="/" className="font-display text-sage text-xl font-bold">
            ZaminFlow
          </Link>
          <div className="flex items-center gap-3">
            <span className="text-muted hidden max-w-48 truncate text-sm sm:inline">
              {user?.fullName}
            </span>
            <button
              type="button"
              onClick={() => {
                setSignOutError(null);
                void signOut().catch((error: unknown) => {
                  setSignOutError(
                    error instanceof ApiError
                      ? error
                      : new ApiError("internal_error"),
                  );
                });
              }}
              className="border-border text-ink min-h-10 rounded-full border px-4 text-sm font-semibold"
            >
              {t("signOut")}
            </button>
          </div>
          <nav
            aria-label="Workspace"
            className="order-3 flex w-full gap-2 overflow-x-auto pb-1 text-sm"
          >
            {items.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className="text-muted hover:bg-sage-soft hover:text-sage shrink-0 rounded-full px-3 py-2 font-semibold"
              >
                {item.label.startsWith("dashboard.") ? (
                  <DashboardLabel role={role} />
                ) : (
                  t(item.label.replace("workspace.", ""))
                )}
              </Link>
            ))}
          </nav>
        </div>
      </header>
      <div className="mx-auto max-w-7xl px-5 py-8 sm:px-8 sm:py-10">
        <div className="mb-7">
          <p className="text-sage text-sm font-semibold">ZaminFlow</p>
          <h1 className="font-display text-ink mt-2 text-3xl font-bold tracking-tight">
            {title}
          </h1>
        </div>
        {signOutError ? (
          <div className="mb-6">
            <ApiErrorBanner error={signOutError} />
            <p className="text-muted mt-2 text-sm">{tCommon("serverError")}</p>
          </div>
        ) : null}
        {children}
      </div>
    </main>
  );
}

function DashboardLabel({ role }: Readonly<{ role: Role }>) {
  const t = useTranslations("dashboard");
  const key = role === "admin" ? "adminTitle" : `${role}Title`;
  return <>{t(key)}</>;
}
