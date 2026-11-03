import { ArrowDown, ArrowLeftRight, ClipboardCheck, Milk } from "lucide-react";
import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Link } from "@/i18n/navigation";
import { routing } from "@/i18n/routing";

type Props = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "landing" });
  return {
    title: `${t("titleFirst")} ${t("titleSecond")}`,
    description: t("description"),
  };
}

export default async function LandingPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("landing");
  const nav = await getTranslations("nav");

  return (
    <main id="main-content">
      <header className="border-border bg-canvas border-b">
        <div className="mx-auto flex min-h-20 max-w-7xl items-center justify-between gap-5 px-5 sm:px-8">
          <Link href="/" className="flex items-center gap-3 rounded-lg">
            <BrandMark />
            <span className="font-display text-ink text-2xl font-bold tracking-tight">
              ZaminFlow
            </span>
          </Link>
          <nav
            aria-label="Main navigation"
            className="text-muted hidden items-center gap-8 text-sm lg:flex"
          >
            <a
              href="#how-it-works"
              className="hover:text-ink transition-colors"
            >
              {nav("howItWorks")}
            </a>
            <a href="#farmers" className="hover:text-ink transition-colors">
              {nav("forFarmers")}
            </a>
            <a href="#processors" className="hover:text-ink transition-colors">
              {nav("forProcessors")}
            </a>
          </nav>
          <div className="hidden items-center gap-3 sm:flex">
            <LanguageLinks locale={locale} label={nav("languages")} />
            <Link
              href="/sign-in"
              className="border-border text-ink inline-flex min-h-11 items-center rounded-full border px-5 text-sm font-semibold transition-colors hover:bg-white"
            >
              {nav("signIn")}
            </Link>
            <Link
              href="/sign-up"
              className="bg-sage inline-flex min-h-11 items-center rounded-full px-5 text-sm font-semibold text-white transition-colors hover:bg-[#244b38]"
            >
              {nav("signUp")}
            </Link>
          </div>
          <details className="group relative sm:hidden">
            <summary
              aria-label={nav("menu")}
              className="border-border text-ink flex size-11 cursor-pointer list-none items-center justify-center rounded-full border marker:hidden"
            >
              <span aria-hidden="true" className="text-xl leading-none">
                ☰
              </span>
            </summary>
            <div className="border-border absolute top-14 right-0 z-20 flex w-64 flex-col gap-2 rounded-xl border bg-white p-3 shadow-xl">
              <a
                className="hover:bg-canvas rounded-lg px-3 py-3 text-sm"
                href="#how-it-works"
              >
                {nav("howItWorks")}
              </a>
              <a
                className="hover:bg-canvas rounded-lg px-3 py-3 text-sm"
                href="#farmers"
              >
                {nav("forFarmers")}
              </a>
              <a
                className="hover:bg-canvas rounded-lg px-3 py-3 text-sm"
                href="#processors"
              >
                {nav("forProcessors")}
              </a>
              <div className="border-border flex items-center justify-between gap-2 border-t pt-3">
                <LanguageLinks locale={locale} label={nav("languages")} />
                <Link
                  href="/sign-in"
                  className="border-border rounded-full border px-4 py-2 text-sm font-semibold"
                >
                  {nav("signIn")}
                </Link>
              </div>
              <Link
                href="/sign-up"
                className="bg-sage rounded-full px-4 py-3 text-center text-sm font-semibold text-white"
              >
                {nav("signUp")}
              </Link>
            </div>
          </details>
        </div>
      </header>

      <section className="mx-auto grid max-w-7xl items-center gap-12 px-5 py-16 sm:px-8 sm:py-24 lg:grid-cols-[1.05fr_0.95fr] lg:gap-16">
        <div>
          <p className="bg-sage-soft text-sage mb-7 inline-flex items-center gap-2 rounded-full px-4 py-2 text-xs font-bold tracking-wide sm:text-sm">
            <span aria-hidden="true" className="bg-sage size-2 rounded-full" />
            {t("eyebrow")}
          </p>
          <h1 className="font-display text-ink max-w-3xl text-4xl leading-[1.12] font-bold tracking-tight sm:text-5xl lg:text-6xl">
            {t("titleFirst")}{" "}
            <span className="text-sage">{t("titleSecond")}</span>
          </h1>
          <p className="text-muted mt-6 max-w-2xl text-lg leading-8">
            {t("description")}
          </p>
          <p className="text-muted mt-2 max-w-2xl text-base leading-7">
            {t("supporting")}
          </p>
          <div className="mt-9 flex flex-col gap-3 sm:flex-row">
            <Link
              href="/sign-up"
              className="bg-sage inline-flex min-h-12 items-center justify-center gap-2 rounded-full px-6 text-sm font-bold text-white transition-colors hover:bg-[#244b38]"
            >
              {t("primaryAction")}
              <ArrowDown aria-hidden="true" size={16} className="-rotate-90" />
            </Link>
            <a
              href="#how-it-works"
              className="border-border text-ink hover:bg-canvas inline-flex min-h-12 items-center justify-center rounded-full border bg-white px-6 text-sm font-bold transition-colors"
            >
              {t("secondaryAction")}
            </a>
          </div>
          <p className="text-muted mt-5 text-sm">{t("trustDescription")}</p>
        </div>

        <div
          id="how-it-works"
          className="rounded-panel border-border border bg-white p-5 shadow-[0_24px_70px_rgba(43,33,27,0.07)] sm:p-7"
        >
          <div className="rounded-card bg-canvas p-4 sm:p-5">
            <p className="text-ink font-semibold">{t("workflowTitle")}</p>
            <p className="text-muted mt-1 text-sm">
              {t("workflowDescription")}
            </p>
          </div>
          <ol className="mt-6 space-y-5">
            <WorkflowStep
              index="01"
              title={t("stepOneTitle")}
              description={t("stepOneDescription")}
              icon={<Milk aria-hidden="true" size={19} />}
              tone="sage"
            />
            <WorkflowStep
              index="02"
              title={t("stepTwoTitle")}
              description={t("stepTwoDescription")}
              icon={<ArrowLeftRight aria-hidden="true" size={19} />}
              tone="gold"
            />
            <WorkflowStep
              index="03"
              title={t("stepThreeTitle")}
              description={t("stepThreeDescription")}
              icon={<ClipboardCheck aria-hidden="true" size={19} />}
              tone="blue"
            />
          </ol>
          <p className="border-border text-muted mt-6 border-t pt-4 text-xs leading-5">
            {t("boundary")}
          </p>
        </div>
      </section>

      <section className="border-border border-y bg-white">
        <div className="mx-auto max-w-7xl px-5 py-14 sm:px-8 sm:py-16">
          <h2 className="font-display text-ink text-3xl font-bold tracking-tight sm:text-4xl">
            {t("audienceTitle")}
          </h2>
          <div className="mt-8 grid gap-5 md:grid-cols-2">
            <AudienceCard
              id="farmers"
              label={t("farmerLabel")}
              title={t("farmerTitle")}
              description={t("farmerDescription")}
            />
            <AudienceCard
              id="processors"
              label={t("processorLabel")}
              title={t("processorTitle")}
              description={t("processorDescription")}
            />
          </div>
        </div>
      </section>

      <footer className="text-muted mx-auto flex max-w-7xl flex-col gap-4 px-5 py-8 text-sm sm:flex-row sm:items-center sm:justify-between sm:px-8">
        <Link href="/" className="font-display text-ink text-lg font-bold">
          ZaminFlow
        </Link>
        <p>{t("footer")}</p>
        <Link
          href="/sign-up"
          className="text-sage font-semibold hover:underline"
        >
          {t("primaryAction")}
        </Link>
      </footer>
    </main>
  );
}

