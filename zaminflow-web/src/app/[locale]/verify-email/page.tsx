import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AuthCard } from "@/components/auth/auth-card";
import { VerifyEmailForm } from "@/components/auth/verify-email-form";

type Props = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ token?: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "auth" });
  return { title: t("emailVerificationTitle") };
}

export default async function VerifyEmailPage({ params, searchParams }: Props) {
  const [{ locale }, query] = await Promise.all([params, searchParams]);
  setRequestLocale(locale);
  const t = await getTranslations("auth");
  return (
    <AuthCard
      title={t("emailVerificationTitle")}
      description={t("emailVerificationDescription")}
    >
      <VerifyEmailForm token={query.token ?? ""} />
    </AuthCard>
  );
}
