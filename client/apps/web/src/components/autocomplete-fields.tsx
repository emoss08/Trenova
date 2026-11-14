import type { FieldRequired } from "@trenova/shared/lib/field-rules";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { benefitPlanTypeLabel } from "@trenova/shared/lib/benefits";
import { describeShiftPattern } from "@trenova/shared/lib/scheduling";
import type {
  SelectOption as GraphQLSelectOption,
  GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import { capturePixelTypeLabel, isCapturePixelType, type CaptureRecordKind } from "@/lib/capture";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import {
  selectOptionMetaBoolean,
  selectOptionMetaEnum,
  selectOptionMetaNumber,
  selectOptionMetaString,
} from "@/lib/select-option-meta";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { SelectOption as StaticSelectOption } from "@trenova/shared/types/fields";
import {
  FUEL_CARD_PROVIDER_LABELS,
  fuelCardProviderSchema,
} from "@trenova/shared/types/fuel-ifta-enums";
import {
  TRAINING_DELIVERY_LABELS,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";
import type { ReactNode } from "react";
import type { Control, FieldPath, FieldValues, Path, RegisterOptions } from "react-hook-form";
import { Autocomplete, AutocompleteField } from "./fields/autocomplete/autocomplete";
import { FieldWrapper, type FieldLayout } from "./fields/field-components";
import { OptionStack } from "./fields/autocomplete/option-stack";
import { MultiSelectAutocompleteField } from "./fields/multi-select-field";
import { ColorOptionValue } from "./fields/select-components";
import { FuelCardMark } from "./fuel-card-mark";
import { SelectField } from "./fields/select-field";

type BaseAutocompleteFieldProps<TOption, TForm extends FieldValues> = {
  control: Control<TForm>;
  name: Path<TForm>;
  label?: string;
  required?: FieldRequired;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  description?: ReactNode;
  /** Where the label sits: above the control, or beside it in a settings row. */
  layout?: FieldLayout;
  clearable?: boolean;
  placeholder?: string;
  disabled?: boolean;
  extraSearchParams?: Record<string, string | string[]>;
  onOptionChange?: (option: TOption | null) => void;
  filterOption?: (option: TOption) => boolean;
  noResultsMessage?: string;
  initialLimit?: number;
  triggerClassName?: string;
};

type ControlledAutocompleteFieldProps<TOption> = {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  onOptionChange?: (option: TOption | null) => void;
  description?: string;
  graphql: GraphQLSelectOptionsConfig;
  renderOption: (option: TOption) => ReactNode;
  getOptionValue: (option: TOption) => string | number;
  getDisplayValue: (option: TOption) => ReactNode;
  placeholder?: string;
  disabled?: boolean;
  clearable?: boolean;
  extraSearchParams?: Record<string, string | string[]>;
  initialLimit?: number;
  noResultsMessage?: string;
};

type BaseMultiSelectAutocompleteFieldProps<
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _TOption,
  TForm extends FieldValues,
> = {
  name: FieldPath<TForm>;
  control: Control<TForm>;
  required?: FieldRequired;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  label?: string;
  description?: string;
  clearable?: boolean;
  placeholder?: string;
  extraSearchParams?: Record<string, string>;
  maxCount?: number;
  triggerClassName?: string;
};

type BaseStaticPermissionAutocompleteFieldProps<TForm extends FieldValues> = {
  name: Path<TForm>;
  control: Control<TForm>;
  required?: FieldRequired;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  label?: string;
  description?: string;
  placeholder?: string;
  disabled?: boolean;
  onValueChange?: (value: string) => void;
};

type PermissionResourceAutocompleteFieldProps<TForm extends FieldValues> =
  BaseStaticPermissionAutocompleteFieldProps<TForm> & {
    resources: ResourceDefinition[];
  };

type PermissionOperationAutocompleteFieldProps<TForm extends FieldValues> =
  BaseStaticPermissionAutocompleteFieldProps<TForm> & {
    operations: OperationDefinition[];
  };

function toPermissionSelectOptions(
  definitions: Array<{
    value: string;
    label: string;
    description?: string;
  }>,
): StaticSelectOption[] {
  return definitions.map((definition) => ({
    value: definition.value,
    label: definition.label,
    description: definition.description,
  }));
}

function selectOptionDateRange(option: GraphQLSelectOption) {
  const startDate = selectOptionMetaNumber(option, "startDate");
  const endDate = selectOptionMetaNumber(option, "endDate");
  if (startDate == null || endDate == null) return "";
  return formatRange(startDate, endDate);
}

const carrierSelectOptionsGraphQL = {
  resource: "CARRIER",
} satisfies GraphQLSelectOptionsConfig;

const customerSelectOptionsGraphQL = {
  resource: "CUSTOMER",
} satisfies GraphQLSelectOptionsConfig;

const usStateSelectOptionsGraphQL = {
  resource: "US_STATE",
} satisfies GraphQLSelectOptionsConfig;

const equipmentTypeSelectOptionsGraphQL = {
  resource: "EQUIPMENT_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const equipmentManufacturerSelectOptionsGraphQL = {
  resource: "EQUIPMENT_MANUFACTURER",
} satisfies GraphQLSelectOptionsConfig;

const tractorSelectOptionsGraphQL = {
  resource: "TRACTOR",
} satisfies GraphQLSelectOptionsConfig;

const trailerSelectOptionsGraphQL = {
  resource: "TRAILER",
} satisfies GraphQLSelectOptionsConfig;

const workerSelectOptionsGraphQL = {
  resource: "WORKER",
} satisfies GraphQLSelectOptionsConfig;

const shipmentSelectOptionsGraphQL = {
  resource: "SHIPMENT",
} satisfies GraphQLSelectOptionsConfig;

const orderSelectOptionsGraphQL = {
  resource: "ORDER",
} satisfies GraphQLSelectOptionsConfig;

const ediTransferSelectOptionsGraphQL = {
  resource: "EDI_TRANSFER",
} satisfies GraphQLSelectOptionsConfig;

const fuelIndexSelectOptionsGraphQL = {
  resource: "FUEL_INDEX",
} satisfies GraphQLSelectOptionsConfig;

const fiscalYearSelectOptionsGraphQL = {
  resource: "FISCAL_YEAR",
} satisfies GraphQLSelectOptionsConfig;

const fiscalPeriodSelectOptionsGraphQL = {
  resource: "FISCAL_PERIOD",
} satisfies GraphQLSelectOptionsConfig;

const glAccountSelectOptionsGraphQL = {
  resource: "GL_ACCOUNT",
} satisfies GraphQLSelectOptionsConfig;

const reversibleJournalEntrySelectOptionsGraphQL = {
  resource: "JOURNAL_ENTRY",
  filters: { reversible: true },
} satisfies GraphQLSelectOptionsConfig;

const fuelSurchargeProgramSelectOptionsGraphQL = {
  resource: "FUEL_SURCHARGE_PROGRAM",
} satisfies GraphQLSelectOptionsConfig;

const factoringCompanySelectOptionsGraphQL = {
  resource: "FACTORING_COMPANY",
} satisfies GraphQLSelectOptionsConfig;

const ediConnectionSelectOptionsGraphQL = {
  resource: "EDI_CONNECTION",
} satisfies GraphQLSelectOptionsConfig;

const locationSelectOptionsGraphQL = {
  resource: "LOCATION",
} satisfies GraphQLSelectOptionsConfig;

const rateZoneSelectOptionsGraphQL = {
  resource: "RATE_ZONE",
} satisfies GraphQLSelectOptionsConfig;

const fleetCodeSelectOptionsGraphQL = {
  resource: "FLEET_CODE",
} satisfies GraphQLSelectOptionsConfig;

const shipmentTypeSelectOptionsGraphQL = {
  resource: "SHIPMENT_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const serviceTypeSelectOptionsGraphQL = {
  resource: "SERVICE_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const locationCategorySelectOptionsGraphQL = {
  resource: "LOCATION_CATEGORY",
} satisfies GraphQLSelectOptionsConfig;

const distanceProfileSelectOptionsGraphQL = {
  resource: "DISTANCE_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const organizationSelectOptionsGraphQL = {
  resource: "ORGANIZATION",
} satisfies GraphQLSelectOptionsConfig;

const userSelectOptionsGraphQL = {
  resource: "USER",
} satisfies GraphQLSelectOptionsConfig;

const roleSelectOptionsGraphQL = {
  resource: "ROLE",
} satisfies GraphQLSelectOptionsConfig;

const rateMatrixSelectOptionsGraphQL = {
  resource: "RATE_MATRIX",
} satisfies GraphQLSelectOptionsConfig;

const rateAgreementSelectOptionsGraphQL = {
  resource: "RATE_AGREEMENT",
} satisfies GraphQLSelectOptionsConfig;

const accessorialChargeSelectOptionsGraphQL = {
  resource: "ACCESSORIAL_CHARGE",
} satisfies GraphQLSelectOptionsConfig;

const accountTypeSelectOptionsGraphQL = {
  resource: "ACCOUNT_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const commoditySelectOptionsGraphQL = {
  resource: "COMMODITY",
} satisfies GraphQLSelectOptionsConfig;

const documentTypeSelectOptionsGraphQL = {
  resource: "DOCUMENT_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const documentSelectOptionsGraphQL = {
  resource: "DOCUMENT",
} satisfies GraphQLSelectOptionsConfig;

const bankReceiptSourceSelectOptionsGraphQL = {
  resource: "BANK_RECEIPT_SOURCE",
} satisfies GraphQLSelectOptionsConfig;

const captureDeviceSelectOptionsGraphQL = {
  resource: "CAPTURE_DEVICE",
} satisfies GraphQLSelectOptionsConfig;

const captureProfileSelectOptionsGraphQL = {
  resource: "CAPTURE_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const detentionPolicySelectOptionsGraphQL = {
  resource: "DETENTION_POLICY",
} satisfies GraphQLSelectOptionsConfig;

const shiftTemplateSelectOptionsGraphQL = {
  resource: "SHIFT_TEMPLATE",
} satisfies GraphQLSelectOptionsConfig;

const workerPolicySelectOptionsGraphQL = {
  resource: "WORKER_POLICY",
} satisfies GraphQLSelectOptionsConfig;

const jobPositionSelectOptionsGraphQL = {
  resource: "JOB_POSITION",
} satisfies GraphQLSelectOptionsConfig;

const formulaTemplateSelectOptionsGraphQL = {
  resource: "FORMULA_TEMPLATE",
} satisfies GraphQLSelectOptionsConfig;

const hazardousMaterialSelectOptionsGraphQL = {
  resource: "HAZARDOUS_MATERIAL",
} satisfies GraphQLSelectOptionsConfig;

const serviceFailureReasonCodeSelectOptionsGraphQL = {
  resource: "SERVICE_FAILURE_REASON_CODE",
} satisfies GraphQLSelectOptionsConfig;

const ediCommunicationProfileSelectOptionsGraphQL = {
  resource: "EDI_COMMUNICATION_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const ediDocumentTypeSelectOptionsGraphQL = {
  resource: "EDI_DOCUMENT_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const ediTransactionSetSelectOptionsGraphQL = {
  resource: "EDI_TRANSACTION_SET",
} satisfies GraphQLSelectOptionsConfig;

const ediMappingProfileSelectOptionsGraphQL = {
  resource: "EDI_MAPPING_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const ediPartnerSelectOptionsGraphQL = {
  resource: "EDI_PARTNER",
} satisfies GraphQLSelectOptionsConfig;

const emailProfileSelectOptionsGraphQL = {
  resource: "EMAIL_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const aiProviderSelectOptionsGraphQL = {
  resource: "AI_PROVIDER",
} satisfies GraphQLSelectOptionsConfig;

/** A document's name, with its type when it has one: "BOL 4471.pdf · Bill of lading". */
function documentOptionLabel(option: GraphQLSelectOption) {
  return option.description ? `${option.label} · ${option.description}` : option.label;
}

function ControlledAutocompleteField<TOption>({
  label,
  value,
  onValueChange,
  onOptionChange,
  description,
  graphql,
  renderOption,
  getOptionValue,
  getDisplayValue,
  placeholder = "Select...",
  disabled,
  clearable = true,
  extraSearchParams,
  initialLimit = 20,
  noResultsMessage,
}: ControlledAutocompleteFieldProps<TOption>) {
  return (
    <FieldWrapper label={label} description={description}>
      <Autocomplete<TOption, FieldValues>
        graphql={graphql}
        value={value}
        onChange={(nextValue) => onValueChange(nextValue ? String(nextValue) : "")}
        onOptionChange={onOptionChange}
        label={label}
        renderOption={renderOption}
        getOptionValue={getOptionValue}
        getDisplayValue={getDisplayValue}
        placeholder={placeholder}
        disabled={!!disabled}
        clearable={clearable}
        extraSearchParams={extraSearchParams}
        initialLimit={initialLimit}
        noResultsMessage={noResultsMessage}
      />
    </FieldWrapper>
  );
}

function aiProviderDisplay(option: GraphQLSelectOption) {
  return option.description ? `${option.label} · ${option.description}` : option.label;
}

type AIProviderAutocompleteProps = {
  label: string;
  /** The chosen provider's id; empty for none. */
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerClassName?: string;
};

/**
 * The enabled providers that answer the assistant, searched by name or model. Clearing
 * it leaves the value empty rather than null, which is what a form that keeps "none" as
 * an empty id expects.
 */
export function AIProviderAutocomplete({
  label,
  value,
  onValueChange,
  placeholder,
  disabled,
  triggerClassName,
}: AIProviderAutocompleteProps) {
  return (
    <Autocomplete<GraphQLSelectOption, FieldValues>
      graphql={aiProviderSelectOptionsGraphQL}
      label={label}
      value={value || null}
      onChange={(nextValue: string | number | null | undefined) =>
        onValueChange(nextValue ? String(nextValue) : "")
      }
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={aiProviderDisplay}
      renderOption={(option) => (
        <OptionStack primary={option.label} secondary={option.description || ""} />
      )}
      placeholder={placeholder}
      disabled={!!disabled}
      clearable
      triggerClassName={triggerClassName}
    />
  );
}

export function RoleAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={roleSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      getOptionLabel={(option) => option.label || ""}
      {...props}
    />
  );
}

export function RoleSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={roleSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack primary={option.label || ""} secondary={option.description || ""} />
      )}
      {...props}
    />
  );
}

