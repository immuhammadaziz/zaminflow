import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { OrderDetailClient } from "@/components/workspace/orders-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{
  params: Promise<{ locale: string; orderId: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketplace" });
  return { title: t("orderDetails") };
}

export default async function FarmerOrderPage({ params }: Props) {
  const { locale, orderId } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("marketplace");
  return (
    <RoleGate role="farmer">
      <WorkspaceShell role="farmer" title={t("orderDetails")}>
        <OrderDetailClient orderId={orderId} role="farmer" />
      </WorkspaceShell>
    </RoleGate>
  );
}
