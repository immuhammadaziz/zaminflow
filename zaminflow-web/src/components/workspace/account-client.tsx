"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { ApiError, ensureApiSuccess, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type User = components["schemas"]["User"];
type ProfileValues = {
  fullName: string;
  phone: string;
  organizationName: string;
  regionCode: string;
  stir: string;
};

const profileSchema = z.object({
  fullName: z.string().trim().min(2).max(160),
  phone: z.union([z.literal(""), z.string().regex(/^\+998\d{9}$/)]),
  organizationName: z.string().trim().max(160),
  regionCode: z.string().max(32),
  stir: z.union([z.literal(""), z.string().regex(/^\d{9}$/)]),
});

function formatDate(value: string, locale: string): string {
  const date = new Date(value);
  return Number.isFinite(date.getTime())
    ? new Intl.DateTimeFormat(locale, {
        dateStyle: "medium",
        timeStyle: "short",
        timeZone: "Asia/Tashkent",
      }).format(date)
    : value;
}

function Panel({
  title,
  children,
}: Readonly<{ title: string; children: React.ReactNode }>) {
  return (
    <section className="rounded-card border-border bg-surface border p-5 sm:p-7">
      <h2 className="font-display text-ink text-xl font-bold">{title}</h2>
      <div className="mt-5">{children}</div>
    </section>
  );
}

function LoadingState() {
  const t = useTranslations("common");
  return (
    <p
      role="status"
      className="rounded-card bg-surface-soft text-muted p-4 text-sm"
    >
      {t("loading")}
    </p>
  );
}

export function ProfileClient() {
  const t = useTranslations("account");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const tStatus = useTranslations("status");
  const locale = useLocale();
  const { api, updateUser } = useSession();
  const queryClient = useQueryClient();
  const [success, setSuccess] = useState("");
  const profileQuery = useQuery({
    queryKey: ["account-profile"],
    queryFn: async () => getApiData(await api.GET("/me")).user,
  });
  const regionsQuery = useQuery({
    queryKey: ["regions"],
    queryFn: async () => getApiData(await api.GET("/meta/regions")).items,
  });

  if (profileQuery.isError || regionsQuery.isError) {
    const error =
      profileQuery.error instanceof ApiError
        ? profileQuery.error
        : regionsQuery.error instanceof ApiError
          ? regionsQuery.error
          : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => {
            void profileQuery.refetch();
            void regionsQuery.refetch();
          }}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!profileQuery.data || !regionsQuery.data) return <LoadingState />;

  return (
    <ProfileForm
      key={`${profileQuery.data.id}:${profileQuery.data.updatedAt ?? ""}`}
      user={profileQuery.data}
      regions={regionsQuery.data}
      locale={locale}
      onSaved={(user) => {
        updateUser(user);
        void queryClient.invalidateQueries({ queryKey: ["account-profile"] });
        setSuccess(t("profileSaved"));
      }}
      onResubmitted={(user) => {
        updateUser(user);
        setSuccess(t("resubmitted"));
      }}
      success={success}
      setSuccess={setSuccess}
      t={t}
      tAuth={tAuth}
      tCommon={tCommon}
      tStatus={tStatus}
      api={api}
    />
  );
}