export function PermissionResourceAutocompleteField<T extends FieldValues>({
  resources,
  label = "Resource",
  placeholder = "Select resource",
  description = "Protected application resource evaluated by this access policy.",
  ...props
}: PermissionResourceAutocompleteFieldProps<T>) {
  const options = toPermissionSelectOptions(
    resources.map((resource) => ({
      value: resource.resource,
      label: resource.displayName,
      description: resource.description || resource.category,
    })),
  );

  return (
    <SelectField<T>
      label={label}
      placeholder={placeholder}
      description={description}
      options={options}
      {...props}
    />
  );
}

export function PermissionOperationAutocompleteField<T extends FieldValues>({
  operations,
  label = "Operation",
  placeholder = "Select operation",
  description = "Action on the selected resource that this policy allows or denies.",
  ...props
}: PermissionOperationAutocompleteFieldProps<T>) {
  const options = toPermissionSelectOptions(
    operations.map((operation) => ({
      value: operation.operation,
      label: operation.displayName || operation.operation,
      description: operation.description,
    })),
  );

  return (
    <SelectField<T>
      label={label}
      placeholder={placeholder}
      description={description}
      options={options}
      {...props}
    />
  );
}

export function UserAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={userSelectOptionsGraphQL}
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      {...props}
    />
  );
}

