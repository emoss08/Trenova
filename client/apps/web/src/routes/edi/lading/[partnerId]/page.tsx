import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { edi } from "@/lib/queries/edi";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useQuery } from "@tanstack/react-query";
import { lazy } from "react";
import { useParams } from "react-router";
import { LadingStudioSkeleton } from "../_components/lading-studio-skeleton";

const LadingStudio = lazy(() => import("../_components/lading-studio"));

/** One partner's Lading project: its versions, files, samples, tests and releases. */
export function LadingStudioPage() {
  const { partnerId = "" } = useParams();
  const t = useT();
  const { data: partner } = useQuery(edi.partner(partnerId));
  useBreadcrumbLabel(partner?.name);

  return (
    <PageLayout
      fill
      pageHeaderProps={{
        title: partner?.name ?? t("Lading studio"),
        description: t("Write, test and release how Trenova trades EDI with this partner"),
      }}
    >
      <LazyComponent fallback={<LadingStudioSkeleton />}>
        <LadingStudio key={partnerId} partnerId={partnerId} partnerName={partner?.name ?? ""} />
      </LazyComponent>
    </PageLayout>
  );
}
