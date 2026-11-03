"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { ApiErrorBanner } from "@/components/shared/api-error-banner";
import { Link, useRouter } from "@/i18n/navigation";
import { ApiError, ensureApiSuccess, getApiData } from "@/lib/api/errors";
import { useSession } from "@/lib/auth/session-context";
import type { components } from "@/lib/api/schema";

type Role = "farmer" | "processor";
type Listing = components["schemas"]["Listing"];
type SortOrder = "newest" | "oldest" | "price_asc" | "price_desc";

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
        timeZone: "Asia/Tashkent",
      }).format(date)
    : value;
}

export function ListingsClient({ role }: Readonly<{ role: Role }>) {
  const t = useTranslations("marketplace");
  const tStatus = useTranslations("status");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const [page, setPage] = useState(1);
  const [regionCode, setRegionCode] = useState("");
  const [minimum, setMinimum] = useState("");
  const [sort, setSort] = useState<SortOrder>("newest");
  const [activeFilters, setActiveFilters] = useState({
    regionCode: "",
    minimum: "",
    sort: "newest" as SortOrder,
  });
  const regionsQuery = useQuery({
    queryKey: ["regions"],
    enabled: role === "processor",
    queryFn: async () => getApiData(await api.GET("/meta/regions")).items,
  });
  const listingsQuery = useQuery({
    queryKey: ["listings", role, page, activeFilters],
    queryFn: async () => {
      const minimumValue = Number(activeFilters.minimum);
      const query = {
        page,
        pageSize: 20,
        sort: activeFilters.sort,
        ...(role === "processor" && activeFilters.regionCode
          ? { regionCode: activeFilters.regionCode }
          : {}),
        ...(role === "processor" &&
        Number.isInteger(minimumValue) &&
        minimumValue > 0
          ? { minLiters: minimumValue }
          : {}),
      };
      return getApiData(await api.GET("/listings", { params: { query } }));
    },
  });

  if (listingsQuery.isError || (role === "processor" && regionsQuery.isError)) {
    const error =
      listingsQuery.error instanceof ApiError
        ? listingsQuery.error
        : regionsQuery.error instanceof ApiError
          ? regionsQuery.error
          : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => {
            void listingsQuery.refetch();
            if (role === "processor") void regionsQuery.refetch();
          }}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }

  if (!listingsQuery.data || (role === "processor" && !regionsQuery.data)) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const data = listingsQuery.data;
  const hasNext = data.page * data.pageSize < data.total;
  const hasPrevious = data.page > 1;
  const listingPath = role === "farmer" ? "/farmer/listings/" : "/marketplace/";

  return (
    <div className="space-y-6">
      {role === "processor" ? (
        <form
          className="rounded-card border-border bg-surface grid gap-3 border p-4 sm:grid-cols-4 sm:items-end"
          onSubmit={(event) => {
            event.preventDefault();
            setPage(1);
            setActiveFilters({ regionCode, minimum, sort });
          }}
        >
          <label className="text-ink text-sm font-semibold">
            {t("region")}
            <select
              value={regionCode}
              onChange={(event) => setRegionCode(event.target.value)}
              className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
            >
              <option value="">{t("region")}</option>
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
          </label>
          <label className="text-ink text-sm font-semibold">
            {t("minimumLiters")}
            <input
              type="number"
              min="1"
              step="1"
              value={minimum}
              onChange={(event) => setMinimum(event.target.value)}
              className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
            />
          </label>
          <label className="text-ink text-sm font-semibold">
            {t("sort")}
            <select
              value={sort}
              onChange={(event) => setSort(event.target.value as SortOrder)}
              className="border-border mt-2 min-h-11 w-full rounded-lg border bg-white px-3 font-normal"
            >
              <option value="newest">{t("newest")}</option>
              <option value="oldest">{t("oldest")}</option>
              <option value="price_asc">{t("priceAscending")}</option>
              <option value="price_desc">{t("priceDescending")}</option>
            </select>
          </label>
          <button
            type="submit"
            className="bg-sage min-h-11 rounded-full px-5 font-semibold text-white"
          >
            {t("applyFilters")}
          </button>
        </form>
      ) : (
        <div className="flex justify-end">
          <Link
            href="/farmer/listings/new"
            className="bg-sage inline-flex min-h-11 items-center rounded-full px-5 font-semibold text-white"
          >
            {t("createListing")}
          </Link>
        </div>
      )}

      {data.items.length ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {data.items.map((listing) => (
            <article
              key={listing.id}
              className="rounded-card border-border bg-surface border p-5"
            >
              <div className="flex items-start justify-between gap-3">
                <p className="text-sage text-sm font-semibold">
                  {listing.regionCode}
                </p>
                <span className="bg-sage-soft text-sage rounded-full px-3 py-1 text-xs font-semibold">
                  {tStatus(listing.status)}
                </span>
              </div>
              <dl className="mt-5 grid grid-cols-2 gap-4 text-sm">
                <div>
                  <dt className="text-muted">{t("availableLiters")}</dt>
                  <dd className="text-ink mt-1 font-semibold">
                    {formatNumber(listing.availableLiters, locale)}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted">{t("fatPercent")}</dt>
                  <dd className="text-ink mt-1 font-semibold">
                    {formatNumber(listing.fatPercent, locale)}%
                  </dd>
                </div>
                <div>
                  <dt className="text-muted">{t("pricePerLiter")}</dt>
                  <dd className="text-ink mt-1 font-semibold">
                    {formatNumber(listing.pricePerLiterUzs, locale)} UZS
                  </dd>
                </div>
                <div>
                  <dt className="text-muted">{t("availableUntil")}</dt>
                  <dd className="text-ink mt-1 font-semibold">
                    {formatDate(listing.availableUntil, locale)}
                  </dd>
                </div>
              </dl>
              <Link
                href={`${listingPath}${listing.id}`}
                className="text-sage mt-5 inline-flex min-h-10 items-center font-semibold underline underline-offset-4"
              >
                {role === "farmer" ? t("editListing") : t("placeOrder")}
              </Link>
            </article>
          ))}
        </div>
      ) : (
        <p className="rounded-card border-border bg-surface text-muted border p-6 text-sm">
          {role === "farmer" ? t("noListings") : t("noOffers")}
        </p>
      )}
      <div className="flex justify-end gap-3">
        <button
          type="button"
          disabled={!hasPrevious}
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