export function UserMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={userSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      {...props}
    />
  );
}

export function UsStateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={usStateSelectOptionsGraphQL}
      initialLimit={100}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      {...props}
    />
  );
}

export function EquipmentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={equipmentTypeSelectOptionsGraphQL}
      popoutLink="/equipment/configuration-files/equipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function EquipmentManufacturerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={equipmentManufacturerSelectOptionsGraphQL}
      popoutLink="/equipment/configuration-files/equipment-manufacturers"
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{t(option.label)}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function TractorAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={tractorSelectOptionsGraphQL}
      popoutLink="/equipment/tractors/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => option.label}
      {...props}
    />
  );
}

export function TrailerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={trailerSelectOptionsGraphQL}
      popoutLink="/equipment/trailers/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => option.label}
      {...props}
    />
  );
}

export function FleetCodeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fleetCodeSelectOptionsGraphQL}
      popoutLink="/dispatch/configuration-files/fleet-codes"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ShipmentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={shipmentTypeSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/shipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ServiceTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={serviceTypeSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/service-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function WorkerAutocompleteField<T extends FieldValues>({
  ownerOperatorsOnly,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  /**
   * Narrows the picker to owner-operators: Contractor-type workers, plus any
   * driver whose effective pay profile carries the OwnerOperator
   * classification.
   */
  ownerOperatorsOnly?: boolean;
}) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={
        ownerOperatorsOnly
          ? { ...workerSelectOptionsGraphQL, filters: { ownerOperatorsOnly: true } }
          : workerSelectOptionsGraphQL
      }
      popoutLink="/workers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {selectOptionMetaString(option, "fleetCode") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t("Fleet: {0}", selectOptionMetaString(option, "fleetCode"))}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ShipmentAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={shipmentSelectOptionsGraphQL}
      popoutLink="/shipment-management/shipments"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{t(option.label)}</span>
          {(selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status")) && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

type ShipmentAutocompleteProps = {
  label: string;
  /** The chosen shipment's id; empty for none. */
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  triggerClassName?: string;
};

/** A shipment searched by PRO, BOL or lane, outside a form. Clearing it leaves the value empty. */
export function ShipmentAutocomplete({
  label,
  value,
  onValueChange,
  placeholder,
  disabled,
  triggerClassName,
}: ShipmentAutocompleteProps) {
  return (
    <Autocomplete<GraphQLSelectOption, FieldValues>
      graphql={shipmentSelectOptionsGraphQL}
      label={label}
      value={value || null}
      onChange={(nextValue: string | number | null | undefined) =>
        onValueChange(nextValue ? String(nextValue) : "")
      }
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <OptionStack primary={option.label} secondary={option.description || ""} />
      )}
      placeholder={placeholder}
      disabled={!!disabled}
      clearable
      triggerClassName={triggerClassName}
    />
  );
}

export function OrderAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={orderSelectOptionsGraphQL}
      popoutLink="/shipment-management/orders"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{t(option.label)}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function CustomerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={customerSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/customers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const code = selectOptionMetaString(option, "code");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{code ? `${code} - ${option.label}` : option.label}</span>
          </div>
        );
      }}
      {...props}
    />
  );
}

