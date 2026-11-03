"use client";

import { useTranslations } from "next-intl";

export default function LocaleError({
  error,
  retry,
}: Readonly<{
  error: Error & { digest?: string };
  retry: () => void;
}>) {
  const t = useTranslations("system");
  return (
    <main id="main-content" className="mx-auto max-w-3xl px-5 py-24 sm:px-8">
      <p className="text-sage text-sm font-semibold">ZaminFlow</p>
      <h1 className="font-display text-ink mt-4 text-4xl font-bold">
        {t("errorTitle")}
      </h1>
      <p className="text-muted mt-4">{t("errorDescription")}</p>
      {error.digest ? (
        <p className="text-muted mt-3 text-xs break-all">
          {t("errorReference")}: {error.digest}
        </p>
      ) : null}
      <button
        type="button"
        onClick={() => retry()}
        className="bg-sage mt-8 min-h-11 rounded-full px-5 font-semibold text-white"
      >
        {t("retryAction")}
      </button>
    </main>
  );
}
