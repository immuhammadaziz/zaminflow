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

const forgotSchema = z.object({ email: z.email().max(254) });
type ForgotValues = z.infer<typeof forgotSchema>;

const resetSchema = z
  .object({
    password: z.string().min(12).max(128),
    confirmPassword: z.string().min(1),
  })
  .refine((value) => value.password === value.confirmPassword, {
    path: ["confirmPassword"],
  });
type ResetValues = z.infer<typeof resetSchema>;

export function ForgotPasswordForm() {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [submitted, setSubmitted] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotValues>({
    resolver: zodResolver(forgotSchema),
    defaultValues: { email: "" },
  });

  const submit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      ensureApiSuccess(
        await api.POST("/auth/password-reset/request", {
          body: { email: values.email.trim() },
        }),
      );
      setSubmitted(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  return (
    <div className="space-y-5">
      {formError ? <ApiErrorBanner error={formError} /> : null}
      {submitted ? (
        <p
          role="status"
          className="rounded-card bg-sage-soft p-4 text-sm leading-6"
        >
          {t("passwordResetRequested")}
        </p>
      ) : (
        <form className="space-y-4" onSubmit={submit} noValidate>
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
            {isSubmitting ? tCommon("loading") : tCommon("continue")}
          </button>
        </form>
      )}
      <Link
        href="/sign-in"
        className="text-sage font-semibold underline underline-offset-4"
      >
        {t("signInAction")}
      </Link>
    </div>
  );
}

export function ResetPasswordForm({ token }: Readonly<{ token: string }>) {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [submitted, setSubmitted] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ResetValues>({
    resolver: zodResolver(resetSchema),
    defaultValues: { password: "", confirmPassword: "" },
  });

  const submit = handleSubmit(async ({ password }) => {
    setFormError(null);
    try {
      ensureApiSuccess(
        await api.POST("/auth/password-reset/confirm", {
          body: { token, newPassword: password },
        }),
      );
      setSubmitted(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  return (
    <div className="space-y-5">
      {formError ? <ApiErrorBanner error={formError} /> : null}
      {submitted ? (
        <div className="space-y-4">
          <p
            role="status"
            className="rounded-card bg-sage-soft p-4 text-sm leading-6"
          >
            {t("passwordResetAccepted")}
          </p>
          <Link
            href="/sign-in"
            className="bg-sage inline-flex min-h-12 w-full items-center justify-center rounded-full px-6 font-semibold text-white"
          >
            {t("signInAction")}
          </Link>
        </div>
      ) : (
        <form className="space-y-4" onSubmit={submit} noValidate>
          <label
            htmlFor="password"
            className="text-ink mb-2 block text-sm font-semibold"
          >
            {t("password")}
          </label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            className="border-border min-h-12 w-full rounded-lg border bg-white px-4"
            {...register("password")}
          />
          <p className="text-muted text-xs">{t("passwordGuidance")}</p>
          {errors.password ? (
            <p className="text-danger text-sm">{t("passwordGuidance")}</p>
          ) : null}
          <label
            htmlFor="confirmPassword"
            className="text-ink mb-2 block text-sm font-semibold"
          >
            {t("confirmPassword")}
          </label>
          <input
            id="confirmPassword"
            type="password"
            autoComplete="new-password"
            className="border-border min-h-12 w-full rounded-lg border bg-white px-4"
            {...register("confirmPassword")}
          />
          {errors.confirmPassword ? (
            <p className="text-danger text-sm">
              {errors.confirmPassword.type === "custom"
                ? t("passwordMismatch")
                : tCommon("required")}
            </p>
          ) : null}
          <button
            type="submit"
            disabled={isSubmitting || !token}
            className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white disabled:opacity-60"
          >
            {isSubmitting ? tCommon("loading") : t("resetPasswordAction")}
          </button>
        </form>
      )}
    </div>
  );
}