function ProfileForm({
  user,
  regions,
  locale,
  onSaved,
  onResubmitted,
  success,
  setSuccess,
  t,
  tAuth,
  tCommon,
  tStatus,
  api,
}: Readonly<{
  user: User;
  regions: components["schemas"]["Region"][];
  locale: string;
  onSaved: (user: User) => void;
  onResubmitted: (user: User) => void;
  success: string;
  setSuccess: (message: string) => void;
  t: ReturnType<typeof useTranslations<"account">>;
  tAuth: ReturnType<typeof useTranslations<"auth">>;
  tCommon: ReturnType<typeof useTranslations<"common">>;
  tStatus: ReturnType<typeof useTranslations<"status">>;
  api: ReturnType<typeof useSession>["api"];
}>) {
  const [error, setError] = useState<ApiError | null>(null);
  const [resubmitError, setResubmitError] = useState<ApiError | null>(null);
  const [resubmitSuccess, setResubmitSuccess] = useState(false);
  const queryClient = useQueryClient();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ProfileValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      fullName: user.fullName,
      phone: user.phone ?? "",
      organizationName: user.organizationName ?? "",
      regionCode: user.regionCode ?? "",
      stir: user.stir ?? "",
    },
  });

  const onSubmit = handleSubmit(async (values) => {
    setError(null);
    setSuccess("");
    const body: components["schemas"]["ProfileUpdate"] = {
      fullName: values.fullName.trim(),
      ...(values.phone.trim() ? { phone: values.phone.trim() } : {}),
      ...(values.organizationName.trim()
        ? { organizationName: values.organizationName.trim() }
        : {}),
      ...(values.regionCode ? { regionCode: values.regionCode } : {}),
      ...(values.stir.trim() ? { stir: values.stir.trim() } : {}),
    };
    try {
      const updatedUser = getApiData(await api.PATCH("/me", { body })).user;
      onSaved(updatedUser);
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause : new ApiError("internal_error"),
      );
    }
  });

  const resubmit = async () => {
    setResubmitError(null);
    setResubmitSuccess(false);
    try {
      ensureApiSuccess(await api.POST("/me/resubmit", { body: {} }));
      const updatedUser = getApiData(await api.GET("/me")).user;
      onResubmitted(updatedUser);
      setResubmitSuccess(true);
      await queryClient.invalidateQueries({ queryKey: ["account-profile"] });
    } catch (cause) {
      setResubmitError(
        cause instanceof ApiError ? cause : new ApiError("internal_error"),
      );
    }
  };

  const inputClass =
    "mt-2 min-h-12 w-full rounded-lg border border-border bg-white px-4 text-ink";
  const field = (
    name: keyof ProfileValues,
    label: string,
    options: Readonly<{ type?: string; maxLength?: number }> = {},
  ) => (
    <label className="text-ink block text-sm font-semibold">
      {label}
      <input
        {...register(name)}
        type={options.type ?? "text"}
        maxLength={options.maxLength}
        aria-invalid={Boolean(errors[name])}
        className={inputClass}
      />
      {errors[name] ? (
        <span className="text-danger mt-1 block text-sm font-normal">
          {tAuth("validationError")}
        </span>
      ) : null}
    </label>
  );

  return (
    <div className="space-y-5">
      <Panel title={t("profileTitle")}>
        <div className="mb-5 flex flex-wrap items-center gap-3 text-sm">
          <span className="bg-sage-soft text-sage rounded-full px-3 py-1 font-semibold">
            {tStatus(user.accountStatus)}
          </span>
          <span className="text-muted">{user.email}</span>
        </div>
        {error ? <ApiErrorBanner error={error} className="mb-4" /> : null}
        {success ? (
          <p role="status" className="text-sage mb-4 text-sm font-semibold">
            {success}
          </p>
        ) : null}
        <form className="space-y-4" onSubmit={onSubmit} noValidate>
          {field("fullName", tAuth("fullName"), { maxLength: 160 })}
          <label className="text-ink block text-sm font-semibold">
            {tAuth("phone")}
            <input
              {...register("phone")}
              type="tel"
              autoComplete="tel"
              placeholder="+998901234567"
              aria-invalid={Boolean(errors.phone)}
              className={inputClass}
            />
            {errors.phone ? (
              <span className="text-danger mt-1 block text-sm font-normal">
                {tAuth("phoneGuidance")}
              </span>
            ) : null}
          </label>
          {field("organizationName", tAuth("organizationName"), {
            maxLength: 160,
          })}
          <label className="text-ink block text-sm font-semibold">
            {tAuth("region")}
            <select {...register("regionCode")} className={inputClass}>
              <option value="">{tCommon("selectRegion")}</option>
              {regions.map((region) => (
                <option key={region.code} value={region.code}>
                  {locale === "ru"
                    ? region.nameRu
                    : locale === "en"
                      ? region.nameEn
                      : region.nameUz}
                </option>
              ))}
            </select>
          </label>
          {field("stir", tAuth("stir"), { maxLength: 9 })}
          <button
            type="submit"
            disabled={isSubmitting}
            className="bg-sage min-h-12 rounded-full px-6 font-semibold text-white disabled:opacity-60"
          >
            {isSubmitting ? tCommon("loading") : t("saveProfile")}
          </button>
        </form>
      </Panel>

      {user.accountStatus === "rejected" ? (
        <Panel title={t("resubmitTitle")}>
          <p className="text-muted text-sm leading-6">
            {t("rejectionReason")}: {user.rejectionReason || "—"}
          </p>
          <p className="text-muted mt-3 text-sm leading-6">
            {t("resubmitDescription")}
          </p>
          {resubmitError ? (
            <ApiErrorBanner error={resubmitError} className="mt-4" />
          ) : null}
          {resubmitSuccess ? (
            <p role="status" className="text-sage mt-4 text-sm font-semibold">
              {t("resubmitted")}
            </p>
          ) : null}
          <button
            type="button"
            disabled={resubmitSuccess}
            onClick={() => void resubmit()}
            className="bg-sage mt-5 min-h-12 rounded-full px-6 font-semibold text-white disabled:opacity-60"
          >
            {t("resubmitAction")}
          </button>
        </Panel>
      ) : null}
    </div>
  );
}

