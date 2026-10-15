import {
  DescriptionListSkeleton,
  SectionBodySkeleton,
  SectionPanelSkeleton,
} from "@/components/section-panel-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

function DeliveryPlaceholder() {
  return <DescriptionListSkeleton items={4} columns={2} />;
}

/** The delivery settings while they are read: method, where it goes, and the last test. */
export function FactoringDeliveryBodySkeleton() {
  const t = useT();
  return (
    <SectionBodySkeleton label={t("Loading delivery settings")}>
      <DeliveryPlaceholder />
    </SectionBodySkeleton>
  );
}

/** The delivery section while its code loads. */
export function FactoringDeliverySectionSkeleton() {
  const t = useT();
  return (
    <SectionPanelSkeleton label={t("Loading delivery settings")} action>
      <div className="p-3">
        <DeliveryPlaceholder />
      </div>
    </SectionPanelSkeleton>
  );
}
