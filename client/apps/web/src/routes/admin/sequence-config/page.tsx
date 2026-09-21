import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

const SequenceConfigForm = lazy(() => import("./_components/sequence-config-form"));

export function SequenceConfigPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Sequence Configuration"),
        description: t("Configure sequence generation formats for shipments and billing workflows"),
      }}
    >
      <SuspenseLoader>
        <SequenceConfigForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
