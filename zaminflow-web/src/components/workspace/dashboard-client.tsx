"use client";

import { useQuery } from "@tanstack/react-query";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link } from "@/i18n/navigation";
import { ApiError, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type Role = "farmer" | "processor" | "admin";
type Dashboard = components["schemas"]["Dashboard"];
type RecordValue = Record<string, unknown>;
type Period = "7d" | "30d" | "90d" | "12m";

const KPI_KEYS: Readonly<Record<Role, readonly string[]>> = {
  farmer: [
    "milkSoldLiters",
    "revenueUzs",
    "activeListings",
    "ordersAwaitingResponse",
    "ownAveragePricePerLiter",
    "platformAveragePricePerLiter",
  ],
  processor: [
    "volumeSourcedLiters",
    "spendUzs",
    "activeSuppliers",
    "averagePricePerLiter",
    "openOrders",
    "qualityAcceptanceRate",
  ],
  admin: ["accountsAwaitingApproval"],
};

function asRecord(value: unknown): RecordValue | null {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as RecordValue)
    : null;
}

function asRows(value: unknown): readonly unknown[] {
  return Array.isArray(value) ? value : [];
}

function asText(value: unknown): string | null {
  return typeof value === "string" ? value : null;
}

function asNumber(value: unknown): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function periodLabelKey(period: Period): string {
  return `period${period}`;
}

function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(
    value,
  );
}

function formatDate(value: string, locale: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return value;
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeZone: "Asia/Tashkent",
  }).format(date);
}

function formatKpi(
  key: string,
  value: unknown,
  unit: unknown,
  locale: string,
  t: (key: string) => string,
): string {
  if (value === null || value === undefined) return "—";
  const number = asNumber(value);
  if (number === null) return typeof value === "string" ? value : "—";
  if (key === "qualityAcceptanceRate") {
    return new Intl.NumberFormat(locale, {
      style: "percent",
      maximumFractionDigits: 1,
    }).format(number);
  }
  const formatted = formatNumber(number, locale);
  if (unit === "UZS") return `${formatted} UZS`;
  if (unit === "UZS/liter") return `${formatted} UZS/L`;
  if (unit === "liters") return `${formatted} ${t("liters")}`;
  return formatted;
}

function chartPoints(value: unknown, key: string, locale: string) {
  return asRows(value).flatMap((row) => {
    const record = asRecord(row);
    const period = asText(record?.period);
    const numeric = asNumber(record?.[key]);
    return period && numeric !== null
      ? [{ period: formatDate(period, locale), value: numeric }]
      : [];
  });
}

