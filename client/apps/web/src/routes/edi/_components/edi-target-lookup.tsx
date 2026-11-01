import { useT } from "@trenova/shared/i18n/use-t";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import type { EDIMappingEntityType } from "@trenova/shared/types/edi";
import type { FieldValues } from "react-hook-form";
import { joinDisplayParts, type TargetLookupSelection } from "./edi-display-utils";
import { mappingTargetSources } from "./edi-schemas";

type TargetLookupProps = {
  entityType: EDIMappingEntityType;
  label?: string;
  value: string;
  onChange: (target: TargetLookupSelection) => void;
};

/** How a mapped record reads in the profile: its label, then what describes it. */
function targetLabel(option: GraphQLSelectOption): string {
  return joinDisplayParts(option.label, option.description, " - ") || option.id;
}

export function TargetLookup({ entityType, label, value, onChange }: TargetLookupProps) {
  const t = useT();

  return (
    <Autocomplete<GraphQLSelectOption, FieldValues>
      graphql={mappingTargetSources[entityType]}
      label={label}
      value={value}
      placeholder={t("Select local record")}
      clearable
      onChange={(nextValue) => {
        if (!nextValue) {
          onChange({ targetId: "", targetLabel: "" });
        }
      }}
      onOptionChange={(option) =>
        onChange({
          targetId: option?.id ?? "",
          targetLabel: option ? targetLabel(option) : "",
        })
      }
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full min-w-0 flex-col items-start pr-4">
          <span className="w-full truncate">{option.label}</span>
          {option.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
            </span>
          )}
        </div>
      )}
    />
  );
}
