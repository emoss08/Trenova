import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { GraphQLSelectOptionsConfig } from "@/lib/graphql/select-options";
import type { SELECT_OPTIONS_ENDPOINTS } from "@/types/server";
import { XIcon } from "lucide-react";
import { useState } from "react";
import type { FieldValues } from "react-hook-form";

type RefOption = Record<string, unknown>;

function optionString(option: RefOption, key: string): string {
  const value = option[key];
  return typeof value === "string" ? value : "";
}

function codeNameDisplay(option: RefOption): string {
  const code = optionString(option, "code");
  const name = optionString(option, "name");
  if (code && name) return `${code} - ${name}`;
  return name || code || optionString(option, "id");
}

function nameDisplay(option: RefOption): string {
  return optionString(option, "name") || optionString(option, "id");
}

function labelDisplay(option: RefOption): string {
  return optionString(option, "label") || optionString(option, "id");
}

type RefEntityConfig = {
  label: string;
  link: SELECT_OPTIONS_ENDPOINTS;
  graphql?: GraphQLSelectOptionsConfig;
  display: (option: RefOption) => string;
};

// Catalog entities the run dialog and builder can resolve through an
// autocomplete. Keys match `reportcatalog` entity keys; entities without a
// select-options surface (stops, moves, line items) are deliberately absent.
export const REPORT_REF_ENTITIES: Record<string, RefEntityConfig> = {
  customer: {
    label: "Customer",
    link: "/customers/select-options/",
    display: codeNameDisplay,
  },
  worker: {
    label: "Worker",
    link: "/workers/select-options/",
    graphql: { resource: "WORKER" },
    display: labelDisplay,
  },
  tractor: {
    label: "Tractor",
    link: "/tractors/select-options/",
    graphql: { resource: "TRACTOR" },
    display: labelDisplay,
  },
  trailer: {
    label: "Trailer",
    link: "/trailers/select-options/",
    graphql: { resource: "TRAILER" },
    display: labelDisplay,
  },
  location: {
    label: "Location",
    link: "/locations/select-options/",
    display: codeNameDisplay,
  },
  location_category: {
    label: "Location Category",
    link: "/location-categories/select-options/",
    display: nameDisplay,
  },
  equipment_type: {
    label: "Equipment Type",
    link: "/equipment-types/select-options/",
    graphql: { resource: "EQUIPMENT_TYPE" },
    display: labelDisplay,
  },
  equipment_manufacturer: {
    label: "Equipment Manufacturer",
    link: "/equipment-manufacturers/select-options/",
    graphql: { resource: "EQUIPMENT_MANUFACTURER" },
    display: labelDisplay,
  },
  fleet_code: {
    label: "Fleet Code",
    link: "/fleet-codes/select-options/",
    display: codeNameDisplay,
  },
  shipment_type: {
    label: "Shipment Type",
    link: "/shipment-types/select-options/",
    display: codeNameDisplay,
  },
  service_type: {
    label: "Service Type",
    link: "/service-types/select-options/",
    display: codeNameDisplay,
  },
  commodity: {
    label: "Commodity",
    link: "/commodities/select-options/",
    display: nameDisplay,
  },
  hazardous_material: {
    label: "Hazardous Material",
    link: "/hazardous-materials/select-options/",
    display: nameDisplay,
  },
  order: {
    label: "Order",
    link: "/orders/select-options/",
    graphql: { resource: "ORDER" },
    display: labelDisplay,
  },
  shipment: {
    label: "Shipment",
    link: "/shipments/select-options/",
    graphql: { resource: "SHIPMENT" },
    display: labelDisplay,
  },
};

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
  const config = REPORT_REF_ENTITIES[entityKey];
  if (!config) {
    return (
      <Input
        value={value}
        placeholder={placeholder ?? "Enter an ID"}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  }

  return (
    <Autocomplete<RefOption, FieldValues>
      link={config.link}
      graphql={config.graphql}
      value={value}
      onChange={(next) => onChange(next ? String(next) : "")}
      renderOption={(option) => config.display(option)}
      getOptionValue={(option) => optionString(option, "id")}
      getDisplayValue={(option) => config.display(option)}
      placeholder={placeholder ?? `Select ${config.label.toLowerCase()}`}
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
  const config = REPORT_REF_ENTITIES[entityKey];
  const [labels, setLabels] = useState<Record<string, string>>({});

  return (
    <div className="flex flex-col gap-1.5">
      <Autocomplete<RefOption, FieldValues>
        link={config?.link ?? "/customers/select-options/"}
        graphql={config?.graphql}
        value=""
        onChange={() => undefined}
        onOptionChange={(option) => {
          if (!option) return;
          const id = optionString(option, "id");
          if (!id || values.includes(id)) return;
          setLabels((prev) => ({ ...prev, [id]: config?.display(option) ?? id }));
          onChange([...values, id]);
        }}
        renderOption={(option) => config?.display(option) ?? optionString(option, "id")}
        getOptionValue={(option) => optionString(option, "id")}
        getDisplayValue={(option) => config?.display(option) ?? optionString(option, "id")}
        placeholder={`Add ${config?.label.toLowerCase() ?? "value"}...`}
        clearable={false}
      />
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {values.map((id) => (
            <Badge key={id} variant="secondary" className="gap-1 pr-1">
              <span className="max-w-40 truncate">{labels[id] ?? id}</span>
              <Button
                variant="ghost"
                size="icon"
                className="size-4"
                aria-label="Remove value"
                onClick={() => onChange(values.filter((v) => v !== id))}
              >
                <XIcon className="size-3" />
              </Button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
