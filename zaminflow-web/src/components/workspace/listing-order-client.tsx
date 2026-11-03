"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useForm } from "react-hook-form";
import { useState } from "react";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { useRouter } from "@/i18n/navigation";
import { ApiError, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type Listing = components["schemas"]["Listing"];

function formatNumber(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(
    value,
  );
}

function formatLocalInputAsIso(value: string): string {
  return new Date(value).toISOString();
}

export function ListingOrderClient({
  listingId,
}: Readonly<{ listingId: string }>) {
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const listingQuery = useQuery({
    queryKey: ["listing", listingId],
    queryFn: async () =>
      getApiData(
        await api.GET("/listings/{listingId}", {
          params: { path: { listingId } },
        }),
      ),
  });

  if (listingQuery.isError) {
    const error =
      listingQuery.error instanceof ApiError
        ? listingQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void listingQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!listingQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  return <ListingOrderForm listing={listingQuery.data} />;
}

const orderSchema = z.object({
  liters: z.number().int().positive(),
  expectedPickupAt: z.string().min(1),
  note: z.string().max(2000),
});
type OrderValues = z.infer<typeof orderSchema>;

function ListingOrderForm({ listing }: Readonly<{ listing: Listing }>) {
  const t = useTranslations("marketplace");
  const tCommon = useTranslations("common");
  const tStatus = useTranslations("status");
  const locale = useLocale();
  const { api } = useSession();
  const router = useRouter();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<OrderValues>({
    resolver: zodResolver(orderSchema),
    defaultValues: { liters: Number.NaN, expectedPickupAt: "", note: "" },
  });

  const submit = handleSubmit(async ({ liters, expectedPickupAt, note }) => {
    setFormError(null);
    if (liters > listing.availableLiters) {
      setError("liters", { type: "range" });
      return;
    }
    try {
      const result = getApiData(
        await api.POST("/orders", {
          body: {
            listingId: listing.id,
            liters,
            expectedPickupAt: formatLocalInputAsIso(expectedPickupAt),
            ...(note.trim() ? { note: note.trim() } : {}),
          },
        }),
      );
      router.replace(`/processor/orders/${result.order.id}`);
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  const availableUntil = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeZone: "Asia/Tashkent",
  }).format(new Date(listing.availableUntil));

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_0.85fr]">
      <section className="rounded-card border-border bg-surface border p-5 sm:p-7">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sage text-sm font-semibold">
              {listing.regionCode}
            </p>
            <h2 className="font-display text-ink mt-2 text-2xl font-bold">
              {t("marketTitle")}
            </h2>
          </div>
          <span className="bg-sage-soft text-sage rounded-full px-3 py-1 text-xs font-semibold">
            {tStatus(listing.status)}
          </span>
        </div>
        <dl className="mt-6 grid gap-5 sm:grid-cols-2">
          <OrderListingDatum
            label={t("availableLiters")}
            value={formatNumber(listing.availableLiters, locale)}
          />
          <OrderListingDatum
            label={t("fatPercent")}
            value={`${formatNumber(listing.fatPercent, locale)}%`}
          />
          <OrderListingDatum
            label={t("pricePerLiter")}
            value={`${formatNumber(listing.pricePerLiterUzs, locale)} UZS`}
          />
          <OrderListingDatum
            label={t("availableUntil")}
            value={availableUntil}
          />
        </dl>
      </section>

      <form
        className="rounded-card border-border bg-surface space-y-5 border p-5 sm:p-7"
        onSubmit={submit}
        noValidate
      >
        <h2 className="text-ink font-semibold">{t("placeOrder")}</h2>
        {formError ? <ApiErrorBanner error={formError} /> : null}
        <label
          className="text-ink block text-sm font-semibold"
          htmlFor="liters"
        >
          {t("orderLiters")}
          <input
            id="liters"
            type="number"
            min="1"
            max={listing.availableLiters}
            step="1"
            className="border-border mt-2 min-h-12 w-full rounded-lg border bg-white px-4 font-normal"
            {...register("liters", { valueAsNumber: true })}
          />
        </label>
        {errors.liters ? (
          <p className="text-danger text-sm">
            {errors.liters.type === "range"
              ? t("quantityTooHigh")
              : tCommon("required")}
          </p>
        ) : null}
        <label
          className="text-ink block text-sm font-semibold"
          htmlFor="expectedPickupAt"
        >
          {t("pickupTime")}
          <input
            id="expectedPickupAt"
            type="datetime-local"
            className="border-border mt-2 min-h-12 w-full rounded-lg border bg-white px-4 font-normal"
            {...register("expectedPickupAt")}
          />
        </label>
        {errors.expectedPickupAt ? (
          <p className="text-danger text-sm">{tCommon("required")}</p>
        ) : null}
        <label className="text-ink block text-sm font-semibold" htmlFor="note">
          {t("orderNote")}
          <textarea
            id="note"
            rows={3}
            maxLength={2000}
            className="border-border mt-2 w-full rounded-lg border bg-white p-3 font-normal"
            {...register("note")}
          />
        </label>
        <p className="text-muted text-xs leading-5">{t("reservationNotice")}</p>
        <button
          type="submit"
          disabled={isSubmitting || listing.status !== "active"}
          className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white disabled:opacity-50"
        >
          {isSubmitting ? tCommon("loading") : t("submitOrder")}
        </button>
      </form>
    </div>
  );
}

function OrderListingDatum({
  label,
  value,
}: Readonly<{ label: string; value: string }>) {
  return (
    <div>
      <dt className="text-muted text-xs font-semibold">{label}</dt>
      <dd className="text-ink mt-1 text-sm font-semibold">{value}</dd>
    </div>
  );
}
