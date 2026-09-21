import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import type { SequenceConfig, SequenceConfigDocument, SequenceType } from "@/types/sequence-config";
import { RotateCcwIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";
import { LocationCodeStrategySection } from "./location-code-sections";
import {
  defaultConfigForType,
  sequenceDescriptions,
  sequenceTitles,
} from "./sequence-config-constants";
import {
  AdvancedSection,
  ContextComponentsSection,
  CoreStructureSection,
  DateComponentsSection,
} from "./sequence-form-sections";
import { SequencePreview } from "./sequence-preview";

type PanelProps = {
  index: number;
  sequenceType: SequenceType;
};

export function SequenceConfigPanel({ index, sequenceType }: PanelProps) {
  const t = useT();

  const { setValue, getValues } = useFormContext<SequenceConfigDocument>();

  const handleReset = () => {
    const current = getValues(`configs.${index}`) as SequenceConfig | undefined;
    if (!current) return;
    setValue(
      `configs.${index}`,
      defaultConfigForType(sequenceType, {
        id: current.id,
        organizationId: current.organizationId,
        businessUnitId: current.businessUnitId,
        version: current.version,
        createdAt: current.createdAt,
        updatedAt: current.updatedAt,
      }),
      { shouldDirty: true, shouldValidate: true, shouldTouch: true },
    );
  };

  const isLocationCode = sequenceType === "location_code";

  return (
    <FormSection
      className="min-w-0 flex-1 gap-6"
      title={sequenceTitles[sequenceType]}
      description={sequenceDescriptions[sequenceType]}
      action={
        <Button type="button" variant="ghost" size="sm" onClick={handleReset} className="gap-1.5">
          <RotateCcwIcon className="size-3.5" />
          {t("Reset to default")}
        </Button>
      }
    >
      <SequencePreview index={index} showTokens={!isLocationCode} />

      {isLocationCode ? (
        <LocationCodeStrategySection index={index} />
      ) : (
        <>
          <CoreStructureSection index={index} />
          <DateComponentsSection index={index} />
          <ContextComponentsSection index={index} />
          <AdvancedSection index={index} />
        </>
      )}
    </FormSection>
  );
}
