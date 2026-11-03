import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { ListingOrderClient } from "@/components/workspace/listing-order-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{
  params: Promise<{ locale: string; listingId: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketplace" });
  return { title: t("placeOrder") };
}

export default async function MarketplaceListingPage({ params }: Props) {
  const { locale, listingId } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("marketplace");
  return (
    <RoleGate role="processor">
      <WorkspaceShell role="processor" title={t("placeOrder")}>
        <ListingOrderClient listingId={listingId} />
      </WorkspaceShell>
    </RoleGate>
  );
}