export function CustomerMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={customerSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      getOptionLabel={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const code = selectOptionMetaString(option, "code");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{code ? `${code} - ${option.label}` : option.label}</span>
          </div>
        );
      }}
      {...props}
    />
  );
}

export function CarrierAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={carrierSelectOptionsGraphQL}
      popoutLink="/dispatch/carriers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const code = selectOptionMetaString(option, "code");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{code ? `${code} - ${option.label}` : option.label}</span>
          </div>
        );
      }}
      {...props}
    />
  );
}

export function EDICommunicationProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediCommunicationProfileSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack
          primary={option.label || ""}
          secondary={
            option.description ||
            `${selectOptionMetaString(option, "method")} · ${selectOptionMetaString(option, "status")}`.trim()
          }
        />
      )}
      {...props}
    />
  );
}

export function EDIConnectionAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediConnectionSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack primary={option.label || ""} secondary={option.description || ""} />
      )}
      {...props}
    />
  );
}

export function EDIMappingProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediMappingProfileSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack primary={option.label || ""} secondary={option.description || undefined} />
      )}
      {...props}
    />
  );
}

export function GLAccountAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={glAccountSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) =>
        option.description ? `${option.label} - ${option.description}` : option.label
      }
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

/** Posted journal entries that are neither reversals nor already reversed, newest first. */
export function ReversibleJournalEntryAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={reversibleJournalEntrySelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="font-mono">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {formatUnixDateMedium(selectOptionMetaNumber(option, "accountingDate"), {
              fallback: "",
            })}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function GLAccountMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={glAccountSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) =>
        option.description ? `${option.label} - ${option.description}` : option.label
      }
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      getOptionLabel={(option) =>
        option.description ? `${option.label} - ${option.description}` : option.label
      }
      {...props}
    />
  );
}

export function AccountTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={accountTypeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/account-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {selectOptionMetaString(option, "name") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "name")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function LocationAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={locationSelectOptionsGraphQL}
      popoutLink="/dispatch/locations"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const code = selectOptionMetaString(option, "code");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{code ? `${code} - ${option.label}` : option.label}</span>
            {option.description && (
              <span className="w-full truncate text-2xs text-muted-foreground">
                {t(option.description)}
              </span>
            )}
          </div>
        );
      }}
      {...props}
    />
  );
}

