import { hasLocale, NextIntlClientProvider } from "next-intl";
import { getMessages, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { routing } from "@/i18n/routing";
import { AppProviders } from "@/components/providers/app-providers";
import { getPublicRuntimeConfig } from "@/lib/config/runtime";

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
  children,
  params,
}: Readonly<{
  children: ReactNode;
  params: Promise<{ locale: string }>;
}>) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) notFound();

  setRequestLocale(locale);
  const messages = await getMessages();
  const { apiBaseUrl } = getPublicRuntimeConfig();

  return (
    <AppProviders apiBaseUrl={apiBaseUrl}>
      <NextIntlClientProvider messages={messages}>
        {children}
      </NextIntlClientProvider>
    </AppProviders>
  );
}