function BrandMark() {
  return (
    <span className="bg-sage-soft text-sage flex size-10 items-center justify-center rounded-xl">
      <svg aria-hidden="true" viewBox="0 0 24 24" className="size-6">
        <path
          d="M5 19 19 5M6 11h12M9 16h9"
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="2"
        />
      </svg>
    </span>
  );
}

function LanguageLinks({
  locale,
  label,
}: Readonly<{ locale: string; label: string }>) {
  return (
    <nav aria-label={label} className="flex items-center gap-1 text-xs">
      {routing.locales.map((item) => (
        <Link
          key={item}
          href="/"
          locale={item}
          aria-current={locale === item ? "page" : undefined}
          className={`min-h-11 min-w-11 rounded-full px-2 py-2 font-semibold transition-colors ${
            locale === item
              ? "bg-sage-soft text-sage"
              : "text-muted hover:text-ink"
          }`}
        >
          {item.toUpperCase()}
        </Link>
      ))}
    </nav>
  );
}

function WorkflowStep({
  index,
  title,
  description,
  icon,
  tone,
}: Readonly<{
  index: string;
  title: string;
  description: string;
  icon: ReactNode;
  tone: "sage" | "gold" | "blue";
}>) {
  const toneClass = {
    sage: "bg-sage-soft text-sage",
    gold: "bg-[#f4ebdd] text-[#87520f]",
    blue: "bg-[#e8eef1] text-[#315b76]",
  }[tone];

  return (
    <li className="flex gap-4">
      <span
        className={`flex size-11 shrink-0 items-center justify-center rounded-full ${toneClass}`}
      >
        {icon}
      </span>
      <div className="border-border flex-1 border-b pb-4 last:border-0 last:pb-0">
        <p className="text-ink text-sm font-bold">
          <span className="text-muted mr-2 text-xs font-bold">{index}</span>
          {title}
        </p>
        <p className="text-muted mt-1 text-sm">{description}</p>
      </div>
    </li>
  );
}

function AudienceCard({
  id,
  label,
  title,
  description,
}: Readonly<{
  id: string;
  label: string;
  title: string;
  description: string;
}>) {
  return (
    <article
      id={id}
      className="rounded-panel border-border bg-canvas border p-6 sm:p-7"
    >
      <p className="text-sage text-xs font-bold tracking-[0.12em] uppercase">
        {label}
      </p>
      <h3 className="font-display text-ink mt-3 text-xl leading-snug font-bold">
        {title}
      </h3>
      <p className="text-muted mt-2 text-sm leading-6">{description}</p>
    </article>
  );
}
