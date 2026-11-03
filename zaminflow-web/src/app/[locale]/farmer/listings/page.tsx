import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { ListingsClient } from "@/components/workspace/listings-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketplace" });
  return { title: t("listingsTitle") };
}

export default async function FarmerListingsPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("marketplace");
  return (
    <RoleGate role="farmer">
      <WorkspaceShell role="farmer" title={t("listingsTitle")}>
        <ListingsClient role="farmer" />
      </WorkspaceShell>
    </RoleGate>
  );
}
