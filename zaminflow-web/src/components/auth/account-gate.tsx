"use client";

import { useEffect, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { usePathname, useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

export function AccountGate({ children }: Readonly<{ children: ReactNode }>) {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const pathname = usePathname();
  const router = useRouter();
  const { status, user, error, refreshSession } = useSession();

  useEffect(() => {
    if (status === "anonymous") router.replace("/sign-in");
    if (status === "admin_enrollment" && pathname !== "/admin/security/mfa") {
      router.replace("/admin/security/mfa");
    }
  }, [pathname, router, status]);

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

  if (status !== "authenticated" || !user) {
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

  return children;
}

export function AccountWorkspace({
  title,
  children,
}: Readonly<{ title: string; children: ReactNode }>) {
  const { user } = useSession();
  if (!user) return null;
  return (
    <WorkspaceShell role={user.role} title={title}>
      {children}
    </WorkspaceShell>
  );
}
