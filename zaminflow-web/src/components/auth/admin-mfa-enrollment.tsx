"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { QRCodeSVG } from "qrcode.react";
import { useTranslations } from "next-intl";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link, useRouter } from "@/i18n/navigation";
import { ApiError, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import { broadcastSignedOut, clearAccessToken } from "@/lib/auth/token-store";

const codeSchema = z.object({ code: z.string().regex(/^[0-9]{6}$/) });
type CodeValues = z.infer<typeof codeSchema>;

export function AdminMfaEnrollment() {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const {
    api,
    status,
    user,
    error: sessionError,
    refreshSession,
  } = useSession();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<readonly string[] | null>(
    null,
  );
  const [copyFailed, setCopyFailed] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<CodeValues>({
    resolver: zodResolver(codeSchema),
    defaultValues: { code: "" },
  });

  const setupQuery = useQuery({
    queryKey: ["admin-totp-setup"],
    enabled: status === "admin_enrollment" && user?.role === "admin",
    queryFn: async () => getApiData(await api.POST("/auth/totp/setup")),
    retry: false,
  });

  useEffect(() => {
    if (status === "authenticated" && user?.role === "admin") {
      router.replace("/admin/dashboard");
    } else if (status === "authenticated" && user?.role !== "admin") {
      router.replace("/");
    }
  }, [router, status, user?.role]);

  const submit = handleSubmit(async ({ code }) => {
    setFormError(null);
    try {
      const result = getApiData(
        await api.POST("/auth/totp/confirm", { body: { code } }),
      );
      setRecoveryCodes(result.recoveryCodes);
      clearAccessToken();
      broadcastSignedOut();
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  const downloadCodes = () => {
    if (!recoveryCodes) return;
    const blob = new Blob([`${recoveryCodes.join("\n")}\n`], {
      type: "text/plain;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "zaminflow-recovery-codes.txt";
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const copyCodes = async () => {
    if (!recoveryCodes) return;
    try {
      await navigator.clipboard.writeText(recoveryCodes.join("\n"));
      setCopyFailed(false);
    } catch {
      setCopyFailed(true);
    }
  };

  if (status === "checking") {
    return (
      <p role="status" className="text-muted text-sm">
        {t("checkingSession")}
      </p>
    );
  }
  if (status === "error") {
    return (
      <div className="space-y-4">
        <ApiErrorBanner
          error={sessionError ?? new ApiError("internal_error")}
        />
        <button
          type="button"
          onClick={() => void refreshSession()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (status === "anonymous" || user?.role !== "admin") {
    return (
      <div className="space-y-4">
        <p className="text-muted text-sm">{t("signInDescription")}</p>
        <Link
          href="/sign-in"
          className="bg-sage inline-flex min-h-12 w-full items-center justify-center rounded-full px-6 font-semibold text-white"
        >
          {t("signInAction")}
        </Link>
      </div>
    );
  }

  if (recoveryCodes) {
    return (
      <div className="space-y-5">
        <h2 className="text-ink text-xl font-bold">
          {t("recoveryCodesTitle")}
        </h2>
        <p className="text-muted text-sm leading-6">
          {t("recoveryCodesDescription")}
        </p>
        <ol className="rounded-card border-border bg-surface-soft grid gap-2 border p-4 font-mono text-sm sm:grid-cols-2">
          {recoveryCodes.map((code) => (
            <li key={code}>{code}</li>
          ))}
        </ol>
        <div className="flex flex-col gap-3 sm:flex-row">
          <button
            type="button"
            onClick={downloadCodes}
            className="border-border min-h-11 flex-1 rounded-full border px-4 font-semibold"
          >
            {t("downloadCodes")}
          </button>
          <button
            type="button"
            onClick={() => void copyCodes()}
            className="border-border min-h-11 flex-1 rounded-full border px-4 font-semibold"
          >
            {t("copyCodes")}
          </button>
        </div>
        {copyFailed ? (
          <p role="alert" className="text-danger text-sm">
            {tCommon("serverError")}
          </p>
        ) : null}
        <Link
          href="/sign-in"
          className="bg-sage inline-flex min-h-12 w-full items-center justify-center rounded-full px-6 font-semibold text-white"
        >
          {t("codesSaved")}
        </Link>
      </div>
    );
  }

  if (status !== "admin_enrollment") {
    return (
      <p role="status" className="text-muted text-sm">
        {t("checkingSession")}
      </p>
    );
  }

  if (setupQuery.isError) {
    const setupError =
      setupQuery.error instanceof ApiError
        ? setupQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={setupError} />
        <button
          type="button"
          onClick={() => void setupQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!setupQuery.data) {
    return (
      <p role="status" className="text-muted text-sm">
        {tCommon("loading")}
      </p>
    );
  }

  return (
    <div className="space-y-5">
      {formError ? <ApiErrorBanner error={formError} /> : null}
      <div className="rounded-card flex justify-center bg-white p-5">
        <QRCodeSVG value={setupQuery.data.otpauthUri} size={208} level="M" />
      </div>
      <div>
        <p className="text-ink text-sm font-semibold">{t("manualSetupKey")}</p>
        <code className="bg-surface-soft mt-2 block rounded-lg p-3 text-sm break-all">
          {setupQuery.data.secret}
        </code>
      </div>
      <form className="space-y-4" onSubmit={submit} noValidate>
        <label
          htmlFor="code"
          className="text-ink mb-2 block text-sm font-semibold"
        >
          {t("totpCode")}
        </label>
        <input
          id="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          aria-invalid={Boolean(errors.code)}
          className="border-border min-h-12 w-full rounded-lg border bg-white px-4"
          {...register("code")}
        />
        {errors.code ? (
          <p className="text-danger text-sm">{t("validationError")}</p>
        ) : null}
        <button
          type="submit"
          disabled={isSubmitting}
          className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white disabled:opacity-60"
        >
          {isSubmitting ? tCommon("loading") : t("confirmEnrollment")}
        </button>
      </form>
    </div>
  );
}
