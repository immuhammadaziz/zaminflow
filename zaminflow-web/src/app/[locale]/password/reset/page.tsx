import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { AuthCard } from "@/components/auth/auth-card";
import { ResetPasswordForm } from "@/components/auth/password-reset-forms";

type Props = Readonly<{
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ token?: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "auth" });
  return { title: t("resetTitle") };
}

export default async function ResetPasswordPage({
  params,
  searchParams,
}: Props) {
  const [{ locale }, query] = await Promise.all([params, searchParams]);
  setRequestLocale(locale);
  const t = await getTranslations("auth");
  return (
    <AuthCard title={t("resetTitle")} description={t("resetDescription")}>
      <ResetPasswordForm token={query.token ?? ""} />
    </AuthCard>
  );
}
