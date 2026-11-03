import type { Metadata } from "next";
import { headers } from "next/headers";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "ZaminFlow — From farm supply to trusted trade",
    template: "%s | ZaminFlow",
  },
  description:
    "A shared marketplace workspace for Uzbekistan's dairy farmers and processors.",
};

export const dynamic = "force-dynamic";

export default async function RootLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  const requestHeaders = await headers();
  const locale = requestHeaders.get("x-locale");

  return (
    <html lang={locale === "ru" || locale === "en" ? locale : "uz"}>
      <body>{children}</body>
    </html>
  );
}
