"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link, useRouter } from "@/i18n/navigation";
import { ApiError, ensureApiSuccess, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type User = components["schemas"]["User"];
type DeletionRequest = components["schemas"]["AccountDeletionRequest"];
type DeletionStatus = DeletionRequest["status"] | "all";
type RecordValue = Record<string, unknown>;

function isRecord(value: unknown): value is RecordValue {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringValue(value: unknown): string {
  return typeof value === "string" ? value : "—";
}

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

export function ApplicationsClient() {
  const t = useTranslations("admin");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [page, setPage] = useState(1);
  const [role, setRole] = useState<"" | "farmer" | "processor">("");
  const [regionCode, setRegionCode] = useState("");
  const [activeFilters, setActiveFilters] = useState({
    role: "" as "" | "farmer" | "processor",
    regionCode: "",
  });
  const regionsQuery = useQuery({
    queryKey: ["regions"],
    queryFn: async () => getApiData(await api.GET("/meta/regions")).items,
  });
  const applicationsQuery = useQuery({
    queryKey: ["admin-applications", page, activeFilters],
    queryFn: async () => {
      const query = {
        page,
        pageSize: 20,
        ...(activeFilters.role ? { role: activeFilters.role } : {}),
        ...(activeFilters.regionCode
          ? { regionCode: activeFilters.regionCode }
          : {}),
      };
      return getApiData(
        await api.GET("/admin/applications", { params: { query } }),
      );
    },
  });

  if (applicationsQuery.isError || regionsQuery.isError) {
    const error =
      applicationsQuery.error instanceof ApiError
        ? applicationsQuery.error
        : regionsQuery.error instanceof ApiError
          ? regionsQuery.error
          : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => {
            void applicationsQuery.refetch();
            void regionsQuery.refetch();
          }}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!applicationsQuery.data || !regionsQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const data = applicationsQuery.data;
  const hasNext = data.page * data.pageSize < data.total;
  return (
    <div className="space-y-5">
      <form
        className="rounded-card border-border bg-surface grid gap-3 border p-4 sm:grid-cols-3 sm:items-end"
        onSubmit={(event) => {
          event.preventDefault();
          setPage(1);
          setActiveFilters({ role, regionCode });
        }}
      >
        <label className="text-ink text-sm font-semibold">
          {t("filterRole")}
          <select
            value={role}
            onChange={(event) =>
              setRole(event.target.value as "" | "farmer" | "processor")
            }
            className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
          >
            <option value="">{t("filterRole")}</option>
            <option value="farmer">{tAuth("farmer")}</option>
            <option value="processor">{tAuth("processor")}</option>
          </select>
        </label>
        <label className="text-ink text-sm font-semibold">
          {t("filterRegion")}
          <select
            value={regionCode}
            onChange={(event) => setRegionCode(event.target.value)}
            className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
          >
            <option value="">{t("filterRegion")}</option>
            {regionsQuery.data.map((region) => (
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
        <button
          type="submit"
          className="bg-sage min-h-11 rounded-full px-5 font-semibold text-white"
        >
          {t("filter")}
        </button>
      </form>

      {data.items.length ? (
        <div className="rounded-card border-border bg-surface overflow-x-auto border">
          <table className="w-full min-w-[42rem] border-collapse text-left text-sm">
            <thead className="bg-surface-soft text-muted text-xs tracking-wide uppercase">
              <tr>
                <th className="px-4 py-3">{t("applicationDetails")}</th>
                <th className="px-4 py-3">{t("filterRole")}</th>
                <th className="px-4 py-3">{t("filterRegion")}</th>
                <th className="px-4 py-3">{t("from")}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {data.items.map((applicant: User) => (
                <tr key={applicant.id} className="border-border border-t">
                  <td className="px-4 py-3">
                    <p className="text-ink font-semibold">
                      {applicant.fullName}
                    </p>
                    <p className="text-muted text-xs">{applicant.email}</p>
                  </td>
                  <td className="px-4 py-3">
                    {applicant.role === "farmer"
                      ? tAuth("farmer")
                      : tAuth("processor")}
                  </td>
                  <td className="px-4 py-3">{applicant.regionCode ?? "—"}</td>
                  <td className="px-4 py-3">
                    {formatDate(applicant.createdAt, locale)}
                  </td>
                  <td className="px-4 py-3">
                    <Link
                      href={`/admin/applications/${applicant.id}`}
                      className="text-sage font-semibold underline underline-offset-2"
                    >
                      {t("viewApplication")}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="rounded-card border-border bg-surface text-muted border p-6 text-sm">
          {t("noApplications")}
        </p>
      )}
      <div className="flex justify-end gap-3">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => setPage((value) => value - 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("back")}
        </button>
        <button
          type="button"
          disabled={!hasNext}
          onClick={() => setPage((value) => value + 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("continue")}
        </button>
      </div>
    </div>
  );
}

export function ApplicationReviewClient({
  userId,
}: Readonly<{ userId: string }>) {
  const t = useTranslations("admin");
  const tAdmin = useTranslations("admin");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState("");
  const [error, setError] = useState<ApiError | null>(null);
  const detailQuery = useQuery({
    queryKey: ["admin-application", userId],
    queryFn: async () =>
      getApiData(
        await api.GET("/admin/applications/{userId}", {
          params: { path: { userId } },
        }),
      ),
  });

  if (detailQuery.isError) {
    const apiError =
      detailQuery.error instanceof ApiError
        ? detailQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={apiError} />
        <button
          type="button"
          onClick={() => void detailQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!detailQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const { user, reviews } = detailQuery.data;
  const decide = async (decision: "approve" | "reject") => {
    if (decision === "approve" && !window.confirm(t("approveConfirm"))) return;
    if (decision === "reject" && !reason.trim()) return;
    setError(null);
    try {
      if (decision === "approve") {
        ensureApiSuccess(
          await api.POST("/admin/applications/{userId}/approve", {
            params: { path: { userId } },
          }),
        );
      } else {
        ensureApiSuccess(
          await api.POST("/admin/applications/{userId}/reject", {
            params: { path: { userId } },
            body: { reason: reason.trim() },
          }),
        );
      }
      await queryClient.invalidateQueries({ queryKey: ["admin-applications"] });
      router.replace("/admin/applications");
    } catch (cause) {
      setError(
        cause instanceof ApiError ? cause : new ApiError("internal_error"),
      );
    }
  };

  return (
    <div className="space-y-6">
      {error ? <ApiErrorBanner error={error} /> : null}
      <section className="rounded-card border-border bg-surface border p-5 sm:p-7">
        <h2 className="font-display text-ink text-2xl font-bold">
          {t("applicationDetails")}
        </h2>
        <dl className="mt-5 grid gap-4 sm:grid-cols-2">
          <DetailItem label={tAuth("fullName")} value={user.fullName} />
          <DetailItem label={tAuth("email")} value={user.email} />
          <DetailItem
            label={t("filterRole")}
            value={
              user.role === "farmer" ? tAuth("farmer") : tAuth("processor")
            }
          />
          <DetailItem
            label={t("filterRegion")}
            value={user.regionCode ?? "—"}
          />
          <DetailItem
            label={tAuth("organizationName")}
            value={user.organizationName ?? "—"}
          />
          <DetailItem label={tAuth("phone")} value={user.phone ?? "—"} />
          <DetailItem label={tAuth("stir")} value={user.stir ?? "—"} />
          <DetailItem
            label={t("from")}
            value={formatDate(user.createdAt, locale)}
          />
        </dl>
      </section>
      <section className="rounded-card border-border bg-surface border p-5">
        <h2 className="text-ink font-semibold">{t("reviewHistory")}</h2>
        {reviews.length ? (
          <ol className="mt-4 space-y-4">
            {reviews.map((review) => (
              <li key={review.id} className="border-sage border-l-2 pl-4">
                <p className="text-ink font-semibold">
                  {tAdmin(reviewDecisionKey(review.decision))}
                </p>
                {review.reason ? (
                  <p className="text-muted mt-1 text-sm">{review.reason}</p>
                ) : null}
                <p className="text-muted mt-1 text-xs">
                  {formatDate(review.createdAt, locale)}
                </p>
              </li>
            ))}
          </ol>
        ) : (
          <p className="text-muted mt-3 text-sm">{t("noApplications")}</p>
        )}
      </section>
      <section className="rounded-card border-border bg-surface space-y-4 border p-5">
        <label
          htmlFor="rejectReason"
          className="text-ink block text-sm font-semibold"
        >
          {t("rejectReason")}
        </label>
        <textarea
          id="rejectReason"
          rows={4}
          maxLength={2000}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          className="border-border w-full rounded-lg border bg-white p-3"
        />
        <div className="flex flex-col gap-3 sm:flex-row">
          <button
            type="button"
            onClick={() => void decide("approve")}
            className="bg-sage min-h-11 flex-1 rounded-full px-5 font-semibold text-white"
          >
            {t("approve")}
          </button>
          <button
            type="button"
            disabled={!reason.trim()}
            onClick={() => void decide("reject")}
            className="border-danger/40 text-danger min-h-11 flex-1 rounded-full border px-5 font-semibold disabled:opacity-50"
          >
            {t("reject")}
          </button>
        </div>
      </section>
    </div>
  );
}

function reviewDecisionKey(
  decision: string,
):
  | "reviewApproved"
  | "reviewRejected"
  | "reviewSuspended"
  | "reviewReinstated"
  | "reviewResubmitted" {
  switch (decision) {
    case "approved":
      return "reviewApproved";
    case "rejected":
      return "reviewRejected";
    case "suspended":
      return "reviewSuspended";
    case "reinstated":
      return "reviewReinstated";
    default:
      return "reviewResubmitted";
  }
}

function DetailItem({
  label,
  value,
}: Readonly<{ label: string; value: string }>) {
  return (
    <div>
      <dt className="text-muted text-xs font-semibold">{label}</dt>
      <dd className="text-ink mt-1 text-sm break-words">{value}</dd>
    </div>
  );
}

type AuditFilters = Readonly<{
  actorId: string;
  action: string;
  targetId: string;
  targetType: string;
  from: string;
  to: string;
}>;

export function AuditEventsClient() {
  const t = useTranslations("admin");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<AuditFilters>({
    actorId: "",
    action: "",
    targetId: "",
    targetType: "",
    from: "",
    to: "",
  });
  const [draft, setDraft] = useState<AuditFilters>(filters);
  const auditQuery = useQuery({
    queryKey: ["admin-audit-events", page, filters],
    queryFn: async () => {
      const query = {
        page,
        pageSize: 20,
        ...(filters.actorId ? { actorId: filters.actorId } : {}),
        ...(filters.action ? { action: filters.action } : {}),
        ...(filters.targetId ? { targetId: filters.targetId } : {}),
        ...(filters.targetType ? { targetType: filters.targetType } : {}),
        ...(filters.from ? { from: new Date(filters.from).toISOString() } : {}),
        ...(filters.to ? { to: new Date(filters.to).toISOString() } : {}),
      };
      return getApiData(
        await api.GET("/admin/audit-events", { params: { query } }),
      );
    },
  });

  if (auditQuery.isError) {
    const error =
      auditQuery.error instanceof ApiError
        ? auditQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void auditQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!auditQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const data = auditQuery.data;
  return (
    <div className="space-y-5">
      <form
        className="rounded-card border-border bg-surface grid gap-3 border p-4 sm:grid-cols-2 lg:grid-cols-4"
        onSubmit={(event) => {
          event.preventDefault();
          setPage(1);
          setFilters(draft);
        }}
      >
        {(
          [
            ["actorId", t("actorId")],
            ["action", t("action")],
            ["targetId", t("targetId")],
            ["targetType", t("targetType")],
            ["from", t("from")],
            ["to", t("to")],
          ] as const
        ).map(([field, label]) => (
          <label key={field} className="text-ink text-sm font-semibold">
            {label}
            <input
              type={
                field === "from" || field === "to" ? "datetime-local" : "text"
              }
              value={draft[field]}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  [field]: event.target.value,
                }))
              }
              className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
            />
          </label>
        ))}
        <button
          type="submit"
          className="bg-sage min-h-11 rounded-full px-5 font-semibold text-white sm:col-span-2 lg:col-span-4"
        >
          {t("filter")}
        </button>
      </form>
      {data.items.length ? (
        <div className="rounded-card border-border bg-surface overflow-x-auto border">
          <table className="w-full min-w-[48rem] border-collapse text-left text-sm">
            <thead className="bg-surface-soft text-muted text-xs tracking-wide uppercase">
              <tr>
                <th className="px-4 py-3">{t("action")}</th>
                <th className="px-4 py-3">{t("actorId")}</th>
                <th className="px-4 py-3">{t("targetType")}</th>
                <th className="px-4 py-3">{t("targetId")}</th>
                <th className="px-4 py-3">{t("from")}</th>
                <th className="px-4 py-3">{tCommon("requestId")}</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((event, index) => {
                const row = isRecord(event) ? event : {};
                return (
                  <tr
                    key={
                      stringValue(row.id) === "—"
                        ? `event-${index}`
                        : stringValue(row.id)
                    }
                    className="border-border border-t"
                  >
                    <td className="text-ink px-4 py-3 font-semibold">
                      {stringValue(row.action)}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {stringValue(row.actorId)}
                    </td>
                    <td className="px-4 py-3">{stringValue(row.targetType)}</td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {stringValue(row.targetId)}
                    </td>
                    <td className="px-4 py-3">
                      {typeof row.createdAt === "string"
                        ? formatDate(row.createdAt, locale)
                        : "—"}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {stringValue(row.requestId)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="rounded-card border-border bg-surface text-muted border p-6 text-sm">
          {t("noAuditEvents")}
        </p>
      )}
      <div className="flex justify-end gap-3">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => setPage((value) => value - 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("back")}
        </button>
        <button
          type="button"
          disabled={data.page * data.pageSize >= data.total}
          onClick={() => setPage((value) => value + 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("continue")}
        </button>
      </div>
    </div>
  );
}

export function DeletionRequestsClient() {
  const t = useTranslations("admin");
  const tAuth = useTranslations("auth");
  const tStatus = useTranslations("status");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<DeletionStatus>("requested");
  const [rejecting, setRejecting] = useState<string | null>(null);
  const [reason, setReason] = useState("");
  const [actionError, setActionError] = useState<ApiError | null>(null);
  const requestQuery = useQuery({
    queryKey: ["admin-deletion-requests", page, status],
    queryFn: async () =>
      getApiData(
        await api.GET("/admin/deletion-requests", {
          params: { query: { page, pageSize: 20, status } },
        }),
      ),
  });

  if (requestQuery.isError) {
    const error =
      requestQuery.error instanceof ApiError
        ? requestQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
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
  if (!requestQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const decide = async (
    request: DeletionRequest,
    decision: "approve" | "reject",
  ) => {
    if (decision === "approve" && !window.confirm(t("confirmDeletion"))) return;
    if (decision === "reject" && (rejecting !== request.id || !reason.trim()))
      return;
    setActionError(null);
    try {
      if (decision === "approve") {
        ensureApiSuccess(
          await api.POST("/admin/deletion-requests/{requestId}/approve", {
            params: { path: { requestId: request.id } },
          }),
        );
      } else {
        ensureApiSuccess(
          await api.POST("/admin/deletion-requests/{requestId}/reject", {
            params: { path: { requestId: request.id } },
            body: { reason: reason.trim() },
          }),
        );
        setRejecting(null);
        setReason("");
      }
      await queryClient.invalidateQueries({
        queryKey: ["admin-deletion-requests"],
      });
    } catch (error) {
      setActionError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  };

  const data = requestQuery.data;
  return (
    <div className="space-y-5">
      <p className="rounded-card border-warning/30 bg-warning/5 text-ink border p-4 text-sm leading-6">
        {t("deletionNotice")}
      </p>
      {actionError ? <ApiErrorBanner error={actionError} /> : null}
      <label className="text-ink flex max-w-sm items-center gap-3 text-sm font-semibold">
        {tStatus("pending")}
        <select
          value={status}
          onChange={(event) => {
            setStatus(event.target.value as DeletionStatus);
            setPage(1);
          }}
          className="border-border min-h-11 flex-1 rounded-lg border bg-white px-3 font-normal"
        >
          {(
            ["requested", "approved", "completed", "rejected", "all"] as const
          ).map((value) => (
            <option key={value} value={value}>
              {value === "all" ? t("deletionTitle") : tStatus(value)}
            </option>
          ))}
        </select>
      </label>
      {data.items.length ? (
        <div className="space-y-4">
          {data.items.map((request) => (
            <article
              key={request.id}
              className="rounded-card border-border bg-surface space-y-4 border p-5"
            >
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <DetailItem
                  label={t("applicationDetails")}
                  value={request.user.fullName}
                />
                <DetailItem label={tAuth("email")} value={request.user.email} />
                <DetailItem
                  label={tStatus("pending")}
                  value={tStatus(request.status)}
                />
                <DetailItem
                  label={t("from")}
                  value={formatDate(request.requestedAt, locale)}
                />
              </div>
              {request.status === "requested" ? (
                <>
                  {rejecting === request.id ? (
                    <label className="text-ink block text-sm font-semibold">
                      {t("deletionReason")}
                      <textarea
                        rows={3}
                        maxLength={2000}
                        value={reason}
                        onChange={(event) => setReason(event.target.value)}
                        className="border-border mt-2 w-full rounded-lg border bg-white p-3 font-normal"
                      />
                    </label>
                  ) : null}
                  <div className="flex flex-wrap gap-3">
                    <button
                      type="button"
                      onClick={() => void decide(request, "approve")}
                      className="bg-sage min-h-10 rounded-full px-4 font-semibold text-white"
                    >
                      {t("approveDeletion")}
                    </button>
                    {rejecting === request.id ? (
                      <>
                        <button
                          type="button"
                          disabled={!reason.trim()}
                          onClick={() => void decide(request, "reject")}
                          className="border-danger/40 text-danger min-h-10 rounded-full border px-4 font-semibold disabled:opacity-50"
                        >
                          {t("rejectDeletion")}
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setRejecting(null);
                            setReason("");
                          }}
                          className="border-border min-h-10 rounded-full border px-4 font-semibold"
                        >
                          {tCommon("cancel")}
                        </button>
                      </>
                    ) : (
                      <button
                        type="button"
                        onClick={() => {
                          setRejecting(request.id);
                          setReason("");
                        }}
                        className="border-danger/40 text-danger min-h-10 rounded-full border px-4 font-semibold"
                      >
                        {t("rejectDeletion")}
                      </button>
                    )}
                  </div>
                </>
              ) : null}
            </article>
          ))}
        </div>
      ) : (
        <p className="rounded-card border-border bg-surface text-muted border p-6 text-sm">
          {t("noDeletionRequests")}
        </p>
      )}
      <div className="flex justify-end gap-3">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => setPage((value) => value - 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("back")}
        </button>
        <button
          type="button"
          disabled={data.page * data.pageSize >= data.total}
          onClick={() => setPage((value) => value + 1)}
          className="border-border min-h-10 rounded-full border px-4 font-semibold disabled:opacity-40"
        >
          {tCommon("continue")}
        </button>
      </div>
    </div>
  );
}
