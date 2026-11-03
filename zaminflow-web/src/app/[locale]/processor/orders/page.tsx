import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { OrdersClient } from "@/components/workspace/orders-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{ params: Promise<{ locale: string }> }>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketplace" });
  return { title: t("myOrders") };
}

export default async function ProcessorOrdersPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("marketplace");
  return (
    <RoleGate role="processor">
      <WorkspaceShell role="processor" title={t("myOrders")}>
        <OrdersClient role="processor" />
      </WorkspaceShell>
    </RoleGate>
  );
}
