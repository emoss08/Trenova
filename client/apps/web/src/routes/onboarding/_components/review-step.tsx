import { SectionPanel } from "@/components/section-panel";
import { operationTypeLabel, timezoneLabel, type OnboardingStepId } from "@/lib/onboarding-form";
import type { OnboardingFormValues } from "@/types/onboarding";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { useWatch, type Control } from "react-hook-form";

function valueOrEmpty(value: string | undefined): ReactNode {
  return value && value.trim() !== "" ? value : <DescriptionEmpty />;
}

export function ReviewStep({
  control,
  stateLabel,
  onEdit,
}: {
  control: Control<OnboardingFormValues>;
  stateLabel: string;
  onEdit: (step: OnboardingStepId) => void;
}) {
  const t = useT();
  const [organization, operationType, loadSampleData] = useWatch({
    control,
    name: ["organization", "operationType", "loadSampleData"],
  });

  const cityLine = [organization.city, stateLabel, organization.postalCode]
    .filter((part) => part && part.trim() !== "")
    .join(", ");

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel
        title={t("Company profile")}
        action={
          <Button type="button" variant="ghost" size="sm" onClick={() => onEdit("company")}>
            {t("Edit")}
          </Button>
        }
      >
        <div className="p-4">
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Company name")}>
              {valueOrEmpty(organization.name)}
            </DescriptionItem>
            <DescriptionItem label={t("Timezone")}>
              {organization.timezone ? (
                t(timezoneLabel(organization.timezone))
              ) : (
                <DescriptionEmpty />
              )}
            </DescriptionItem>
            <DescriptionItem label={t("Address")} span="full">
              {organization.addressLine1 ? (
                <>
                  {organization.addressLine1}
                  {cityLine ? <span className="block">{cityLine}</span> : null}
                </>
              ) : (
                <DescriptionEmpty />
              )}
            </DescriptionItem>
            <DescriptionItem label={t("SCAC code")}>
              {valueOrEmpty(organization.scacCode?.toUpperCase())}
            </DescriptionItem>
            <DescriptionItem label={t("DOT number")} numeric>
              {valueOrEmpty(organization.dotNumber)}
            </DescriptionItem>
          </DescriptionList>
        </div>
      </SectionPanel>

      <SectionPanel
        title={t("Setup")}
        action={
          <Button type="button" variant="ghost" size="sm" onClick={() => onEdit("operation")}>
            {t("Edit")}
          </Button>
        }
      >
        <div className="p-4">
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Operation type")}>
              {t(operationTypeLabel(operationType))}
            </DescriptionItem>
            <DescriptionItem label={t("Sample data")}>
              {loadSampleData ? t("Load sample data") : t("Start empty")}
            </DescriptionItem>
          </DescriptionList>
        </div>
      </SectionPanel>
    </div>
  );
}