export function SecurityClient() {
  const t = useTranslations("account");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [passwordError, setPasswordError] = useState<ApiError | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const sessionsQuery = useQuery({
    queryKey: ["account-sessions", page],
    queryFn: async () =>
      getApiData(
        await api.GET("/auth/sessions", {
          params: { query: { page, pageSize: 10 } },
        }),
      ),
  });
  const revokeMutation = useMutation({
    mutationFn: async (sessionId: string) =>
      ensureApiSuccess(
        await api.DELETE("/auth/sessions/{sessionId}", {
          params: { path: { sessionId } },
        }),
      ),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["account-sessions"] }),
  });

  if (sessionsQuery.isError) {
    const error =
      sessionsQuery.error instanceof ApiError
        ? sessionsQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void sessionsQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!sessionsQuery.data) return <LoadingState />;

  const sessions = sessionsQuery.data.items;
  const hasNext =
    sessionsQuery.data.page * sessionsQuery.data.pageSize <
    sessionsQuery.data.total;
  return (
    <div className="space-y-5">
      <Panel title={t("sessionsTitle")}>
        <p className="text-muted mb-4 text-sm leading-6">
          {t("sessionCaution")}
        </p>
        {revokeMutation.error ? (
          <ApiErrorBanner
            error={
              revokeMutation.error instanceof ApiError
                ? revokeMutation.error
                : new ApiError("internal_error")
            }
            className="mb-4"
          />
        ) : null}
        {sessions.length === 0 ? (
          <p className="text-muted text-sm">{t("noSessions")}</p>
        ) : (
          <ul className="divide-border divide-y">
            {sessions.map((session) => (
              <li
                key={session.id}
                className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="text-sm">
                  <p className="text-ink font-semibold">
                    {t("sessionCreated")}:{" "}
                    {formatDate(session.createdAt, locale)}
                  </p>
                  <p className="text-muted mt-1">
                    {t("sessionLastUsed")}:{" "}
                    {formatDate(session.lastUsedAt, locale)}
                  </p>
                </div>
                <button
                  type="button"
                  disabled={revokeMutation.isPending}
                  onClick={() => {
                    if (window.confirm(t("revokeConfirm")))
                      revokeMutation.mutate(session.id);
                  }}
                  className="border-danger/40 text-danger min-h-10 rounded-full border px-4 text-sm font-semibold disabled:opacity-60"
                >
                  {t("revokeSession")}
                </button>
              </li>
            ))}
          </ul>
        )}
        {sessionsQuery.data.total > 10 ? (
          <div className="mt-4 flex items-center justify-between">
            <button
              type="button"
              disabled={page <= 1}
              onClick={() => setPage((value) => value - 1)}
              className="border-border min-h-10 rounded-full border px-4 disabled:opacity-50"
            >
              {tCommon("back")}
            </button>
            <span className="text-muted text-sm">
              {page} /{" "}
              {Math.ceil(
                sessionsQuery.data.total / sessionsQuery.data.pageSize,
              )}
            </span>
            <button
              type="button"
              disabled={!hasNext}
              onClick={() => setPage((value) => value + 1)}
              className="border-border min-h-10 rounded-full border px-4 disabled:opacity-50"
            >
              {tCommon("continue")}
            </button>
          </div>
        ) : null}
      </Panel>

      <PasswordChangeForm
        api={api}
        t={t}
        tAuth={tAuth}
        tCommon={tCommon}
        error={passwordError}
        setError={setPasswordError}
        success={passwordSuccess}
        setSuccess={setPasswordSuccess}
      />
    </div>
  );
}

