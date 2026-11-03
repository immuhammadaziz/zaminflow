"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link } from "@/i18n/navigation";
import { ApiError, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";

const signUpSchema = z.object({
  role: z.enum(["farmer", "processor"]),
  fullName: z.string().refine((value) => {
    const length = Array.from(value.trim()).length;
    return length >= 2 && length <= 160;
  }),
  email: z.email().max(254),
  password: z.string().min(12).max(128),
  organizationName: z.string().refine((value) => {
    const length = Array.from(value.trim()).length;
    return length >= 2 && length <= 200;
  }),
  phone: z.string().regex(/^\+998[0-9]{9}$/),
  regionCode: z.string().min(1).max(32),
  stir: z.string().regex(/^[0-9]{9}$/),
});

type SignUpValues = z.infer<typeof signUpSchema>;
type SignUpField = keyof SignUpValues;

const fieldNames: ReadonlySet<string> = new Set([
  "role",
  "fullName",
  "email",
  "password",
  "organizationName",
  "phone",
  "regionCode",
  "stir",
]);

function isSignUpField(field: string): field is SignUpField {
  return fieldNames.has(field);
}

export function SignUpForm() {
  const t = useTranslations("auth");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [step, setStep] = useState<1 | 2>(1);
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [submitted, setSubmitted] = useState(false);
  const {
    register,
    handleSubmit,
    trigger,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<SignUpValues>({
    resolver: zodResolver(signUpSchema),
    defaultValues: {
      role: "farmer",
      fullName: "",
      email: "",
      password: "",
      organizationName: "",
      phone: "+998",
      regionCode: "",
      stir: "",
    },
  });

  const rolesQuery = useQuery({
    queryKey: ["signup-roles"],
    queryFn: async () => getApiData(await api.GET("/meta/roles")).items,
  });
  const regionsQuery = useQuery({
    queryKey: ["signup-regions"],
    queryFn: async () => getApiData(await api.GET("/meta/regions")).items,
  });

  const submit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      getApiData(await api.POST("/auth/sign-up", { body: values }));
      setSubmitted(true);
    } catch (error) {
      const apiError =
        error instanceof ApiError ? error : new ApiError("internal_error");
      setFormError(apiError);
      for (const fieldError of apiError.fieldErrors) {
        if (isSignUpField(fieldError.field)) {
          setError(fieldError.field, { type: "server" });
        }
      }
    }
  });

  const continueToBusinessDetails = async () => {
    const valid = await trigger(["role", "fullName", "email", "password"]);
    if (valid) setStep(2);
  };

  if (submitted) {
    return (
      <div className="space-y-5">
        <p
          role="status"
          className="rounded-card bg-sage-soft text-ink p-4 text-sm leading-6"
        >
          {t("registrationSuccess")}
        </p>
        <Link
          href="/sign-in"
          className="bg-sage inline-flex min-h-12 w-full items-center justify-center rounded-full px-6 font-semibold text-white"
        >
          {t("signInAction")}
        </Link>
      </div>
    );
  }

  if (rolesQuery.isError || regionsQuery.isError) {
    const error =
      rolesQuery.error instanceof ApiError
        ? rolesQuery.error
        : regionsQuery.error instanceof ApiError
          ? regionsQuery.error
          : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => {
            void rolesQuery.refetch();
            void regionsQuery.refetch();
          }}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }

  if (rolesQuery.isLoading || regionsQuery.isLoading) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const inputClass =
    "min-h-12 w-full rounded-lg border border-border bg-white px-4 text-ink";
  const labelClass = "mb-2 block text-sm font-semibold text-ink";

  return (
    <form className="space-y-5" onSubmit={submit} noValidate>
      {formError ? <ApiErrorBanner error={formError} /> : null}
      {step === 1 ? (
        <>
          <div>
            <label htmlFor="role" className={labelClass}>
              {t("role")}
            </label>
            <select id="role" className={inputClass} {...register("role")}>
              {rolesQuery.data?.map((role) => (
                <option key={role.value} value={role.value}>
                  {role.value === "farmer" ? t("farmer") : t("processor")}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="fullName" className={labelClass}>
              {t("fullName")}
            </label>
            <input
              id="fullName"
              autoComplete="name"
              className={inputClass}
              {...register("fullName")}
            />
            {errors.fullName ? (
              <FieldError message={t("validationError")} />
            ) : null}
          </div>
          <div>
            <label htmlFor="email" className={labelClass}>
              {t("email")}
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              className={inputClass}
              {...register("email")}
            />
            {errors.email ? (
              <FieldError message={tCommon("invalidEmail")} />
            ) : null}
          </div>
          <div>
            <label htmlFor="password" className={labelClass}>
              {t("password")}
            </label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              className={inputClass}
              {...register("password")}
            />
            <p className="text-muted mt-1 text-xs">{t("passwordGuidance")}</p>
            {errors.password ? (
              <FieldError message={t("passwordGuidance")} />
            ) : null}
          </div>
          <button
            type="button"
            onClick={() => void continueToBusinessDetails()}
            className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white"
          >
            {t("submitRegistration")}
          </button>
        </>
      ) : (
        <>
          <div>
            <label htmlFor="organizationName" className={labelClass}>
              {t("organizationName")}
            </label>
            <input
              id="organizationName"
              autoComplete="organization"
              className={inputClass}
              {...register("organizationName")}
            />
            {errors.organizationName ? (
              <FieldError message={t("validationError")} />
            ) : null}
          </div>
          <div>
            <label htmlFor="phone" className={labelClass}>
              {t("phone")}
            </label>
            <input
              id="phone"
              type="tel"
              inputMode="tel"
              autoComplete="tel"
              className={inputClass}
              {...register("phone")}
            />
            <p className="text-muted mt-1 text-xs">{t("phoneGuidance")}</p>
            {errors.phone ? <FieldError message={t("phoneGuidance")} /> : null}
          </div>
          <div>
            <label htmlFor="regionCode" className={labelClass}>
              {t("region")}
            </label>
            <select
              id="regionCode"
              className={inputClass}
              {...register("regionCode")}
            >
              <option value="">{tCommon("required")}</option>
              {regionsQuery.data?.map((region) => (
                <option key={region.code} value={region.code}>
                  {locale === "ru"
                    ? region.nameRu
                    : locale === "en"
                      ? region.nameEn
                      : region.nameUz}
                </option>
              ))}
            </select>
            {errors.regionCode ? (
              <FieldError message={t("validationError")} />
            ) : null}
          </div>
          <div>
            <label htmlFor="stir" className={labelClass}>
              {t("stir")}
            </label>
            <input
              id="stir"
              inputMode="numeric"
              autoComplete="off"
              className={inputClass}
              {...register("stir")}
            />
            {errors.stir ? <FieldError message={t("validationError")} /> : null}
          </div>
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setStep(1)}
              className="border-border min-h-12 flex-1 rounded-full border px-5 font-semibold"
            >
              {tCommon("back")}
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="bg-sage min-h-12 flex-1 rounded-full px-5 font-semibold text-white disabled:opacity-60"
            >
              {isSubmitting ? tCommon("loading") : t("submitAccount")}
            </button>
          </div>
        </>
      )}
    </form>
  );
}

function FieldError({ message }: Readonly<{ message: string }>) {
  return <p className="text-danger mt-1 text-sm">{message}</p>;
}
