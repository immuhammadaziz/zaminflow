"use client";

import { useEffect, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { usePathname, useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api/errors";
import { getUserDestination } from "@/lib/auth/routes";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type Role = components["schemas"]["User"]["role"];

export function RoleGate({
  role,
  children,
}: Readonly<{ role: Role; children: ReactNode }>) {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const pathname = usePathname();
  const { status, user, error, refreshSession } = useSession();

  useEffect(() => {
    if (status === "anonymous" && pathname !== "/sign-in") {
      router.replace("/sign-in");
    }
    if (status === "admin_enrollment" && pathname !== "/admin/security/mfa") {
      router.replace("/admin/security/mfa");
    }
    if (status === "authenticated" && user) {
      const destination = getUserDestination(user);
      if (destination !== pathname) router.replace(destination);
    }
  }, [pathname, router, status, user]);

  if (
    status === "authenticated" &&
    user?.role === role &&
    user.accountStatus === "approved" &&
    user.emailVerified
  ) {
    return children;
  }

  if (status === "error") {
    return (
      <main className="mx-auto max-w-xl px-5 py-16">
        <ApiErrorBanner error={error ?? new ApiError("internal_error")} />
        <button
          type="button"
          onClick={() => void refreshSession()}
          className="border-border mt-4 min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-xl px-5 py-16">
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-5 text-sm"
      >
        {t("checkingSession")}
      </p>
    </main>
  );
}
