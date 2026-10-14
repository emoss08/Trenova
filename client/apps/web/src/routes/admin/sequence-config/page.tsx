import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { SequenceConfigSkeleton } from "./skeleton";

const SequenceConfigForm = lazy(() => import("./_components/sequence-config-form"));

export function SequenceConfigPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Sequence configuration"),
        description: t("Configure sequence generation formats for shipments and billing workflows"),
      }}
    >
      <LazyComponent fallback={<SequenceConfigSkeleton />}>
        <SequenceConfigForm />
      </LazyComponent>
    </PageLayout>
  );
}
