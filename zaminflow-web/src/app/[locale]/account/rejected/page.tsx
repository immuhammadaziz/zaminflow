import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AccountGate, AccountWorkspace } from "@/components/auth/account-gate";
import { ProfileClient } from "@/components/workspace/account-client";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "auth" });
  return { title: t("rejectedTitle") };
}

export default async function RejectedAccountPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("auth");
  return (
    <AccountGate>
      <AccountWorkspace title={t("rejectedTitle")}>
        <ProfileClient />
      </AccountWorkspace>
    </AccountGate>
  );
}
