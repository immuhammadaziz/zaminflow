import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import { RoleGate } from "@/components/auth/role-gate";
import { ListingEditor } from "@/components/workspace/listings-client";
import { WorkspaceShell } from "@/components/workspace/workspace-shell";

type Props = Readonly<{
  params: Promise<{ locale: string; listingId: string }>;
}>;

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: "marketplace" });
  return { title: t("editListing") };
}

export default async function FarmerListingPage({ params }: Props) {
  const { locale, listingId } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("marketplace");
  return (
    <RoleGate role="farmer">
      <WorkspaceShell role="farmer" title={t("editListing")}>
        <ListingEditor listingId={listingId} />
      </WorkspaceShell>
    </RoleGate>
  );
}
