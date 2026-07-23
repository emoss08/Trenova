import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { EDIMappingEntityType } from "@trenova/shared/types/edi";
import type { FieldValues } from "react-hook-form";
import {
  getTargetOptionDescription,
  getTargetOptionLabel,
  getTargetOptionValue,
  type TargetLookupSelection,
} from "./edi-display-utils";
import { mappingTargetEndpoints } from "./edi-schemas";

type TargetLookupProps = {
  entityType: EDIMappingEntityType;
  label?: string;
  value: string;
  onChange: (target: TargetLookupSelection) => void;
};

export function TargetLookup({ entityType, label, value, onChange }: TargetLookupProps) {
  return (
    <Autocomplete<Record<string, unknown>, FieldValues>
      link={mappingTargetEndpoints[entityType]}
      label={label}
      value={value}
      placeholder="Select local record"
      clearable
      onChange={(nextValue) => {
        if (!nextValue) {
          onChange({ targetId: "", targetLabel: "" });
        }
      }}
      onOptionChange={(option) =>
        onChange({
          targetId: getTargetOptionValue(option),
          targetLabel: getTargetOptionLabel(option),
        })
      }
      getOptionValue={getTargetOptionValue}
      getDisplayValue={getTargetOptionLabel}
      renderOption={(option) => {
        const description = getTargetOptionDescription(option);
        return (
          <div className="flex size-full min-w-0 flex-col items-start pr-4">
            <span className="w-full truncate">{getTargetOptionLabel(option)}</span>
            {description && (
              <span className="w-full truncate text-2xs text-muted-foreground">{description}</span>
            )}
          </div>
        );
      }}
    />
  );
}
