"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link } from "@/i18n/navigation";
import { ApiError, ensureApiSuccess } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";

const emailSchema = z.object({ email: z.email().max(254) });
type EmailValues = z.infer<typeof emailSchema>;

export function VerifyEmailForm({ token }: Readonly<{ token: string }>) {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<EmailValues>({
    resolver: zodResolver(emailSchema),
    defaultValues: { email: "" },
  });

  const resend = handleSubmit(async (values) => {
    setFormError(null);
    setNotice(null);
    try {
      ensureApiSuccess(
        await api.POST("/auth/email-verification/request", {
          body: { email: values.email.trim() },
        }),
      );
      setNotice(t("verificationSent"));
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  const confirm = async () => {
    setFormError(null);
    setNotice(null);
    try {
      ensureApiSuccess(
        await api.POST("/auth/email-verification/confirm", { body: { token } }),
      );
      setNotice(t("verificationConfirmed"));
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  };

  return (
    <div className="space-y-5">
      {formError ? <ApiErrorBanner error={formError} /> : null}
      {notice ? (
        <p
          role="status"
          className="rounded-card bg-sage-soft p-4 text-sm leading-6"
        >
          {notice}
        </p>
      ) : null}
      {token ? (
        <button
          type="button"
          onClick={() => void confirm()}
          className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white"
        >
          {t("confirmEmail")}
        </button>
      ) : (
        <form className="space-y-4" onSubmit={resend} noValidate>
          <label
            htmlFor="email"
            className="text-ink mb-2 block text-sm font-semibold"
          >
            {t("email")}
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            className="border-border min-h-12 w-full rounded-lg border bg-white px-4"
            {...register("email")}
          />
          {errors.email ? (
            <p className="text-danger text-sm">{tCommon("invalidEmail")}</p>
          ) : null}
          <button
            type="submit"
            disabled={isSubmitting}
            className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white disabled:opacity-60"
          >
            {isSubmitting ? tCommon("loading") : t("verificationResend")}
          </button>
        </form>
      )}
      <Link
        href="/sign-in"
        className="text-sage inline-flex min-h-11 items-center font-semibold underline underline-offset-4"
      >
        {t("signInAction")}
      </Link>
    </div>
  );
}
