"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff } from "lucide-react";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link, useRouter } from "@/i18n/navigation";
import { ApiError } from "@/lib/api/errors";
import { getUserDestination } from "@/lib/auth/routes";
import { useSession } from "@/lib/auth/session-context";

const signInSchema = z.object({
  email: z.email().max(254),
  password: z.string().min(1),
  totpCode: z.string().optional(),
  recoveryCode: z.string().optional(),
});

type SignInValues = z.infer<typeof signInSchema>;

function isSignInField(field: string): field is keyof SignInValues {
  return ["email", "password", "totpCode", "recoveryCode"].includes(field);
}

export function SignInForm() {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();
  const { signIn, status } = useSession();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [showAdminCode, setShowAdminCode] = useState(false);
  const [useRecovery, setUseRecovery] = useState(false);
  const {
    register,
    handleSubmit,
    setError,
    clearErrors,
    formState: { errors, isSubmitting },
  } = useForm<SignInValues>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    clearErrors();
    const credentials = {
      email: values.email.trim(),
      password: values.password,
      ...(showAdminCode && !useRecovery && values.totpCode
        ? { totpCode: values.totpCode.trim() }
        : {}),
      ...(showAdminCode && useRecovery && values.recoveryCode
        ? { recoveryCode: values.recoveryCode.trim() }
        : {}),
    };

    try {
      const user = await signIn(credentials);
      if (
        user.role === "admin" &&
        !credentials.totpCode &&
        !credentials.recoveryCode
      ) {
        router.replace("/admin/security/mfa");
        return;
      }
      router.replace(getUserDestination(user));
    } catch (error) {
      const apiError =
        error instanceof ApiError ? error : new ApiError("internal_error");
      setFormError(apiError);
      for (const fieldError of apiError.fieldErrors) {
        if (isSignInField(fieldError.field)) {
          setError(fieldError.field, {
            type: "server",
            message: fieldError.code,
          });
        }
      }
    }
  });

  return (
    <form className="space-y-5" onSubmit={onSubmit} noValidate>
      {formError ? <ApiErrorBanner error={formError} /> : null}
      <div>
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
          inputMode="email"
          aria-invalid={Boolean(errors.email)}
          aria-describedby={errors.email ? "email-error" : undefined}
          className="border-border text-ink min-h-12 w-full rounded-lg border bg-white px-4"
          {...register("email")}
        />
        {errors.email ? (
          <p id="email-error" className="text-danger mt-1 text-sm">
            {errors.email.type === "server"
              ? t("validationError")
              : tCommon("invalidEmail")}
          </p>
        ) : null}
      </div>
      <div>
        <label
          htmlFor="password"
          className="text-ink mb-2 block text-sm font-semibold"
        >
          {t("password")}
        </label>
        <div className="relative">
          <input
            id="password"
            type={passwordVisible ? "text" : "password"}
            autoComplete="current-password"
            aria-invalid={Boolean(errors.password)}
            aria-describedby={errors.password ? "password-error" : undefined}
            className="border-border text-ink min-h-12 w-full rounded-lg border bg-white px-4 pr-12"
            {...register("password")}
          />
          <button
            type="button"
            aria-label={
              passwordVisible
                ? tCommon("hidePassword")
                : tCommon("showPassword")
            }
            onClick={() => setPasswordVisible((visible) => !visible)}
            className="text-muted absolute inset-y-0 right-0 inline-flex w-12 items-center justify-center"
          >
            {passwordVisible ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        </div>
        {errors.password ? (
          <p id="password-error" className="text-danger mt-1 text-sm">
            {errors.password.type === "server"
              ? t("validationError")
              : tCommon("required")}
          </p>
        ) : null}
      </div>
      {showAdminCode ? (
        <div>
          <label
            htmlFor={useRecovery ? "recoveryCode" : "totpCode"}
            className="text-ink mb-2 block text-sm font-semibold"
          >
            {useRecovery ? t("recoveryCode") : t("totpCode")}
          </label>
          {useRecovery ? (
            <input
              id="recoveryCode"
              autoComplete="one-time-code"
              className="border-border text-ink min-h-12 w-full rounded-lg border bg-white px-4"
              {...register("recoveryCode")}
            />
          ) : (
            <input
              id="totpCode"
              inputMode="numeric"
              autoComplete="one-time-code"
              className="border-border text-ink min-h-12 w-full rounded-lg border bg-white px-4"
              {...register("totpCode")}
            />
          )}
          <button
            type="button"
            className="text-sage mt-2 text-sm font-semibold underline underline-offset-4"
            onClick={() => setUseRecovery((value) => !value)}
          >
            {useRecovery ? t("useAuthenticator") : t("useRecoveryCode")}
          </button>
        </div>
      ) : (
        <button
          type="button"
          className="text-sage text-sm font-semibold underline underline-offset-4"
          onClick={() => setShowAdminCode(true)}
        >
          {t("adminChallengeTitle")}
        </button>
      )}
      <div className="flex items-center justify-between gap-4 text-sm">
        <Link
          href="/password/forgot"
          className="text-sage font-semibold underline underline-offset-4"
        >
          {t("forgotPassword")}
        </Link>
        <Link
          href="/sign-up"
          className="text-sage font-semibold underline underline-offset-4"
        >
          {t("createAccount")}
        </Link>
      </div>
      <button
        type="submit"
        disabled={isSubmitting || status === "checking"}
        className="bg-sage inline-flex min-h-12 w-full items-center justify-center rounded-full px-6 font-semibold text-white transition-colors hover:bg-[#244b38] disabled:cursor-not-allowed disabled:opacity-60"
      >
        {isSubmitting ? tCommon("loading") : t("signInAction")}
      </button>
    </form>
  );
}
