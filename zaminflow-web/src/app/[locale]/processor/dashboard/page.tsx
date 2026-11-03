import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { DashboardClient } from "@/components/workspace/dashboard-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "dashboard" });
  return { title: t("processorTitle") };
}

export default async function ProcessorDashboardPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("dashboard");
  return (
    <RoleGate role="processor">
      <WorkspaceShell role="processor" title={t("processorTitle")}>
        <DashboardClient role="processor" />
      </WorkspaceShell>
    </RoleGate>
  );
}