const listingSchema = z
  .object({
    regionCode: z.string().min(1).max(32),
    availableLiters: z.number().int().positive(),
    fatPercent: z.number().min(0).max(100),
    pricePerLiterUzs: z.number().int().positive(),
    availableFrom: z.string().min(1),
    availableUntil: z.string().min(1),
  })
  .refine(
    (value) =>
      new Date(value.availableUntil).getTime() >
      new Date(value.availableFrom).getTime(),
    { path: ["availableUntil"] },
  );
type ListingValues = z.infer<typeof listingSchema>;

function toLocalInput(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "";
  const localDate = new Date(
    date.getTime() - date.getTimezoneOffset() * 60_000,
  );
  return localDate.toISOString().slice(0, 16);
}

function toIso(value: string): string {
  return new Date(value).toISOString();
}

export function ListingEditor({ listingId }: Readonly<{ listingId?: string }>) {
  const tCommon = useTranslations("common");
  const { api } = useSession();
  const listingQuery = useQuery({
    queryKey: ["listing", listingId],
    enabled: Boolean(listingId),
    queryFn: async () =>
      getApiData(
        await api.GET("/listings/{listingId}", {
          params: { path: { listingId: listingId ?? "" } },
        }),
      ),
  });

  if (listingId && listingQuery.isError) {
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
  if (listingId && !listingQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  return (
    <ListingEditorForm
      key={listingId ?? "new"}
      {...(listingId ? { listingId } : {})}
      listing={listingQuery.data ?? null}
    />
  );
}

function ListingEditorForm({
  listingId,
  listing,
}: Readonly<{ listingId?: string; listing: Listing | null }>) {
  const t = useTranslations("marketplace");
  const tCommon = useTranslations("common");
  const locale = useLocale();
  const { api } = useSession();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const regionsQuery = useQuery({
    queryKey: ["regions"],
    queryFn: async () => getApiData(await api.GET("/meta/regions")).items,
  });
  const form = useForm<ListingValues>({
    resolver: zodResolver(listingSchema),
    defaultValues: listing
      ? {
          regionCode: listing.regionCode,
          availableLiters: listing.availableLiters,
          fatPercent: listing.fatPercent,
          pricePerLiterUzs: listing.pricePerLiterUzs,
          availableFrom: toLocalInput(listing.availableFrom),
          availableUntil: toLocalInput(listing.availableUntil),
        }
      : {
          regionCode: "",
          availableLiters: 0,
          fatPercent: 0,
          pricePerLiterUzs: 0,
          availableFrom: "",
          availableUntil: "",
        },
  });

  if (regionsQuery.isError) {
    const error =
      regionsQuery.error instanceof ApiError
        ? regionsQuery.error
        : new ApiError("internal_error");
    return (
      <div className="space-y-4">
        <ApiErrorBanner error={error} />
        <button
          type="button"
          onClick={() => void regionsQuery.refetch()}
          className="border-border min-h-11 rounded-full border px-5 font-semibold"
        >
          {tCommon("retry")}
        </button>
      </div>
    );
  }
  if (!regionsQuery.data) {
    return (
      <p
        role="status"
        className="rounded-card bg-surface-soft text-muted p-4 text-sm"
      >
        {tCommon("loading")}
      </p>
    );
  }

  const submit = form.handleSubmit(async (values) => {
    setFormError(null);
    setNotice(null);
    const body = {
      ...values,
      availableFrom: toIso(values.availableFrom),
      availableUntil: toIso(values.availableUntil),
    };
    try {
      if (listingId) {
        ensureApiSuccess(
          await api.PATCH("/listings/{listingId}", {
            params: { path: { listingId } },
            body: {
              availableLiters: body.availableLiters,
              fatPercent: body.fatPercent,
              pricePerLiterUzs: body.pricePerLiterUzs,
              availableFrom: body.availableFrom,
              availableUntil: body.availableUntil,
            },
          }),
        );
        setNotice(t("offerSaved"));
        await queryClient.invalidateQueries({
          queryKey: ["listing", listingId],
        });
      } else {
        getApiData(await api.POST("/listings", { body }));
        setNotice(t("offerSaved"));
      }
      await queryClient.invalidateQueries({ queryKey: ["listings"] });
      if (!listingId) router.replace("/farmer/listings");
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  });

  const setStatus = async (status: "active" | "paused") => {
    if (!listingId) return;
    setFormError(null);
    try {
      ensureApiSuccess(
        await api.PATCH("/listings/{listingId}", {
          params: { path: { listingId } },
          body: { status },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: ["listing", listingId] });
      await queryClient.invalidateQueries({ queryKey: ["listings"] });
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  };

  const archive = async () => {
    if (!listingId || !window.confirm(t("archiveConfirm"))) return;
    setFormError(null);
    try {
      ensureApiSuccess(
        await api.DELETE("/listings/{listingId}", {
          params: { path: { listingId } },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: ["listings"] });
      router.replace("/farmer/listings");
    } catch (error) {
      setFormError(
        error instanceof ApiError ? error : new ApiError("internal_error"),
      );
    }
  };

  const labelClass = "mb-2 block text-sm font-semibold text-ink";
  const inputClass =
    "min-h-12 w-full rounded-lg border border-border bg-white px-4";
  const numberField = (
    key: "availableLiters" | "fatPercent" | "pricePerLiterUzs",
    title: string,
    min: number,
    max?: number,
    step = "1",
  ) => (
    <div>
      <label htmlFor={key} className={labelClass}>
        {title}
      </label>
      <input
        id={key}
        type="number"
        min={min}
        max={max}
        step={step}
        className={inputClass}
        {...form.register(key, { valueAsNumber: true })}
      />
      {form.formState.errors[key] ? (
        <p className="text-danger mt-1 text-sm">{tCommon("required")}</p>
      ) : null}
    </div>
  );

  return (
    <form
      className="rounded-card border-border bg-surface max-w-2xl space-y-5 border p-5 sm:p-7"
      onSubmit={submit}
      noValidate
    >
      {formError ? <ApiErrorBanner error={formError} /> : null}
      {notice ? (
        <p
          role="status"
          className="bg-sage-soft text-ink rounded-lg p-3 text-sm"
        >
          {notice}
        </p>
      ) : null}
      <div>
        <label htmlFor="regionCode" className={labelClass}>
          {t("region")}
        </label>
        <select
          id="regionCode"
          disabled={Boolean(listingId)}
          className={`${inputClass} disabled:bg-surface-soft`}
          {...form.register("regionCode")}
        >
          <option value="">{tCommon("required")}</option>
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
        {form.formState.errors.regionCode ? (
          <p className="text-danger mt-1 text-sm">{tCommon("required")}</p>
        ) : null}
      </div>
      {numberField("availableLiters", t("availableLiters"), 1)}
      {numberField("fatPercent", t("fatPercent"), 0, 100, "any")}
      {numberField("pricePerLiterUzs", t("pricePerLiter"), 1)}
      <div>
        <label htmlFor="availableFrom" className={labelClass}>
          {t("availableFrom")}
        </label>
        <input
          id="availableFrom"
          type="datetime-local"
          className={inputClass}
          {...form.register("availableFrom")}
        />
        {form.formState.errors.availableFrom ? (
          <p className="text-danger mt-1 text-sm">{tCommon("required")}</p>
        ) : null}
      </div>
      <div>
        <label htmlFor="availableUntil" className={labelClass}>
          {t("availableUntil")}
        </label>
        <input
          id="availableUntil"
          type="datetime-local"
          className={inputClass}
          {...form.register("availableUntil")}
        />
        {form.formState.errors.availableUntil ? (
          <p className="text-danger mt-1 text-sm">{tCommon("required")}</p>
        ) : null}
      </div>
      <button
        type="submit"
        disabled={form.formState.isSubmitting}
        className="bg-sage min-h-12 w-full rounded-full px-6 font-semibold text-white disabled:opacity-60"
      >
        {form.formState.isSubmitting ? tCommon("loading") : t("saveOffer")}
      </button>
      {listing &&
      (listing.status === "active" || listing.status === "paused") ? (
        <div className="flex flex-col gap-3 sm:flex-row">
          <button
            type="button"
            onClick={() =>
              void setStatus(listing.status === "active" ? "paused" : "active")
            }
            className="border-border min-h-11 flex-1 rounded-full border px-5 font-semibold"
          >
            {listing.status === "active" ? t("pauseOffer") : t("publishOffer")}
          </button>
          <button
            type="button"
            onClick={() => void archive()}
            className="border-danger/40 text-danger min-h-11 flex-1 rounded-full border px-5 font-semibold"
          >
            {t("archiveOffer")}
          </button>
        </div>
      ) : null}
    </form>
  );
}
