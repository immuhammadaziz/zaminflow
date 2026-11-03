import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AccountGate, AccountWorkspace } from "@/components/auth/account-gate";
import { ProfileClient } from "@/components/workspace/account-client";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "account" });
  return { title: t("profileTitle") };
}

export default async function AccountProfilePage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("account");
  return (
    <AccountGate>
      <AccountWorkspace title={t("profileTitle")}>
        <ProfileClient />
      </AccountWorkspace>
    </AccountGate>
  );
}
