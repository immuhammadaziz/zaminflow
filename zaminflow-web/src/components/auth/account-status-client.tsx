"use client";

import { useTranslations } from "next-intl";
import { useEffect } from "react";
import { useState } from "react";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link, usePathname, useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api/errors";
import { getUserDestination } from "@/lib/auth/routes";
import { useSession } from "@/lib/auth/session-context";

type Status = "pending" | "suspended";

export function AccountStatusClient({ status }: Readonly<{ status: Status }>) {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const pathname = usePathname();
  const router = useRouter();
  const { user, error, refreshSession, signOut } = useSession();
  const [signOutError, setSignOutError] = useState<ApiError | null>(null);

  useEffect(() => {
    const refresh = () => void refreshSession();
    const interval = window.setInterval(refresh, 60_000);
    window.addEventListener("focus", refresh);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("focus", refresh);
    };
  }, [refreshSession]);

  useEffect(() => {
    if (user && getUserDestination(user) !== pathname) {
      router.replace(getUserDestination(user));
    }
  }, [pathname, router, user]);

  if (!user || user.accountStatus !== status) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {t("checkingSession")}
      </p>
    );
  }

  return (
    <div className="space-y-5">
      {error ? <ApiErrorBanner error={error} /> : null}
      <p className="rounded-card bg-sage-soft text-ink p-4 text-sm leading-6">
        {user.emailVerified
          ? status === "pending"
            ? t("awaitingApprovalDescription")
            : t("suspendedDescription")
          : t("unverifiedAccountDescription")}
      </p>
      {signOutError ? <ApiErrorBanner error={signOutError} /> : null}
      {!user.emailVerified ? (
        <Link
          href="/verify-email"
          className="bg-sage inline-flex min-h-11 items-center rounded-full px-5 font-semibold text-white"
        >
          {t("verificationResend")}
        </Link>
      ) : null}
      <div className="flex flex-wrap gap-3">
        <button
          type="button"
          onClick={() => void refreshSession()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
        <button
          type="button"
          onClick={() => {
            setSignOutError(null);
            void signOut().catch((cause: unknown) => {
              setSignOutError(
                cause instanceof ApiError
                  ? cause
                  : new ApiError("internal_error"),
              );
            });
          }}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {t("signOut")}
        </button>
      </div>
    </div>
  );
}
