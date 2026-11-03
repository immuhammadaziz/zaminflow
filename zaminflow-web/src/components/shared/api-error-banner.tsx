"use client";

import { useTranslations } from "next-intl";
import backendErrorCodes from "../../../contracts/error-codes.json";
import { ApiError } from "@/lib/api/errors";

export function ApiErrorBanner({
  error,
  className = "",
}: Readonly<{ error: ApiError; className?: string }>) {
  const t = useTranslations();
  const tErrors = useTranslations("errors");
  const code = (backendErrorCodes as readonly string[]).includes(error.code)
    ? error.code
    : "internal_error";

  return (
    <div
      role="alert"
      className={`rounded-card border-danger/30 bg-danger/5 text-danger border p-4 text-sm ${className}`}
    >
      <p>{tErrors(code)}</p>
      {error.requestId ? (
        <p className="mt-2 text-xs break-all">
          {t("common.requestId")}: {error.requestId}
        </p>
      ) : null}
    </div>
  );
}