export function OrganizationAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={organizationSelectOptionsGraphQL}
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const scac = selectOptionMetaString(option, "scacCode");
        return scac ? `${scac} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const scac = selectOptionMetaString(option, "scacCode");
        const city = selectOptionMetaString(option, "city");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{scac ? `${scac} - ${option.label}` : option.label}</span>
            {city && <span className="w-full truncate text-2xs text-muted-foreground">{city}</span>}
          </div>
        );
      }}
      {...props}
    />
  );
}

export function EDIPartnerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediPartnerSelectOptionsGraphQL}
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack
          primary={option.label || ""}
          secondary={option.description || selectOptionMetaString(option, "code") || undefined}
        />
      )}
      {...props}
    />
  );
}

export function EDIDocumentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediDocumentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack primary={option.label || ""} secondary={option.description || undefined} />
      )}
      {...props}
    />
  );
}

export function EDITransactionSetAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ediTransactionSetSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack primary={option.label || ""} secondary={option.description || undefined} />
      )}
      {...props}
    />
  );
}

export function EmailProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={emailProfileSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <OptionStack
          primary={option.label || ""}
          secondary={
            option.description ||
            `${selectOptionMetaString(option, "senderEmail")} · ${selectOptionMetaString(option, "provider")}`.trim()
          }
        />
      )}
      {...props}
    />
  );
}

export function ControlledEDIPartnerAutocompleteField({
  label = "Partner",
  placeholder,
  ...props
}: Omit<
  ControlledAutocompleteFieldProps<GraphQLSelectOption>,
  "label" | "graphql" | "renderOption" | "getOptionValue" | "getDisplayValue" | "initialLimit"
> & {
  label?: string;
  placeholder?: string;
}) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={ediPartnerSelectOptionsGraphQL}
      initialLimit={50}
      placeholder={placeholder}
      renderOption={(option) => (
        <OptionStack
          primary={option.label || ""}
          secondary={option.description || selectOptionMetaString(option, "code") || undefined}
        />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label || ""}
      {...props}
    />
  );
}

type ControlledGraphQLAutocompleteFieldProps = Omit<
  ControlledAutocompleteFieldProps<GraphQLSelectOption>,
  "label" | "graphql" | "renderOption" | "getOptionValue" | "getDisplayValue"
> & {
  label?: string;
};

export function ControlledShipmentAutocompleteField({
  label = "Shipment",
  placeholder = "Search by Pro # or BOL...",
  ...props
}: ControlledGraphQLAutocompleteFieldProps) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={shipmentSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <OptionStack
          primary={option.label}
          secondary={
            selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status")
          }
        />
      )}
      {...props}
    />
  );
}

type CaptureRecordPicker = {
  graphql: GraphQLSelectOptionsConfig;
  placeholder: string;
  display: (option: GraphQLSelectOption) => string;
  detail: (option: GraphQLSelectOption) => string;
};

function codeAndName(option: GraphQLSelectOption): string {
  const code = selectOptionMetaString(option, "code");
  return code ? `${code} - ${option.label}` : option.label;
}

/*
 * Each kind of record a captured document can be filed onto, searched the way
 * the rest of the app searches it. Keyed by the kind's permission resource,
 * which is the name the capture API uses for it.
 */
const CAPTURE_RECORD_PICKERS: Record<CaptureRecordKind, CaptureRecordPicker> = {
  shipment: {
    graphql: shipmentSelectOptionsGraphQL,
    placeholder: "Search by Pro # or BOL...",
    display: (option) => option.label,
    detail: (option) =>
      selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status") || "",
  },
  worker: {
    graphql: workerSelectOptionsGraphQL,
    placeholder: "Search by name",
    display: (option) => option.label,
    detail: (option) => selectOptionMetaString(option, "fleetCode") || "",
  },
  tractor: {
    graphql: tractorSelectOptionsGraphQL,
    placeholder: "Search by unit number",
    display: (option) => option.label,
    detail: () => "",
  },
  trailer: {
    graphql: trailerSelectOptionsGraphQL,
    placeholder: "Search by unit number",
    display: (option) => option.label,
    detail: () => "",
  },
  customer: {
    graphql: customerSelectOptionsGraphQL,
    placeholder: "Search by code or name",
    display: codeAndName,
    detail: () => "",
  },
  carrier: {
    graphql: carrierSelectOptionsGraphQL,
    placeholder: "Search by code or name",
    display: codeAndName,
    detail: () => "",
  },
};

/** Picks a record of the given kind to file a captured document onto. */
export function ControlledCaptureRecordAutocompleteField({
  kind,
  label,
  placeholder,
  ...props
}: Omit<ControlledGraphQLAutocompleteFieldProps, "label"> & {
  kind: CaptureRecordKind;
  label: string;
}) {
  const t = useT();
  const picker = CAPTURE_RECORD_PICKERS[kind];

  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={picker.graphql}
      placeholder={placeholder ?? t(picker.placeholder)}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={picker.display}
      renderOption={(option) => (
        <OptionStack primary={picker.display(option)} secondary={picker.detail(option)} />
      )}
      {...props}
    />
  );
}

/**
 * Picks a document type outside a form. A category narrows it to the types
 * kept for that kind of record, as the Documents tab does.
 */
export function ControlledDocumentTypeAutocompleteField(
  props: Omit<
    ControlledAutocompleteFieldProps<GraphQLSelectOption>,
    "link" | "graphql" | "renderOption" | "getOptionValue" | "getDisplayValue" | "extraSearchParams"
  >,
) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      graphql={documentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {selectOptionMetaString(option, "name") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "name")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ControlledCarrierAutocompleteField({
  label = "Carrier",
  placeholder = "Search by code or name",
  ...props
}: ControlledGraphQLAutocompleteFieldProps) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={carrierSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? `${code} - ${option.label}` : option.label;
      }}
      renderOption={(option) => {
        const code = selectOptionMetaString(option, "code");
        return (
          <div className="flex size-full flex-col items-start">
            <span>{code ? `${code} - ${option.label}` : option.label}</span>
          </div>
        );
      }}
      {...props}
    />
  );
}

export function ControlledEDITransferAutocompleteField({
  label = "Transfer",
  placeholder = "Search transfers by BOL...",
  ...props
}: ControlledGraphQLAutocompleteFieldProps) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={ediTransferSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <OptionStack
          primary={option.label}
          secondary={
            selectOptionMetaString(option, "customerLabel") ||
            selectOptionMetaString(option, "partner") ||
            selectOptionMetaString(option, "status")
          }
        />
      )}
      {...props}
    />
  );
}

export function ControlledFormulaTemplateAutocompleteField({
  label = "Formula template",
  placeholder = "Search templates...",
  ...props
}: ControlledGraphQLAutocompleteFieldProps) {
  const t = useT();

  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={formulaTemplateSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function FormulaTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={formulaTemplateSelectOptionsGraphQL}
      popoutLink="billing/configuration-files/formula-templates"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function LocationCategoryAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={locationCategorySelectOptionsGraphQL}
      popoutLink="/dispatch/configuration-files/location-categories"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function DistanceProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={distanceProfileSelectOptionsGraphQL}
      popoutLink="/admin/distance-profiles"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full min-w-0 flex-col items-start">
          <span className="w-full truncate">{t(option.label)}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "routingType")} ·{" "}
            {selectOptionMetaString(option, "distanceUnits")}
            {selectOptionMetaBoolean(option, "isDefault") ? ` ${t("· Default")}` : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function DocumentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={documentTypeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/document-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {selectOptionMetaString(option, "name") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "name")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function DocumentTypeMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={documentTypeSelectOptionsGraphQL}
      nestedValues
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => selectOptionMetaString(option, "name") || option.label || ""}
      getOptionLabel={(option) => selectOptionMetaString(option, "name") || option.label || ""}
      renderOption={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      {...props}
    />
  );
}

export function DocumentMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={documentSelectOptionsGraphQL}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => documentOptionLabel(option)}
      getOptionLabel={(option) => documentOptionLabel(option)}
      renderBadge={(option) => (
        <span className="max-w-56 truncate">{documentOptionLabel(option)}</span>
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ServiceFailureReasonCodeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={serviceFailureReasonCodeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate font-medium">{t(option.label)}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function HazardousMaterialAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={hazardousMaterialSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/hazardous-materials"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {selectOptionMetaString(option, "class") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t("Class {0}", selectOptionMetaString(option, "class"))}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function CommodityAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={commoditySelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/commodities"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function FiscalYearAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fiscalYearSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => {
        const status = selectOptionMetaString(option, "status");
        const dateRange = selectOptionDateRange(option);
        return (
          <div className="flex size-full flex-col items-start">
            <span className="flex items-center gap-1.5">
              {t(option.label)}
              {selectOptionMetaBoolean(option, "isCurrent") && (
                <span className="inline-flex items-center rounded-md border border-success-border bg-success-subtle px-1 py-px text-2xs font-medium text-success-foreground">
                  {t("Current")}
                </span>
              )}
              {status && status !== "Open" && (
                <span className="rounded-md bg-muted px-1 py-0.5 text-2xs text-muted-foreground">
                  {status}
                </span>
              )}
            </span>
            {dateRange && (
              <span className="w-full truncate text-2xs text-muted-foreground">{dateRange}</span>
            )}
          </div>
        );
      }}
      {...props}
    />
  );
}

export function FiscalPeriodAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fiscalPeriodSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => {
        const status = selectOptionMetaString(option, "status");
        const periodType = selectOptionMetaString(option, "periodType");
        const dateRange = selectOptionDateRange(option);
        return (
          <div className="flex size-full flex-col items-start">
            <span className="flex items-center gap-1.5">
              {t(option.label)}
              {status && status !== "Open" && (
                <span className="rounded-md bg-muted px-1 py-0.5 text-2xs text-muted-foreground">
                  {status}
                </span>
              )}
            </span>
            <span className="w-full truncate text-2xs text-muted-foreground">
              {[periodType, dateRange].filter(Boolean).join(" · ")}
            </span>
          </div>
        );
      }}
      {...props}
    />
  );
}

function bankReceiptSourceFromText(text: string): GraphQLSelectOption {
  return { id: text, label: text, description: null, meta: null };
}

export function BatchSourceAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={bankReceiptSourceSelectOptionsGraphQL}
      preload
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => option.label}
      createOption={bankReceiptSourceFromText}
      {...props}
    />
  );
}

export function FuelIndexAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fuelIndexSelectOptionsGraphQL}
      popoutLink="/fuel/configuration-files/surcharge"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const region = selectOptionMetaString(option, "region");
        return region ? `${option.label} · ${region}` : option.label;
      }}
      renderOption={(option) => {
        const region = selectOptionMetaString(option, "region");
        const fuelType = selectOptionMetaString(option, "fuelType");
        return (
          <div className="flex size-full flex-col items-start">
            <span className="flex items-center gap-1.5">
              {t(option.label)}
              {region && (
                <span className="rounded-md bg-muted px-1 py-0.5 text-2xs text-muted-foreground">
                  {region}
                </span>
              )}
            </span>
            {option?.description && (
              <span className="w-full truncate text-2xs text-muted-foreground">
                {t(option.description)}
                {fuelType ? ` · ${fuelType}` : ""}
              </span>
            )}
          </div>
        );
      }}
      {...props}
    />
  );
}

export function FuelSurchargeProgramAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fuelSurchargeProgramSelectOptionsGraphQL}
      popoutLink="/fuel/configuration-files/surcharge"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function FactoringCompanyAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={factoringCompanySelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/factoring-companies"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function ShipmentTypeMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={shipmentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      getOptionLabel={(option) => option.label || ""}
      {...props}
    />
  );
}

export function ServiceTypeMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={serviceTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      getOptionLabel={(option) => option.label || ""}
      {...props}
    />
  );
}

export function CommodityMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={commoditySelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => option.label || ""}
      getOptionLabel={(option) => option.label || ""}
      {...props}
    />
  );
}

export function EquipmentTypeMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      graphql={equipmentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      getOptionLabel={(option) => option.label || ""}
      {...props}
    />
  );
}

export function DetentionPolicyAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={detentionPolicySelectOptionsGraphQL}
      popoutLink="/detention/configuration-files/detention-policies"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {selectOptionMetaString(option, "code") && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "code")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function AccessorialChargeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={accessorialChargeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/accessorial-charges"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function RateZoneAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={rateZoneSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/rate-zones"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {(option.description || selectOptionMetaString(option, "code")) && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description || selectOptionMetaString(option, "code")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function RateMatrixAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={rateMatrixSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/rate-matrices"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          {(option.description || selectOptionMetaString(option, "code")) && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description || selectOptionMetaString(option, "code")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function RateAgreementAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={rateAgreementSelectOptionsGraphQL}
      popoutLink="/billing/rate-agreements"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {option.description || selectOptionMetaString(option, "code")}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

/**
 * A shift pattern. The option carries the pattern's mask and cycle in its
 * meta so a picker can say what it is putting somebody on.
 */
/**
 * A job title. Driving titles are held by workers and front-office ones by
 * users, so the picker is told which roster it is filling and offers only
 * that half of the chart.
 */
export function JobPositionAutocompleteField<T extends FieldValues>({
  driving,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  driving: boolean;
}) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={{
        ...jobPositionSelectOptionsGraphQL,
        filters: driving ? { drivingOnly: true } : { nonDrivingOnly: true },
      }}
      popoutLink="/hr/org-structure"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "code")}
            {selectOptionMetaString(option, "department")
              ? ` · ${selectOptionMetaString(option, "department")}`
              : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function ShiftTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={shiftTemplateSelectOptionsGraphQL}
      popoutLink="/hr/scheduling"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "code")}
            {selectOptionMetaString(option, "daysOfWeek")
              ? ` · ${describeShiftPattern(
                  selectOptionMetaString(option, "daysOfWeek") ?? "",
                  Number(selectOptionMetaString(option, "cycleWeeks") ?? 1),
                )}`
              : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function WorkerPolicyAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={workerPolicySelectOptionsGraphQL}
      popoutLink="/hr/policies"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{t(option.label)}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {t(
              "{0} · v {1}",
              selectOptionMetaString(option, "code"),
              selectOptionMetaString(option, "versionLabel"),
            )}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

const fuelCardSelectOptionsGraphQL = {
  resource: "FUEL_CARD",
} satisfies GraphQLSelectOptionsConfig;

export function FuelCardAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={fuelCardSelectOptionsGraphQL}
      popoutLink="/fuel/configuration-files/fuel-cards"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => {
        const provider = selectOptionMetaEnum(option, "provider", fuelCardProviderSchema.options);

        return (
          <span className="flex min-w-0 items-center gap-2">
            {provider && <FuelCardMark provider={provider} size="sm" />}
            <span className="truncate">{option.label}</span>
          </span>
        );
      }}
      renderOption={(option) => {
        const provider = selectOptionMetaEnum(option, "provider", fuelCardProviderSchema.options);

        return (
          <div className="flex size-full min-w-0 items-center gap-2">
            {provider && <FuelCardMark provider={provider} />}
            <OptionStack
              primary={t(option.label)}
              secondary={
                provider
                  ? FUEL_CARD_PROVIDER_LABELS[provider]
                  : selectOptionMetaString(option, "provider")
              }
            />
          </div>
        );
      }}
      {...props}
    />
  );
}

const iftaFuelTypeSelectOptionsGraphQL = {
  resource: "IFTA_FUEL_TYPE",
} satisfies GraphQLSelectOptionsConfig;

export function IftaFuelTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={iftaFuelTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{t(option.label)}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {t(option.description)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

const payCodeSelectOptionsGraphQL = {
  resource: "PAY_CODE",
} satisfies GraphQLSelectOptionsConfig;

const payProfileSelectOptionsGraphQL = {
  resource: "PAY_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const workerCredentialTypeSelectOptionsGraphQL = {
  resource: "WORKER_CREDENTIAL_TYPE",
} satisfies GraphQLSelectOptionsConfig;

const trainingCourseSelectOptionsGraphQL = {
  resource: "TRAINING_COURSE",
} satisfies GraphQLSelectOptionsConfig;

const performanceReviewTemplateSelectOptionsGraphQL = {
  resource: "PERFORMANCE_REVIEW_TEMPLATE",
} satisfies GraphQLSelectOptionsConfig;

const ptoPolicySelectOptionsGraphQL = {
  resource: "PTO_POLICY",
} satisfies GraphQLSelectOptionsConfig;

const benefitPlanSelectOptionsGraphQL = {
  resource: "BENEFIT_PLAN",
} satisfies GraphQLSelectOptionsConfig;

const iftaJurisdictionSelectOptionsGraphQL = {
  resource: "IFTA_JURISDICTION",
} satisfies GraphQLSelectOptionsConfig;

export type PayCodeDirection = "Earning" | "Deduction";

/**
 * A settlement line is either an earning or a deduction, never both, so a form
 * that already knows which side it is on narrows the picker to that side. Left
 * open, the option's own direction is spelled out so the two cannot be mixed up.
 */
export function PayCodeAutocompleteField<T extends FieldValues>({
  direction,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  direction?: PayCodeDirection;
}) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={
        direction
          ? { ...payCodeSelectOptionsGraphQL, filters: { direction } }
          : payCodeSelectOptionsGraphQL
      }
      popoutLink="/payroll/pay-codes"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => payCodeOptionLabel(option, direction)}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{payCodeOptionLabel(option, direction)}</span>
          {selectOptionMetaString(option, "direction") && !direction && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {selectOptionMetaString(option, "direction")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

function payCodeOptionLabel(option: GraphQLSelectOption, direction?: PayCodeDirection) {
  const name = selectOptionMetaString(option, "name");
  const label = name ? `${option.label} — ${name}` : option.label;
  if (direction) return label;

  const optionDirection = selectOptionMetaString(option, "direction");
  return optionDirection ? `${label} (${optionDirection})` : label;
}

/**
 * The uncontrolled twin of PayCodeAutocompleteField, for the adjustment dialogs
 * that hold their state in useState rather than a react-hook-form control.
 */
export function ControlledPayCodeAutocompleteField({
  label = "Pay code",
  placeholder = "Search pay codes...",
  direction,
  ...props
}: ControlledGraphQLAutocompleteFieldProps & {
  direction?: PayCodeDirection;
}) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      graphql={
        direction
          ? { ...payCodeSelectOptionsGraphQL, filters: { direction } }
          : payCodeSelectOptionsGraphQL
      }
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => payCodeOptionLabel(option, direction)}
      renderOption={(option) => (
        <OptionStack
          primary={payCodeOptionLabel(option, direction)}
          secondary={direction ? undefined : selectOptionMetaString(option, "direction")}
        />
      )}
      {...props}
    />
  );
}

export function PayProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={payProfileSelectOptionsGraphQL}
      popoutLink="/payroll/pay-profiles"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => payProfileOptionLabel(option)}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{payProfileOptionLabel(option)}</span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

function payProfileOptionLabel(option: GraphQLSelectOption) {
  return selectOptionMetaString(option, "classification") === "OwnerOperator"
    ? `${option.label} (O-O)`
    : option.label;
}

export function WorkerCredentialTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={workerCredentialTypeSelectOptionsGraphQL}
      popoutLink="/hr/credential-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "code")}
            {option.description ? ` · ${option.description}` : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function TrainingCourseAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={trainingCourseSelectOptionsGraphQL}
      popoutLink="/hr/training-courses"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {trainingCourseOptionSummary(option)}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function trainingCourseOptionSummary(option: GraphQLSelectOption) {
  const delivery = selectOptionMetaString(option, "delivery") as TrainingDelivery;
  const parts = [TRAINING_DELIVERY_LABELS[delivery] ?? delivery];
  const minutes = selectOptionMetaNumber(option, "durationMinutes");
  if (minutes) parts.push(translate("{0} min", minutes));
  const passingScore = selectOptionMetaString(option, "passingScore");
  if (passingScore) parts.push(translate("pass ≥ {0}%", Number(passingScore).toFixed(0)));
  const validityMonths = selectOptionMetaNumber(option, "validityMonths");
  if (validityMonths) parts.push(translate("valid {0} mo", validityMonths));

  return parts.filter(Boolean).join(" · ");
}

export function PerformanceReviewTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={performanceReviewTemplateSelectOptionsGraphQL}
      popoutLink="/hr/review-templates"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {reviewTemplateOptionSummary(option)}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

function reviewTemplateOptionSummary(option: GraphQLSelectOption) {
  const itemCount = selectOptionMetaNumber(option, "itemCount") ?? 0;
  const cadenceMonths = selectOptionMetaNumber(option, "cadenceMonths");
  return cadenceMonths
    ? translate(
        "{0, plural, one {# item} other {# items}} · {1, plural, one {every # month} other {every # months}}",
        itemCount,
        cadenceMonths,
      )
    : translate("{0, plural, one {# item} other {# items}}", itemCount);
}

export function PtoPolicyAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={ptoPolicySelectOptionsGraphQL}
      popoutLink="/hr/pto-policies"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "code")}
            {selectOptionMetaBoolean(option, "isDefault") ? " · Default" : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function BenefitPlanAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={benefitPlanSelectOptionsGraphQL}
      popoutLink="/hr/benefits"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => benefitPlanOptionLabel(option)}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{benefitPlanOptionLabel(option)}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {benefitPlanTypeLabel(selectOptionMetaString(option, "planType"))}
            {selectOptionMetaString(option, "carrier")
              ? ` · ${selectOptionMetaString(option, "carrier")}`
              : ""}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

function benefitPlanOptionLabel(option: GraphQLSelectOption) {
  const planYear = selectOptionMetaNumber(option, "planYear");
  return planYear ? `${option.label} (${planYear})` : option.label;
}

/**
 * Non-member jurisdictions stay selectable — mileage is still recorded in them,
 * it just does not land on the IFTA return — so the option says which it is
 * rather than the list quietly leaving them out.
 */
export function IftaJurisdictionAutocompleteField<T extends FieldValues>({
  membersOnly,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  membersOnly?: boolean;
}) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={
        membersOnly
          ? { ...iftaJurisdictionSelectOptionsGraphQL, filters: { membersOnly: true } }
          : iftaJurisdictionSelectOptionsGraphQL
      }
      initialLimit={100}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {selectOptionMetaString(option, "countryCode")}
            {selectOptionMetaBoolean(option, "isIftaMember") ? "" : " · Not an IFTA member"}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

/**
 * One of the person's own paired computers. Only their active computers are
 * offered; the option says whether each is connected right now.
 */
export function CaptureDeviceAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();
  const status = (option: GraphQLSelectOption) =>
    selectOptionMetaBoolean(option, "isOnline") ? (
      <Badge variant="success" appearance="outline">
        {t("Online")}
      </Badge>
    ) : (
      <Badge variant="neutral" appearance="outline">
        {t("Offline")}
      </Badge>
    );

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={captureDeviceSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate">{option.label}</span>
          {status(option)}
        </span>
      )}
      renderOption={(option) => (
        <div className="flex size-full min-w-0 flex-col items-start">
          <span className="flex w-full min-w-0 items-center gap-2">
            <span className="truncate">{option.label}</span>
            {status(option)}
          </span>
          {option.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

/**
 * A scan profile the person may scan with. An empty value means the
 * organization's default profile.
 */
export function CaptureProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  const t = useT();
  const summary = (option: GraphQLSelectOption) => {
    const dpi = selectOptionMetaNumber(option, "dpi");
    const pixelType = selectOptionMetaString(option, "pixelType");
    const parts = [
      dpi === null ? null : t("{0} dpi", dpi),
      isCapturePixelType(pixelType) ? capturePixelTypeLabel(t, pixelType) : null,
      selectOptionMetaBoolean(option, "duplex") ? t("Both sides") : null,
    ].filter((part): part is string => part !== null);
    return parts.join(" · ");
  };

  return (
    <AutocompleteField<GraphQLSelectOption, T>
      graphql={captureProfileSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full min-w-0 flex-col items-start">
          <span className="flex w-full min-w-0 items-center gap-2">
            <span className="truncate">{option.label}</span>
            {selectOptionMetaBoolean(option, "isDefault") && (
              <Badge variant="neutral" appearance="outline">
                {t("Default")}
              </Badge>
            )}
          </span>
          {summary(option) !== "" && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {summary(option)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}
