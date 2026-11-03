"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link } from "@/i18n/navigation";
import { ApiError, ensureApiSuccess, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type OrderStatus = components["schemas"]["OrderStatus"];
type Role = "farmer" | "processor";

const orderStatuses: readonly OrderStatus[] = [
  "placed",
  "accepted",
  "rejected",
  "cancelled",
  "ready_for_pickup",
  "picked_up",
  "delivered",
  "completed",
  "disputed",
];

function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(
    value,
  );
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

export function OrdersClient({ role }: Readonly<{ role: Role }>) {
  const t = useTranslations("marketplace");
  const tStatus = useTranslations("status");
  const tDashboard = useTranslations("dashboard");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<OrderStatus | "">("");
  const ordersQuery = useQuery({
    queryKey: ["orders", page, status],
    queryFn: async () => {
      const query = {
        page,
        pageSize: 20,
        ...(status ? { status } : {}),
      };
      return getApiData(await api.GET("/orders", { params: { query } }));
    },
  });

  if (ordersQuery.isError) {
    const error =
      ordersQuery.error instanceof ApiError
        ? ordersQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void ordersQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!ordersQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const data = ordersQuery.data;
  const hasNext = data.page * data.pageSize < data.total;
  const detailPrefix =
    role === "farmer" ? "/farmer/orders/" : "/processor/orders/";

  return (
    <div className="space-y-5">
      <label className="text-ink flex max-w-sm items-center gap-3 text-sm font-semibold">
        {tDashboard("status")}
        <select
          value={status}
          onChange={(event) => {
            setStatus(event.target.value as OrderStatus | "");
            setPage(1);
          }}
          className="border-border min-h-11 flex-1 rounded-lg border bg-white px-3 font-normal"
        >
          <option value="">{t("myOrders")}</option>
          {orderStatuses.map((value) => (
            <option key={value} value={value}>
              {tStatus(value)}
            </option>
          ))}
        </select>
      </label>
      {data.items.length ? (
        <div className="rounded-card border-border bg-surface overflow-hidden border">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[42rem] border-collapse text-left text-sm">
              <thead className="bg-surface-soft text-muted text-xs tracking-wide uppercase">
                <tr>
                  <th className="px-4 py-3">{t("orderDetails")}</th>
                  <th className="px-4 py-3">{t("orderLiters")}</th>
                  <th className="px-4 py-3">{t("pricePerLiter")}</th>
                  <th className="px-4 py-3">{t("pickupTime")}</th>
                  <th className="px-4 py-3">{t("orderNote")}</th>
                </tr>
              </thead>
              <tbody>
                {data.items.map((order) => (
                  <tr key={order.id} className="border-border border-t">
                    <td className="px-4 py-3">
                      <Link
                        href={`${detailPrefix}${order.id}`}
                        className="text-sage font-semibold underline underline-offset-2"
                      >
                        {order.id.slice(0, 8)}
                      </Link>
                      <span className="bg-sage-soft ml-2 rounded-full px-2 py-1 text-xs">
                        {tStatus(order.status)}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {formatNumber(order.liters, locale)}
                    </td>
                    <td className="px-4 py-3">
                      {formatNumber(order.pricePerLiterUzs, locale)} UZS
                    </td>
                    <td className="px-4 py-3">
                      {formatDate(order.expectedPickupAt, locale)}
                    </td>
                    <td className="px-4 py-3">{order.note || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <p className="rounded-card border-border bg-surface text-muted border p-6 text-sm">
          {t("noOrders")}
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

const disputeSchema = z.object({ reason: z.string().trim().min(1).max(2000) });
type DisputeValues = z.infer<typeof disputeSchema>;

export function OrderDetailClient({
  orderId,
  role,
}: Readonly<{ orderId: string; role: Role }>) {
  const t = useTranslations("marketplace");
  const tStatus = useTranslations("status");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const queryClient = useQueryClient();
  const [actionError, setActionError] = useState<ApiError | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const orderQuery = useQuery({
    queryKey: ["order", orderId],
    queryFn: async () =>
      getApiData(
        await api.GET("/orders/{orderId}", { params: { path: { orderId } } }),
      ),
  });
  const disputeForm = useForm<DisputeValues>({
    resolver: zodResolver(disputeSchema),
    defaultValues: { reason: "" },
  });

  const refreshOrder = async () => {
    await queryClient.invalidateQueries({ queryKey: ["order", orderId] });
  };
  const runAction = async (action: () => Promise<void>) => {
    setActionError(null);
    setNotice(null);
    try {
      await action();
      await refreshOrder();
    } catch (error) {
      setActionError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  };

  if (orderQuery.isError) {
    const error =
      orderQuery.error instanceof ApiError
        ? orderQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void orderQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!orderQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }
  const order = orderQuery.data;
  const isFarmer = role === "farmer";
  const canDecide = isFarmer && order.status === "placed";
  const canCancel = !isFarmer && order.status === "placed";
  const nextStatus =
    isFarmer && order.status === "accepted"
      ? "ready_for_pickup"
      : isFarmer && order.status === "ready_for_pickup"
        ? "picked_up"
        : isFarmer && order.status === "picked_up"
          ? "delivered"
          : !isFarmer && order.status === "delivered"
            ? "completed"
            : null;
  const canDispute = ![
    "completed",
    "rejected",
    "cancelled",
    "disputed",
  ].includes(order.status);
  const canReportQuality = [
    "accepted",
    "ready_for_pickup",
    "picked_up",
    "delivered",
  ].includes(order.status);

  const decide = async (kind: "accept" | "reject") => {
    if (kind === "reject" && !rejectReason.trim()) return;
    await runAction(async () => {
      if (kind === "accept") {
        ensureApiSuccess(
          await api.POST("/orders/{orderId}/accept", {
            params: { path: { orderId } },
          }),
        );
      } else {
        ensureApiSuccess(
          await api.POST("/orders/{orderId}/reject", {
            params: { path: { orderId } },
            body: { reason: rejectReason.trim() },
          }),
        );
        setRejectReason("");
      }
    });
  };

  const cancel = async () => {
    if (!window.confirm(t("cancelOrder"))) return;
    await runAction(async () => {
      ensureApiSuccess(
        await api.POST("/orders/{orderId}/cancel", {
          params: { path: { orderId } },
        }),
      );
    });
  };

  const advance = async () => {
    if (!nextStatus) return;
    await runAction(async () => {
      ensureApiSuccess(
        await api.POST("/orders/{orderId}/status", {
          params: { path: { orderId } },
          body: { status: nextStatus },
        }),
      );
    });
  };

  const submitDispute = disputeForm.handleSubmit(async ({ reason }) => {
    await runAction(async () => {
      ensureApiSuccess(
        await api.POST("/orders/{orderId}/disputes", {
          params: { path: { orderId } },
          body: { reason },
        }),
      );
      disputeForm.reset();
    });
  });

  return (
    <div className="space-y-6">
      {actionError ? <ApiErrorBanner error={actionError} /> : null}
      {notice ? (
        <p role="status" className="rounded-card bg-sage-soft p-4 text-sm">
          {notice}
        </p>
      ) : null}
      <section className="rounded-card border-border bg-surface border p-5 sm:p-7">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-muted text-sm">{t("orderDetails")}</p>
            <h2 className="text-ink mt-1 font-mono text-lg font-semibold break-all">
              {order.id}
            </h2>
          </div>
          <span className="bg-sage-soft text-sage rounded-full px-3 py-1 text-sm font-semibold">
            {tStatus(order.status)}
          </span>
        </div>
        <dl className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          <OrderDatum
            label={t("orderLiters")}
            value={`${formatNumber(order.liters, locale)} L`}
          />
          <OrderDatum
            label={t("pricePerLiter")}
            value={`${formatNumber(order.pricePerLiterUzs, locale)} UZS`}
          />
          <OrderDatum
            label={t("pickupTime")}
            value={formatDate(order.expectedPickupAt, locale)}
          />
          <OrderDatum
            label={t("qualityOutcome")}
            value={
              order.qualityOutcome
                ? t(
                    `quality${order.qualityOutcome === "accepted" ? "Accepted" : "Rejected"}`,
                  )
                : "—"
            }
          />
        </dl>
        <p className="text-muted mt-5 text-xs">
          {t("order")}: {order.listingId}
        </p>
      </section>

      {canDecide ? (
        <section className="rounded-card border-border bg-surface flex flex-col gap-3 border p-5 sm:flex-row">
          <button
            type="button"
            onClick={() => void decide("accept")}
            className="bg-sage min-h-11 flex-1 rounded-full px-5 font-semibold text-white"
          >
            {t("acceptOrder")}
          </button>
          <div className="flex flex-1 flex-col gap-2 sm:flex-row">
            <input
              value={rejectReason}
              onChange={(event) => setRejectReason(event.target.value)}
              maxLength={2000}
              aria-label={t("decisionReason")}
              placeholder={t("decisionReason")}
              className="border-border min-h-11 flex-1 rounded-lg border bg-white px-3"
            />
            <button
              type="button"
              disabled={!rejectReason.trim()}
              onClick={() => void decide("reject")}
              className="border-danger/40 text-danger min-h-11 rounded-full border px-5 font-semibold disabled:opacity-50"
            >
              {t("rejectOrder")}
            </button>
          </div>
        </section>
      ) : null}
      {canCancel ? (
        <button
          type="button"
          onClick={() => void cancel()}
          className="border-danger/40 text-danger min-h-11 rounded-full border px-5 font-semibold"
        >
          {t("cancelOrder")}
        </button>
      ) : null}
      {nextStatus ? (
        <button
          type="button"
          onClick={() => void advance()}
          className="bg-sage min-h-11 rounded-full px-5 font-semibold text-white"
        >
          {t("nextMilestone")}: {tStatus(nextStatus)}
        </button>
      ) : null}

      {order.history?.length ? (
        <section className="rounded-card border-border bg-surface border p-5">
          <h2 className="text-ink font-semibold">{t("history")}</h2>
          <ol className="border-border mt-4 space-y-4 border-l pl-4">
            {order.history.map((item) => (
              <li key={`${item.createdAt}-${item.status}`} className="relative">
                <span
                  aria-hidden="true"
                  className="bg-sage absolute top-1 -left-[1.32rem] size-2 rounded-full"
                />
                <p className="text-ink font-semibold">{tStatus(item.status)}</p>
                <p className="text-muted mt-1 text-xs">
                  {formatDate(item.createdAt, locale)}
                </p>
                {item.note ? (
                  <p className="text-muted mt-1 text-sm">{item.note}</p>
                ) : null}
              </li>
            ))}
          </ol>
        </section>
      ) : null}

      {canDispute ? (
        <form
          className="rounded-card border-border bg-surface space-y-3 border p-5"
          onSubmit={submitDispute}
          noValidate
        >
          <h2 className="text-ink font-semibold">{t("disputeOrder")}</h2>
          <label
            htmlFor="disputeReason"
            className="text-muted block text-sm font-medium"
          >
            {t("disputeReason")}
          </label>
          <textarea
            id="disputeReason"
            rows={3}
            maxLength={2000}
            className="border-border w-full rounded-lg border bg-white p-3"
            {...disputeForm.register("reason")}
          />
          {disputeForm.formState.errors.reason ? (
            <p className="text-danger text-sm">{tCommon("required")}</p>
          ) : null}
          <button
            type="submit"
            className="border-border min-h-11 rounded-full border px-5 font-semibold"
          >
            {t("submitDispute")}
          </button>
        </form>
      ) : null}
      {canReportQuality ? (
        <QualityReportForm
          orderId={orderId}
          onSaved={() => {
            setNotice(t("qualitySaved"));
            void refreshOrder();
          }}
          onError={setActionError}
        />
      ) : null}
      <button
        type="button"
        onClick={() => void orderQuery.refetch()}
        className="border-border min-h-10 rounded-full border px-4 text-sm font-semibold"
      >
        {tCommon("retry")}
      </button>
    </div>
  );
}

function OrderDatum({
  label,
  value,
}: Readonly<{ label: string; value: string }>) {
  return (
    <div>
      <dt className="text-muted text-xs font-semibold">{label}</dt>
      <dd className="text-ink mt-1 text-sm font-semibold break-words">
        {value}
      </dd>
    </div>
  );
}

const measurementSchema = z.object({
  metrics: z
    .array(
      z.object({
        name: z.string().trim().min(1).max(32),
        value: z.number().min(-1000).max(1000),
      }),
    )
    .min(1)
    .max(16)
    .refine(
      (metrics) =>
        new Set(metrics.map((metric) => metric.name.trim())).size ===
        metrics.length,
    ),
  outcome: z.enum(["accepted", "rejected"]),
  note: z.string().max(2000),
});
type MeasurementValues = z.infer<typeof measurementSchema>;

function QualityReportForm({
  orderId,
  onSaved,
  onError,
}: Readonly<{
  orderId: string;
  onSaved: () => void;
  onError: (error: ApiError | null) => void;
}>) {
  const t = useTranslations("marketplace");
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const {
    control,
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    reset,
  } = useForm<MeasurementValues>({
    resolver: zodResolver(measurementSchema),
    defaultValues: {
      metrics: [{ name: "", value: Number.NaN }],
      outcome: "accepted",
      note: "",
    },
  });
  const { fields, append, remove } = useFieldArray({
    control,
    name: "metrics",
  });

  const submit = handleSubmit(async ({ metrics, outcome, note }) => {
    onError(null);
    const measurements = Object.fromEntries(
      metrics.map(({ name, value }) => [name.trim(), value]),
    );
    try {
      ensureApiSuccess(
        await api.POST("/orders/{orderId}/quality-reports", {
          params: { path: { orderId } },
          body: {
            measurements,
            outcome,
            ...(note.trim() ? { note: note.trim() } : {}),
          },
        }),
      );
      reset();
      onSaved();
    } catch (error) {
      onError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  return (
    <form
      className="rounded-card border-border bg-surface space-y-4 border p-5"
      onSubmit={submit}
      noValidate
    >
      <h2 className="text-ink font-semibold">{t("qualityReport")}</h2>
      {fields.map((field, index) => (
        <div
          key={field.id}
          className="grid gap-3 sm:grid-cols-[1fr_10rem_auto]"
        >
          <label className="text-muted text-sm font-medium">
            {t("measurementName")}
            <input
              maxLength={32}
              className="border-border text-ink mt-1 min-h-11 w-full rounded-lg border bg-white px-3"
              {...register(`metrics.${index}.name`)}
            />
          </label>
          <label className="text-muted text-sm font-medium">
            {t("measurementValue")}
            <input
              type="number"
              min={-1000}
              max={1000}
              step="any"
              className="border-border text-ink mt-1 min-h-11 w-full rounded-lg border bg-white px-3"
              {...register(`metrics.${index}.value`, { valueAsNumber: true })}
            />
          </label>
          <button
            type="button"
            disabled={fields.length <= 1}
            onClick={() => remove(index)}
            className="border-border min-h-11 self-end rounded-full border px-4 text-sm font-semibold disabled:opacity-40"
          >
            {t("removeMeasurement")}
          </button>
        </div>
      ))}
      {errors.metrics ? (
        <p className="text-danger text-sm">{tCommon("required")}</p>
      ) : null}
      <button
        type="button"
        disabled={fields.length >= 16}
        onClick={() => append({ name: "", value: Number.NaN })}
        className="border-border min-h-10 rounded-full border px-4 text-sm font-semibold disabled:opacity-40"
      >
        {t("addMeasurement")}
      </button>
      <label className="text-muted block text-sm font-medium">
        {t("qualityOutcome")}
        <select
          className="border-border text-ink mt-1 min-h-11 w-full rounded-lg border bg-white px-3"
          {...register("outcome")}
        >
          <option value="accepted">{t("qualityAccepted")}</option>
          <option value="rejected">{t("qualityRejected")}</option>
        </select>
      </label>
      <label className="text-muted block text-sm font-medium">
        {t("orderNote")}
        <textarea
          rows={2}
          maxLength={2000}
          className="border-border text-ink mt-1 w-full rounded-lg border bg-white p-3"
          {...register("note")}
        />
      </label>
      <button
        type="submit"
        disabled={isSubmitting}
        className="bg-sage min-h-11 rounded-full px-5 font-semibold text-white disabled:opacity-60"
      >
        {isSubmitting ? tCommon("loading") : t("qualityReport")}
      </button>
    </form>
  );
}
