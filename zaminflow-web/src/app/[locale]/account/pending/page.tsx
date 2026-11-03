import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AccountGate } from "@/components/auth/account-gate";
import { AuthCard } from "@/components/auth/auth-card";
import { AccountStatusClient } from "@/components/auth/account-status-client";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "auth" });
  return { title: t("awaitingApprovalTitle") };
}

export default async function PendingAccountPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("auth");
  return (
    <AccountGate>
      <AuthCard
        title={t("awaitingApprovalTitle")}
        description={t("awaitingApprovalDescription")}
      >
        <AccountStatusClient status="pending" />
      </AuthCard>
    </AccountGate>
  );
}
