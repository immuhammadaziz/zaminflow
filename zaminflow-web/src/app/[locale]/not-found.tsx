"use client";

import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";

export default function LocaleNotFound() {
  const t = useTranslations("system");

  return (
    <main id="main-content" className="mx-auto max-w-3xl px-5 py-24 sm:px-8">
      <p className="text-sage text-sm font-semibold">ZaminFlow</p>
      <h1 className="font-display text-ink mt-4 text-4xl font-bold">
        {t("notFoundTitle")}
      </h1>
      <p className="text-muted mt-4">{t("notFoundDescription")}</p>
      <Link
        href="/"
        className="bg-sage mt-8 inline-flex min-h-11 items-center rounded-full px-5 font-semibold text-white"
      >
        {t("homeAction")}
      </Link>
    </main>
  );
}
