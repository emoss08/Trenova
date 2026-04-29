import { Button } from "@/components/ui/button";
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
    <div className="flex min-w-0 flex-1 flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold tracking-tight text-foreground">
            {sequenceTitles[sequenceType]}
          </h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {sequenceDescriptions[sequenceType]}
          </p>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleReset}
          className="gap-1.5"
        >
          <RotateCcwIcon className="size-3.5" />
          Reset to default
        </Button>
      </div>

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
    </div>
  );
}
