import { useT } from "@trenova/shared/i18n/use-t";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import type {
  GraphQLSelectOptionsConfig,
  SelectOption as GraphQLSelectOption,
} from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import type { FieldValues } from "react-hook-form";

type RefEntityConfig = {
  label: string;
  graphql: GraphQLSelectOptionsConfig;
};

/** A record as a report names it: its code before its label when it has one of its own. */
function refDisplay(option: GraphQLSelectOption): string {
  const code = selectOptionMetaString(option, "code");
  return code && code !== option.label ? `${code} - ${option.label}` : option.label || option.id;
}

// Catalog entities the run dialog and builder can resolve through an
// autocomplete. Keys match `reportcatalog` entity keys; entities without a
// select-options surface (stops, moves, line items) are deliberately absent.
export const REPORT_REF_ENTITIES: Record<string, RefEntityConfig> = {
  customer: { label: "Customer", graphql: { resource: "CUSTOMER" } },
  worker: { label: "Worker", graphql: { resource: "WORKER" } },
  tractor: { label: "Tractor", graphql: { resource: "TRACTOR" } },
  trailer: { label: "Trailer", graphql: { resource: "TRAILER" } },
  location: { label: "Location", graphql: { resource: "LOCATION" } },
  location_category: { label: "Location category", graphql: { resource: "LOCATION_CATEGORY" } },
  equipment_type: { label: "Equipment type", graphql: { resource: "EQUIPMENT_TYPE" } },
  equipment_manufacturer: {
    label: "Equipment manufacturer",
    graphql: { resource: "EQUIPMENT_MANUFACTURER" },
  },
  fleet_code: { label: "Fleet code", graphql: { resource: "FLEET_CODE" } },
  shipment_type: { label: "Shipment type", graphql: { resource: "SHIPMENT_TYPE" } },
  service_type: { label: "Service type", graphql: { resource: "SERVICE_TYPE" } },
  commodity: { label: "Commodity", graphql: { resource: "COMMODITY" } },
  hazardous_material: { label: "Hazardous material", graphql: { resource: "HAZARDOUS_MATERIAL" } },
  order: { label: "Order", graphql: { resource: "ORDER" } },
  shipment: { label: "Shipment", graphql: { resource: "SHIPMENT" } },
};

const REF_SELECT_PLACEHOLDERS: Record<string, string> = defineLabels({
  customer: "Select customer",
  worker: "Select worker",
  tractor: "Select tractor",
  trailer: "Select trailer",
  location: "Select location",
  location_category: "Select location category",
  equipment_type: "Select equipment type",
  equipment_manufacturer: "Select equipment manufacturer",
  fleet_code: "Select fleet code",
  shipment_type: "Select shipment type",
  service_type: "Select service type",
  commodity: "Select commodity",
  hazardous_material: "Select hazardous material",
  order: "Select order",
  shipment: "Select shipment",
});

const REF_ADD_PLACEHOLDERS: Record<string, string> = defineLabels({
  customer: "Add customer...",
  worker: "Add worker...",
  tractor: "Add tractor...",
  trailer: "Add trailer...",
  location: "Add location...",
  location_category: "Add location category...",
  equipment_type: "Add equipment type...",
  equipment_manufacturer: "Add equipment manufacturer...",
  fleet_code: "Add fleet code...",
  shipment_type: "Add shipment type...",
  service_type: "Add service type...",
  commodity: "Add commodity...",
  hazardous_material: "Add hazardous material...",
  order: "Add order...",
  shipment: "Add shipment...",
});

export const REPORT_REF_ENTITY_CHOICES: { value: string; label: string }[] = Object.entries(
  REPORT_REF_ENTITIES,
).map(([value, config]) => ({ value, label: config.label }));

type ReportRefAutocompleteProps = {
  entityKey: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  clearable?: boolean;
};

export function ReportRefAutocomplete({
  entityKey,
  value,
  onChange,
  placeholder,
  clearable = true,
}: ReportRefAutocompleteProps) {
  const t = useT();
  const config = REPORT_REF_ENTITIES[entityKey];
  if (!config) {
    return (
      <Input
        value={value}
        placeholder={placeholder ?? t("Enter an ID")}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  }

  return (
    <Autocomplete<GraphQLSelectOption, FieldValues>
      graphql={config.graphql}
      value={value}
      onChange={(next) => onChange(next ? String(next) : "")}
      renderOption={refDisplay}
      getOptionValue={(option) => option.id}
      getDisplayValue={refDisplay}
      placeholder={placeholder ?? REF_SELECT_PLACEHOLDERS[entityKey]}
      clearable={clearable}
    />
  );
}

type ReportRefMultiAutocompleteProps = {
  entityKey: string;
  values: string[];
  onChange: (values: string[]) => void;
};

// Multi-select over the same autocomplete: picking appends a chip, chips are
// individually removable. Labels for already-selected values resolve through
// the autocomplete's own selected-value lookup, so the chips show raw IDs
// only until the option cache warms.
export function ReportRefMultiAutocomplete({
  entityKey,
  values,
  onChange,
}: ReportRefMultiAutocompleteProps) {
  const t = useT();

  const config = REPORT_REF_ENTITIES[entityKey];
  const [labels, setLabels] = useState<Record<string, string>>({});

  if (!config) {
    return (
      <Input
        value={values.join(", ")}
        placeholder={t("Comma-separated values")}
        onChange={(event) =>
          onChange(
            event.target.value
              .split(",")
              .map((part) => part.trim())
              .filter(Boolean),
          )
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      <Autocomplete<GraphQLSelectOption, FieldValues>
        graphql={config.graphql}
        value=""
        onChange={() => undefined}
        onOptionChange={(option) => {
          if (!option || values.includes(option.id)) return;
          setLabels((prev) => ({ ...prev, [option.id]: refDisplay(option) }));
          onChange([...values, option.id]);
        }}
        renderOption={refDisplay}
        getOptionValue={(option) => option.id}
        getDisplayValue={refDisplay}
        placeholder={REF_ADD_PLACEHOLDERS[entityKey]}
        clearable={false}
      />
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {values.map((id) => (
            <Badge key={id} variant="neutral" className="gap-1 pr-1">
              <span className="max-w-40 truncate">{labels[id] ?? id}</span>
              <Button
                variant="ghost"
                size="icon"
                className="size-4"
                aria-label={t("Remove value")}
                onClick={() => onChange(values.filter((v) => v !== id))}
              >
                <XCloseIcon className="size-3" />
              </Button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
