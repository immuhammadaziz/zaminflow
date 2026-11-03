import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { ApplicationReviewClient } from "@/components/workspace/admin-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{
  params: Promise<{ locale: string; userId: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "admin" });
  return { title: t("applicationDetails") };
}

export default async function AdminApplicationPage({ params }: Props) {
  const { locale, userId } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("admin");
  return (
    <RoleGate role="admin">
      <WorkspaceShell role="admin" title={t("applicationDetails")}>
        <ApplicationReviewClient userId={userId} />
      </WorkspaceShell>
    </RoleGate>
  );
}