export function DashboardClient({ role }: Readonly<{ role: Role }>) {
  const t = useTranslations("dashboard");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [period, setPeriod] = useState<Period>("30d");
  const dashboardQuery = useQuery({
    queryKey: ["dashboard", role, period],
    queryFn: async (): Promise<Dashboard> => {
      if (role === "farmer") {
        return getApiData(
          await api.GET("/dashboards/farmer", {
            params: { query: { period } },
          }),
        );
      }
      if (role === "processor") {
        return getApiData(
          await api.GET("/dashboards/processor", {
            params: { query: { period } },
          }),
        );
      }
      return getApiData(
        await api.GET("/dashboards/admin", { params: { query: { period } } }),
      );
    },
  });

  if (dashboardQuery.isError) {
    const error =
      dashboardQuery.error instanceof ApiError
        ? dashboardQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        {typeof navigator !== "undefined" && !navigator.onLine ? (
          <p
            role="status"
            className="rounded-card bg-surface-soft text-muted p-4 text-sm"
          >
            {tCommon("offline")}
          </p>
        ) : null}
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void dashboardQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }

  if (!dashboardQuery.data) {
    return (
      <div role="status" aria-label={tCommon("loading")} className="space-y-5">
        <div className="rounded-card bg-surface-soft h-12 animate-pulse" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }, (_, index) => (
            <div
              key={index}
              className="rounded-card bg-surface-soft h-28 animate-pulse"
            />
          ))}
        </div>
      </div>
    );
  }

  const dashboard = dashboardQuery.data;
  const kpis = dashboard.kpis;
  const charts = dashboard.charts;
  const tables = dashboard.tables;
  const completed = chartPoints(charts.completedVolume, "liters", locale);
  const signups = chartPoints(charts.signups, "signups", locale);

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="text-muted text-sm">
          <span>
            {t("periodFrom")}: {formatDate(dashboard.period.from, locale)}
          </span>
          <span className="mx-2" aria-hidden="true">
            ·
          </span>
          <span>
            {t("periodTo")}: {formatDate(dashboard.period.to, locale)}
          </span>
        </div>
        <label className="text-ink flex items-center gap-3 text-sm font-semibold">
          {t("period")}
          <select
            value={period}
            onChange={(event) => setPeriod(event.target.value as Period)}
            className="border-border min-h-11 rounded-lg border bg-white px-3"
          >
            {(["7d", "30d", "90d", "12m"] as const).map((range) => (
              <option key={range} value={range}>
                {t(periodLabelKey(range))}
              </option>
            ))}
          </select>
        </label>
      </div>

      <section
        aria-label={t("period")}
        className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
      >
        {KPI_KEYS[role].map((key) => {
          const metric = asRecord(kpis[key]);
          if (!metric) return null;
          return (
            <article
              key={key}
              className="rounded-card border-border bg-surface border p-5"
            >
              <h2 className="text-muted text-sm font-semibold">{t(key)}</h2>
              <p className="text-ink mt-3 text-2xl font-bold tracking-tight">
                {formatKpi(key, metric.value, metric.unit, locale, t)}
              </p>
              {role === "admin" && key === "accountsAwaitingApproval" ? (
                <OldestApplicant
                  value={metric.oldestApplicant}
                  locale={locale}
                  t={t}
                />
              ) : null}
            </article>
          );
        })}
      </section>

      <section className="grid gap-5 lg:grid-cols-2">
        <ChartCard
          title={t("completedVolume")}
          points={completed}
          empty={t("empty")}
        />
        {role === "admin" ? (
          <ChartCard title={t("signups")} points={signups} empty={t("empty")} />
        ) : null}
      </section>

      {role !== "admin" ? (
        <section className="grid gap-5 lg:grid-cols-2">
          <RowsCard
            title={t("upcomingOrders")}
            rows={asRows(tables.upcomingOrders)}
            fields={[
              {
                key: "id",
                label: t("order"),
                linkPrefix:
                  role === "farmer" ? "/farmer/orders/" : "/processor/orders/",
              },
              { key: "status", label: t("status"), status: true },
              { key: "liters", label: t("liters") },
              { key: "expectedPickupAt", label: t("pickup"), date: true },
            ]}
            empty={t("empty")}
            locale={locale}
          />
          {role === "farmer" ? (
            <RowsCard
              title={t("recentActivity")}
              rows={asRows(tables.recentActivity)}
              fields={[
                { key: "action", label: t("recentActivity"), status: true },
                { key: "createdAt", label: t("period"), date: true },
              ]}
              empty={t("empty")}
              locale={locale}
            />
          ) : (
            <RowsCard
              title={t("supplyByRegion")}
              rows={asRows(charts.supplyByRegion)}
              fields={[
                { key: "regionName", label: t("region") },
                { key: "liters", label: t("liters") },
              ]}
              empty={t("empty")}
              locale={locale}
            />
          )}
        </section>
      ) : (
        <section className="grid gap-5 xl:grid-cols-2">
          <RowsCard
            title={t("accountsByRoleStatus")}
            rows={asRows(tables.accountsByRoleStatus)}
            fields={[
              { key: "role", label: t("role"), status: true },
              { key: "status", label: t("status"), status: true },
              { key: "count", label: t("count") },
            ]}
            empty={t("empty")}
            locale={locale}
          />
          <RowsCard
            title={t("activeListingsByRegion")}
            rows={asRows(tables.activeListingsByRegion)}
            fields={[
              { key: "regionName", label: t("region") },
              { key: "listings", label: t("activeListings") },
              { key: "availableLiters", label: t("liters") },
            ]}
            empty={t("empty")}
            locale={locale}
          />
          <RowsCard
            title={t("ordersByStatus")}
            rows={asRows(tables.ordersByStatus)}
            fields={[
              { key: "status", label: t("status"), status: true },
              { key: "count", label: t("count") },
            ]}
            empty={t("empty")}
            locale={locale}
          />
          <RowsCard
            title={t("recentAuditEvents")}
            rows={asRows(tables.recentAuditEvents)}
            fields={[
              { key: "action", label: t("recentAuditEvents") },
              { key: "targetType", label: t("role") },
              { key: "createdAt", label: t("period"), date: true },
            ]}
            empty={t("empty")}
            locale={locale}
          />
        </section>
      )}
      {role === "farmer" && asRows(charts.qualityTrend).length ? (
        <RowsCard
          title={t("qualityTrend")}
          rows={asRows(charts.qualityTrend)}
          fields={[
            { key: "metric", label: t("qualityTrend") },
            { key: "average", label: t("count") },
            { key: "period", label: t("period"), date: true },
          ]}
          empty={t("empty")}
          locale={locale}
        />
      ) : null}
      {role === "admin" ? (
        <Link
          href="/admin/applications"
          className="bg-sage inline-flex min-h-11 items-center rounded-full px-5 font-semibold text-white"
        >
          {t("accountsAwaitingApproval")}
        </Link>
      ) : null}
    </div>
  );
}