function PasswordChangeForm({
  api,
  t,
  tAuth,
  tCommon,
  error,
  setError,
  success,
  setSuccess,
}: Readonly<{
  api: ReturnType<typeof useSession>["api"];
  t: ReturnType<typeof useTranslations<"account">>;
  tAuth: ReturnType<typeof useTranslations<"auth">>;
  tCommon: ReturnType<typeof useTranslations<"common">>;
  error: ApiError | null;
  setError: (error: ApiError | null) => void;
  success: boolean;
  setSuccess: (success: boolean) => void;
}>) {
  const schema = z.object({
    currentPassword: z.string().min(1),
    newPassword: z.string().min(12).max(128),
  });
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { currentPassword: "", newPassword: "" },
  });
  const onSubmit = handleSubmit(async (values) => {
    setError(null);
    setSuccess(false);
    try {
      ensureApiSuccess(
        await api.POST("/auth/change-password", { body: values }),
      );
      reset();
      setSuccess(true);
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause : new ApiError("internal_error"),
      );
    }
  });
  const inputClass =
    "mt-2 min-h-12 w-full rounded-lg border border-border bg-white px-4 text-ink";

  return (
    <Panel title={t("changePasswordTitle")}>
      <p className="text-muted mb-4 text-sm leading-6">
        {t("passwordChangeCaution")}
      </p>
      {error ? <ApiErrorBanner error={error} className="mb-4" /> : null}
      {success ? (
        <p role="status" className="text-sage mb-4 text-sm font-semibold">
          {t("passwordChanged")}
        </p>
      ) : null}
      <form className="max-w-xl space-y-4" onSubmit={onSubmit} noValidate>
        <label className="text-ink block text-sm font-semibold">
          {t("currentPassword")}
          <input
            {...register("currentPassword")}
            type="password"
            autoComplete="current-password"
            aria-invalid={Boolean(errors.currentPassword)}
            className={inputClass}
          />
        </label>
        <label className="text-ink block text-sm font-semibold">
          {t("newPassword")}
          <input
            {...register("newPassword")}
            type="password"
            autoComplete="new-password"
            aria-invalid={Boolean(errors.newPassword)}
            className={inputClass}
          />
          {errors.newPassword ? (
            <span className="text-danger mt-1 block text-sm font-normal">
              {tAuth("passwordGuidance")}
            </span>
          ) : null}
        </label>
        <button
          type="submit"
          disabled={isSubmitting}
          className="bg-sage min-h-12 rounded-full px-6 font-semibold text-white disabled:opacity-60"
        >
          {isSubmitting ? tCommon("loading") : t("changePasswordAction")}
        </button>
      </form>
    </Panel>
  );
}

export function DeletionClient() {
  const t = useTranslations("account");
  const tCommon = useTranslations("common");
  const tStatus = useTranslations("status");
  const locale = useLocale();
  const { api } = useSession();
  const queryClient = useQueryClient();
  const [error, setError] = useState<ApiError | null>(null);
  const [requested, setRequested] = useState(false);
  const requestQuery = useQuery({
    queryKey: ["account-deletion-request"],
    queryFn: async () => {
      try {
        return getApiData(await api.GET("/me/deletion-request")).request;
      } catch (cause) {
        if (cause instanceof ApiError && cause.code === "not_found")
          return null;
        throw cause;
      }
    },
  });

  if (requestQuery.isError) {
    const requestError =
      requestQuery.error instanceof ApiError
        ? requestQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={requestError} />
        <button
          type="button"
          onClick={() => void requestQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (requestQuery.data === undefined) return <LoadingState />;

  const requestDeletion = async () => {
    setError(null);
    setRequested(false);
    if (!window.confirm(t("deletionConfirm"))) return;
    try {
      ensureApiSuccess(await api.DELETE("/me"));
      setRequested(true);
      await queryClient.invalidateQueries({
        queryKey: ["account-deletion-request"],
      });
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause : new ApiError("internal_error"),
      );
    }
  };

  return (
    <Panel title={t("deletionTitle")}>
      <div className="border-warning/40 bg-warning/5 text-ink rounded-lg border p-4 text-sm leading-6">
        {t("deletionExplanation")}
      </div>
      {error ? <ApiErrorBanner error={error} className="mt-4" /> : null}
      {requested ? (
        <p role="status" className="text-sage mt-4 text-sm font-semibold">
          {t("deletionRequested")}
        </p>
      ) : null}
      {requestQuery.data ? (
        <dl className="mt-5 grid gap-2 text-sm sm:grid-cols-[auto_1fr]">
          <dt className="text-ink font-semibold">{t("requestStatus")}</dt>
          <dd className="text-muted">{tStatus(requestQuery.data.status)}</dd>
          <dt className="text-ink font-semibold">{t("requestDate")}</dt>
          <dd className="text-muted">
            {formatDate(requestQuery.data.requestedAt, locale)}
          </dd>
          {requestQuery.data.reason ? (
            <>
              <dt className="text-ink font-semibold">{t("reviewReason")}</dt>
              <dd className="text-muted">{requestQuery.data.reason}</dd>
            </>
          ) : null}
        </dl>
      ) : (
        <p className="text-muted mt-5 text-sm">{t("noDeletionRequest")}</p>
      )}
      {!requestQuery.data || requestQuery.data.status === "rejected" ? (
        <button
          type="button"
          onClick={() => void requestDeletion()}
          className="border-danger/40 text-danger mt-6 min-h-12 rounded-full border px-6 font-semibold"
        >
          {t("requestDeletion")}
        </button>
      ) : null}
    </Panel>
  );
}
