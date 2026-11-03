import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AccountGate, AccountWorkspace } from "@/components/auth/account-gate";
import { SecurityClient } from "@/components/workspace/account-client";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "account" });
  return { title: t("securityTitle") };
}

export default async function AccountSecurityPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("account");
  return (
    <AccountGate>
      <AccountWorkspace title={t("securityTitle")}>
        <SecurityClient />
      </AccountWorkspace>
    </AccountGate>
  );
}