function OldestApplicant({
  value,
  locale,
  t,
}: Readonly<{
  value: unknown;
  locale: string;
  t: (key: string) => string;
}>) {
  const applicant = asRecord(value);
  const user = asRecord(applicant?.user);
  const fullName = asText(user?.fullName);
  const email = asText(user?.email);
  const createdAt = asText(applicant?.createdAt);
  if (!fullName && !email) {
    return <p className="text-muted mt-2 text-xs">{t("empty")}</p>;
  }
  return (
    <div className="border-border text-muted mt-3 border-t pt-3 text-xs">
      <p className="text-ink font-semibold">{t("oldestApplicant")}</p>
      <p className="mt-1">{fullName}</p>
      <p>{email}</p>
      {createdAt ? <p>{formatDate(createdAt, locale)}</p> : null}
    </div>
  );
}

function ChartCard({
  title,
  points,
  empty,
}: Readonly<{
  title: string;
  points: readonly { period: string; value: number }[];
  empty: string;
}>) {
  return (
    <section className="rounded-card border-border bg-surface border p-5">
      <h2 className="text-ink font-semibold">{title}</h2>
      {points.length ? (
        <div className="mt-4 h-64" role="img" aria-label={title}>
          <ResponsiveContainer width="100%" height="100%">
            <LineChart
              data={points}
              margin={{ top: 8, right: 12, bottom: 4, left: 0 }}
            >
              <CartesianGrid stroke="#d9d5c9" strokeDasharray="3 3" />
              <XAxis dataKey="period" tick={{ fontSize: 11 }} />
              <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
              <Tooltip />
              <Line
                type="monotone"
                dataKey="value"
                stroke="#2f5d46"
                strokeWidth={2}
                dot={{ r: 3 }}
                isAnimationActive={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      ) : (
        <p className="bg-surface-soft text-muted mt-4 rounded-lg p-4 text-sm">
          {empty}
        </p>
      )}
    </section>
  );
}

type TableField = Readonly<{
  key: string;
  label: string;
  date?: boolean;
  status?: boolean;
  linkPrefix?: string;
}>;

function RowsCard({
  title,
  rows,
  fields,
  empty,
  locale,
}: Readonly<{
  title: string;
  rows: readonly unknown[];
  fields: readonly TableField[];
  empty: string;
  locale: string;
}>) {
  const tStatus = useTranslations("status");
  if (!rows.length) {
    return (
      <section className="rounded-card border-border bg-surface border p-5">
        <h2 className="text-ink font-semibold">{title}</h2>
        <p className="bg-surface-soft text-muted mt-4 rounded-lg p-4 text-sm">
          {empty}
        </p>
      </section>
    );
  }

  return (
    <section className="rounded-card border-border bg-surface overflow-hidden border">
      <h2 className="text-ink px-5 py-4 font-semibold">{title}</h2>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[28rem] border-collapse text-left text-sm">
          <thead className="bg-surface-soft text-muted text-xs tracking-wide uppercase">
            <tr>
              {fields.map((field) => (
                <th key={field.key} className="px-4 py-3 font-semibold">
                  {field.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => {
              const record = asRecord(row);
              return (
                <tr
                  key={asText(record?.id) ?? `${title}-${index}`}
                  className="border-border border-t"
                >
                  {fields.map((field) => {
                    const raw = record?.[field.key];
                    let value =
                      typeof raw === "number"
                        ? formatNumber(raw, locale)
                        : (asText(raw) ?? "—");
                    if (field.date && typeof raw === "string")
                      value = formatDate(raw, locale);
                    if (field.status && typeof raw === "string") {
                      const statusKey = raw.startsWith("order.")
                        ? raw.slice(6)
                        : raw;
                      value =
                        statusKey in statusKeys
                          ? tStatus(statusKey as keyof typeof statusKeys)
                          : raw;
                    }
                    return (
                      <td key={field.key} className="text-ink px-4 py-3">
                        {field.linkPrefix && typeof raw === "string" ? (
                          <Link
                            href={`${field.linkPrefix}${raw}`}
                            className="text-sage font-semibold underline underline-offset-2"
                          >
                            {raw.slice(0, 8)}
                          </Link>
                        ) : (
                          value
                        )}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

const statusKeys = {
  pending: true,
  approved: true,
  rejected: true,
  suspended: true,
  placed: true,
  accepted: true,
  cancelled: true,
  ready_for_pickup: true,
  picked_up: true,
  delivered: true,
  completed: true,
  disputed: true,
  active: true,
  paused: true,
  sold_out: true,
  expired: true,
  archived: true,
};
