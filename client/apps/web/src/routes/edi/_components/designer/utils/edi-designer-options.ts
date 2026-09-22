import type { SelectOption } from "@trenova/shared/types/fields";
import {
  ediDocumentDirectionSchema,
  ediDocumentStatusSchema,
  ediMappingEntityTypeSchema,
  ediMessageStatusSchema,
  ediTemplateElementSourceSchema,
  ediTemplateStatusSchema,
  ediTransactionSetSchema,
  ediValidationModeSchema,
} from "@trenova/shared/types/edi";

const templateElementSourceLabels: Record<
  (typeof ediTemplateElementSourceSchema.options)[number],
  string
> = {
  constant: "Constant",
  fieldPath: "Field path",
  partnerSetting: "Partner setting",
  mapping: "Mapping",
  runtime: "Runtime",
  repeat: "Repeat",
  transform: "Transform",
  starlark: "Starlark",
};

const mappingEntityTypeLabels: Record<(typeof ediMappingEntityTypeSchema.options)[number], string> =
  {
    Customer: "Customer",
    ServiceType: "Service type",
    ShipmentType: "Shipment type",
    FormulaTemplate: "Formula template",
    Location: "Location",
    Commodity: "Commodity",
    AccessorialCharge: "Accessorial charge",
    ServiceFailureReasonCode: "Service failure reason code",
  };

const validationModeLabels: Record<(typeof ediValidationModeSchema.options)[number], string> = {
  Strict: "Strict",
  WarnOnly: "Warn only",
  Disabled: "Disabled",
};

function toOptions<T extends string>(
  values: readonly T[],
  labels: Partial<Record<T, string>> = {},
): SelectOption[] {
  return values.map((value) => ({ value, label: labels[value] ?? value }));
}

export const templateStatusOptions = toOptions(ediTemplateStatusSchema.options);

export const templateElementSourceOptions = toOptions(
  ediTemplateElementSourceSchema.options,
  templateElementSourceLabels,
);

export const transformBaseSourceOptions = templateElementSourceOptions.filter(
  (option) => option.value !== "transform" && option.value !== "starlark",
);

export const mappingEntityTypeOptions = toOptions(
  ediMappingEntityTypeSchema.options,
  mappingEntityTypeLabels,
);

export const validationModeOptions = toOptions(
  ediValidationModeSchema.options,
  validationModeLabels,
);

export const transactionSetOptions = toOptions(ediTransactionSetSchema.options);

export const documentDirectionOptions = toOptions(ediDocumentDirectionSchema.options);

export const documentStatusOptions = toOptions(ediDocumentStatusSchema.options);

export const messageStatusOptions = toOptions(ediMessageStatusSchema.options);

export const conditionModeOptions: SelectOption[] = [
  { value: "none", label: "None" },
  { value: "truthy", label: "Path truthy" },
  { value: "falsey", label: "Path falsey" },
  { value: "comparison", label: "Comparison" },
  { value: "starlarkFunction", label: "Starlark function" },
  { value: "inlineStarlark", label: "Inline starlark" },
];

export const conditionOperatorOptions: SelectOption[] = [
  { value: "==", label: "==" },
  { value: "!=", label: "!=" },
];

export const acknowledgmentTypeOptions: SelectOption[] = [
  { value: "None", label: "None" },
  { value: "997", label: "997" },
  { value: "999", label: "999" },
];
