import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AccountGate, AccountWorkspace } from "@/components/auth/account-gate";
import { DeletionClient } from "@/components/workspace/account-client";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "account" });
  return { title: t("deletionTitle") };
}

export default async function AccountDeletionPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("account");
  return (
    <AccountGate>
      <AccountWorkspace title={t("deletionTitle")}>
        <DeletionClient />
      </AccountWorkspace>
    </AccountGate>
  );
}
