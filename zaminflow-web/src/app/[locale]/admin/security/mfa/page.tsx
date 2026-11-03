import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AdminMfaEnrollment } from "@/components/auth/admin-mfa-enrollment";
import { AuthCard } from "@/components/auth/auth-card";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "auth" });
  return { title: t("adminEnrollmentTitle") };
}

export default async function AdminMfaPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("auth");
  return (
    <AuthCard
      title={t("adminEnrollmentTitle")}
      description={t("adminEnrollmentDescription")}
    >
      <AdminMfaEnrollment />
    </AuthCard>
  );
}
