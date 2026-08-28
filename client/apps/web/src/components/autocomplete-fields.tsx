import type {
  SelectOption as GraphQLSelectOption,
  GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import { formatRange } from "@trenova/shared/lib/date";
import type { BatchSourceOption } from "@/types/bank-receipt-batch";
import type { Document } from "@trenova/shared/types/document";
import type { SelectOption as StaticSelectOption } from "@trenova/shared/types/fields";
import type { API_ENDPOINTS, SELECT_OPTIONS_ENDPOINTS } from "@trenova/shared/types/server";
import type { ReactNode } from "react";

// Ensure server endpoint types remain referenced for cross-package type checks
export type _ServerEndpointTypes = API_ENDPOINTS | SELECT_OPTIONS_ENDPOINTS;
import type { Control, FieldPath, FieldValues, Path, RegisterOptions } from "react-hook-form";
import { Autocomplete, AutocompleteField } from "./fields/autocomplete/autocomplete";
import { FieldWrapper } from "./fields/field-components";
import { MultiSelectAutocompleteField } from "./fields/multi-select-field";
import { ColorOptionValue } from "./fields/select-components";
import { SelectField } from "./fields/select-field";

type BaseAutocompleteFieldProps<TOption, TForm extends FieldValues> = {
  control: Control<TForm>;
  name: Path<TForm>;
  label?: string;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  description?: string;
  clearable?: boolean;
  placeholder?: string;
  disabled?: boolean;
  extraSearchParams?: Record<string, string | string[]>;
  selectedValueLink?: API_ENDPOINTS;
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
  link: SELECT_OPTIONS_ENDPOINTS;
  selectedValueLink?: API_ENDPOINTS;
  graphql?: GraphQLSelectOptionsConfig;
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

function selectOptionMetaString(option: GraphQLSelectOption, key: string) {
  const value = option.meta?.[key];
  return typeof value === "string" ? value : "";
}

function selectOptionMetaNumber(option: GraphQLSelectOption, key: string) {
  const value = option.meta?.[key];
  return typeof value === "number" ? value : null;
}

function selectOptionMetaBoolean(option: GraphQLSelectOption, key: string) {
  return option.meta?.[key] === true;
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

const fuelSurchargeProgramSelectOptionsGraphQL = {
  resource: "FUEL_SURCHARGE_PROGRAM",
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

const detentionPolicySelectOptionsGraphQL = {
  resource: "DETENTION_POLICY",
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

const ediMappingProfileSelectOptionsGraphQL = {
  resource: "EDI_MAPPING_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const ediPartnerSelectOptionsGraphQL = {
  resource: "EDI_PARTNER",
} satisfies GraphQLSelectOptionsConfig;

const ediPartnerDocumentProfileSelectOptionsGraphQL = {
  resource: "EDI_PARTNER_DOCUMENT_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

const ediTemplateSelectOptionsGraphQL = {
  resource: "EDI_TEMPLATE",
} satisfies GraphQLSelectOptionsConfig;

const emailProfileSelectOptionsGraphQL = {
  resource: "EMAIL_PROFILE",
} satisfies GraphQLSelectOptionsConfig;

function getDocumentLabel(option: Document) {
  const documentTypeLabel = option.documentType?.name?.trim();
  const fileName = option.originalName?.trim() || option.fileName?.trim() || option.id;

  return documentTypeLabel ? `${fileName} · ${documentTypeLabel}` : fileName;
}

function ControlledAutocompleteField<TOption>({
  label,
  value,
  onValueChange,
  onOptionChange,
  description,
  link,
  selectedValueLink,
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
        link={link}
        selectedValueLink={selectedValueLink}
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

function EDIOptionStack({ primary, secondary }: { primary: ReactNode; secondary?: ReactNode }) {
  return (
    <div className="flex size-full min-w-0 flex-col items-start pr-4">
      <span className="w-full truncate">{primary}</span>
      {secondary ? (
        <span className="text-2xs text-muted-foreground w-full truncate">{secondary}</span>
      ) : null}
    </div>
  );
}

export function RoleAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      link="/roles/select-options/"
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
      link="/roles/select-options/"
      graphql={roleSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || ""} />
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
      link="/users/select-options/"
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
      link="/users/select-options/"
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
      link="/us-states/select-options/"
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
      link="/equipment-types/select-options/"
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
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/equipment-manufacturers/select-options/"
      graphql={equipmentManufacturerSelectOptionsGraphQL}
      popoutLink="/equipment/configuration-files/equipment-manufacturers"
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          {option.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
      link="/tractors/select-options/"
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
      link="/trailers/select-options/"
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
      link="/fleet-codes/select-options/"
      graphql={fleetCodeSelectOptionsGraphQL}
      popoutLink="/dispatch/configuration-files/fleet-codes"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
      link="/shipment-types/select-options/"
      graphql={shipmentTypeSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/shipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
      link="/service-types/select-options/"
      graphql={serviceTypeSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/service-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/workers/select-options/"
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
          <span>{option.label}</span>
          {selectOptionMetaString(option, "fleetCode") && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              Fleet: {selectOptionMetaString(option, "fleetCode")}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/shipments/select-options/"
      graphql={shipmentSelectOptionsGraphQL}
      popoutLink="/shipment-management/shipments"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          {(selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status")) && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status")}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function OrderAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/orders/select-options/"
      graphql={orderSelectOptionsGraphQL}
      popoutLink="/shipment-management/orders"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          {option.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
      link="/customers/select-options/"
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

export function CarrierAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/carriers/select-options/"
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
      link="/edi/communication-profiles/select-options/"
      graphql={ediCommunicationProfileSelectOptionsGraphQL}
      selectedValueLink="/edi/communication-profiles/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack
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
      link="/edi/connections/select-options/"
      graphql={ediConnectionSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack
          primary={option.label || ""}
          secondary={option.description || ""}
        />
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
      link="/edi/mapping-profiles/select-options/"
      graphql={ediMappingProfileSelectOptionsGraphQL}
      selectedValueLink="/edi/mapping-profiles/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack
          primary={option.label || ""}
          secondary={option.description || undefined}
        />
      )}
      {...props}
    />
  );
}

export function GLAccountAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/gl-accounts/select-options/"
      graphql={glAccountSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) =>
        option.description ? `${option.label} - ${option.description}` : option.label
      }
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function GLAccountMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <MultiSelectAutocompleteField<GraphQLSelectOption, T>
      link="/gl-accounts/select-options/"
      graphql={glAccountSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) =>
        option.description ? `${option.label} - ${option.description}` : option.label
      }
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
      link="/account-types/select-options/"
      graphql={accountTypeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/account-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {selectOptionMetaString(option, "name") && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/locations/select-options/"
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
              <span className="text-2xs text-muted-foreground w-full truncate">
                {option.description}
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
      link="/organizations/select-options/"
      graphql={organizationSelectOptionsGraphQL}
      selectedValueLink="/organizations/"
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
            {city && (
              <span className="text-2xs text-muted-foreground w-full truncate">{city}</span>
            )}
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
      link="/edi/partners/select-options/"
      graphql={ediPartnerSelectOptionsGraphQL}
      selectedValueLink="/edi/partners/"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || selectOptionMetaString(option, "code") || undefined} />
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
      link="/edi/catalog/document-types/select-options/"
      graphql={ediDocumentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || undefined} />
      )}
      {...props}
    />
  );
}

export function EDITemplateAutocompleteField<T extends FieldValues>({
  transactionSet,
  direction,
  extraSearchParams,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  transactionSet?: string;
  direction?: string;
}) {
  const graphqlFilters = {
    ...(transactionSet ? { transactionSet } : {}),
    ...(direction ? { direction } : {}),
  };
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/edi/templates/select-options/"
      graphql={
        Object.keys(graphqlFilters).length > 0
          ? { ...ediTemplateSelectOptionsGraphQL, filters: graphqlFilters }
          : ediTemplateSelectOptionsGraphQL
      }
      selectedValueLink="/edi/templates/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || selectOptionMetaString(option, "status") || undefined} />
      )}
      {...props}
    />
  );
}

export function EDIDocumentProfileAutocompleteField<T extends FieldValues>({
  partnerId,
  transactionSet,
  direction,
  extraSearchParams,
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T> & {
  partnerId?: string;
  transactionSet?: string;
  direction?: string;
}) {
  const graphqlFilters: Record<string, unknown> = {
    ...(partnerId ? { partnerId } : {}),
    ...(transactionSet ? { transactionSet } : {}),
    ...(direction ? { direction } : {}),
  };
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/edi/document-profiles/select-options/"
      graphql={
        Object.keys(graphqlFilters).length > 0
          ? { ...ediPartnerDocumentProfileSelectOptionsGraphQL, filters: graphqlFilters }
          : ediPartnerDocumentProfileSelectOptionsGraphQL
      }
      selectedValueLink="/edi/document-profiles/"
      extraSearchParams={{
        ...(partnerId ? { partnerId } : {}),
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || undefined} />
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
      link="/email-profiles/select-options/"
      graphql={emailProfileSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <EDIOptionStack
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
  | "label"
  | "link"
  | "graphql"
  | "renderOption"
  | "getOptionValue"
  | "getDisplayValue"
  | "selectedValueLink"
  | "initialLimit"
> & {
  label?: string;
  placeholder?: string;
}) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      link="/edi/partners/select-options/"
      graphql={ediPartnerSelectOptionsGraphQL}
      selectedValueLink="/edi/partners/"
      initialLimit={50}
      placeholder={placeholder}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || selectOptionMetaString(option, "code") || undefined} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label || ""}
      {...props}
    />
  );
}

type ControlledGraphQLAutocompleteFieldProps = Omit<
  ControlledAutocompleteFieldProps<GraphQLSelectOption>,
  | "label"
  | "link"
  | "graphql"
  | "renderOption"
  | "getOptionValue"
  | "getDisplayValue"
  | "selectedValueLink"
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
      link="/shipments/select-options/"
      graphql={shipmentSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <EDIOptionStack
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

export function ControlledEDITransferAutocompleteField({
  label = "Transfer",
  placeholder = "Search transfers by BOL...",
  ...props
}: ControlledGraphQLAutocompleteFieldProps) {
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label={label}
      link="/edi/transfers/select-options/"
      graphql={ediTransferSelectOptionsGraphQL}
      placeholder={placeholder}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <EDIOptionStack
          primary={option.label}
          secondary={
            selectOptionMetaString(option, "customerLabel") ||
            selectOptionMetaString(option, "sourcePartner") ||
            selectOptionMetaString(option, "status")
          }
        />
      )}
      {...props}
    />
  );
}

export function ControlledEDITemplateAutocompleteField({
  transactionSet,
  direction,
  extraSearchParams,
  ...props
}: Omit<
  ControlledAutocompleteFieldProps<GraphQLSelectOption>,
  | "label"
  | "link"
  | "graphql"
  | "renderOption"
  | "getOptionValue"
  | "getDisplayValue"
  | "selectedValueLink"
  | "extraSearchParams"
> & {
  transactionSet?: string;
  direction?: string;
  extraSearchParams?: Record<string, string | string[]>;
}) {
  const graphqlFilters: Record<string, unknown> = {
    ...(transactionSet ? { transactionSet } : {}),
    ...(direction ? { direction } : {}),
  };
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label="Template"
      link="/edi/templates/select-options/"
      graphql={
        Object.keys(graphqlFilters).length > 0
          ? { ...ediTemplateSelectOptionsGraphQL, filters: graphqlFilters }
          : ediTemplateSelectOptionsGraphQL
      }
      selectedValueLink="/edi/templates/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || selectOptionMetaString(option, "status") || undefined} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label || ""}
      {...props}
    />
  );
}

export function ControlledEDIDocumentProfileAutocompleteField({
  partnerId,
  transactionSet,
  direction,
  extraSearchParams,
  ...props
}: Omit<
  ControlledAutocompleteFieldProps<GraphQLSelectOption>,
  | "label"
  | "link"
  | "graphql"
  | "renderOption"
  | "getOptionValue"
  | "getDisplayValue"
  | "selectedValueLink"
  | "extraSearchParams"
> & {
  partnerId?: string;
  transactionSet?: string;
  direction?: string;
  extraSearchParams?: Record<string, string | string[]>;
}) {
  const graphqlFilters: Record<string, unknown> = {
    ...(partnerId ? { partnerId } : {}),
    ...(transactionSet ? { transactionSet } : {}),
    ...(direction ? { direction } : {}),
  };
  return (
    <ControlledAutocompleteField<GraphQLSelectOption>
      label="Document Profile"
      link="/edi/document-profiles/select-options/"
      graphql={
        Object.keys(graphqlFilters).length > 0
          ? { ...ediPartnerDocumentProfileSelectOptionsGraphQL, filters: graphqlFilters }
          : ediPartnerDocumentProfileSelectOptionsGraphQL
      }
      selectedValueLink="/edi/document-profiles/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...(partnerId ? { partnerId } : {}),
        ...extraSearchParams,
      }}
      renderOption={(option) => (
        <EDIOptionStack primary={option.label || ""} secondary={option.description || undefined} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.label || ""}
      {...props}
    />
  );
}

export function FormulaTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/formula-templates/select-options/"
      graphql={formulaTemplateSelectOptionsGraphQL}
      popoutLink="billing/configuration-files/formula-templates"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
      link="/location-categories/select-options/"
      graphql={locationCategorySelectOptionsGraphQL}
      popoutLink="/dispatch/configuration-files/location-categories"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/distance-profiles/select-options/"
      graphql={distanceProfileSelectOptionsGraphQL}
      popoutLink="/admin/distance-profiles"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full min-w-0 flex-col items-start">
          <span className="w-full truncate">{option.label}</span>
          <span className="text-2xs text-muted-foreground w-full truncate">
            {selectOptionMetaString(option, "routingType")} ·{" "}
            {selectOptionMetaString(option, "distanceUnits")}
            {selectOptionMetaBoolean(option, "isDefault") ? " · Default" : ""}
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
      link="/document-types/select-options/"
      graphql={documentTypeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/document-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue
            color={selectOptionMetaString(option, "color")}
            value={option.label}
          />
          {selectOptionMetaString(option, "name") && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
      link="/document-types/select-options/"
      graphql={documentTypeSelectOptionsGraphQL}
      nestedValues
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => selectOptionMetaString(option, "name") || option.label || ""}
      getOptionLabel={(option) => selectOptionMetaString(option, "name") || option.label || ""}
      renderOption={(option) => (
        <ColorOptionValue
          color={selectOptionMetaString(option, "color")}
          value={option.label}
        />
      )}
      {...props}
    />
  );
}

export function DocumentMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<Document, T>) {
  return (
    <MultiSelectAutocompleteField<Document, T>
      link="/documents/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => getDocumentLabel(option)}
      getOptionLabel={(option) => getDocumentLabel(option)}
      renderBadge={(option) => (
        <span className="max-w-56 truncate">{getDocumentLabel(option)}</span>
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate">{option.originalName || option.fileName}</span>
          <span className="text-2xs text-muted-foreground w-full truncate">
            {option.documentType?.name || option.resourceType}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function ServiceFailureReasonCodeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/service-failure-reason-codes/select-options/"
      graphql={serviceFailureReasonCodeSelectOptionsGraphQL}
      selectedValueLink="/service-failure-reason-codes/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="w-full truncate font-medium">{option.label}</span>
          {option.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/hazardous-materials/select-options/"
      graphql={hazardousMaterialSelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/hazardous-materials"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {selectOptionMetaString(option, "class") && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              Class {selectOptionMetaString(option, "class")}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/commodities/select-options/"
      graphql={commoditySelectOptionsGraphQL}
      popoutLink="/shipment-management/configuration-files/commodities"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/fiscal-years/"
      graphql={fiscalYearSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => {
        const status = selectOptionMetaString(option, "status");
        const dateRange = selectOptionDateRange(option);
        return (
          <div className="flex size-full flex-col items-start">
            <span className="flex items-center gap-1.5">
              {option.label}
              {selectOptionMetaBoolean(option, "isCurrent") && (
                <span className="inline-flex items-center rounded border border-green-600/30 bg-green-600/20 px-1 py-px text-[10px] font-medium text-green-700 dark:text-green-400">
                  Current
                </span>
              )}
              {status && status !== "Open" && (
                <span className="bg-muted text-2xs text-muted-foreground rounded px-1 py-0.5">
                  {status}
                </span>
              )}
            </span>
            {dateRange && (
              <span className="text-2xs text-muted-foreground w-full truncate">{dateRange}</span>
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/fiscal-periods/"
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
              {option.label}
              {status && status !== "Open" && (
                <span className="bg-muted text-2xs text-muted-foreground rounded px-1 py-0.5">
                  {status}
                </span>
              )}
            </span>
            <span className="text-2xs text-muted-foreground w-full truncate">
              {[periodType, dateRange].filter(Boolean).join(" · ")}
            </span>
          </div>
        );
      }}
      {...props}
    />
  );
}

export function BatchSourceAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<BatchSourceOption, T>) {
  return (
    <AutocompleteField<BatchSourceOption, T>
      link="/accounting/bank-receipt-batches/select-options/sources/"
      preload
      getOptionValue={(option) => option.value}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => option.label}
      {...props}
    />
  );
}

export function FuelIndexAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GraphQLSelectOption, T>) {
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/fuel-indices/select-options/"
      graphql={fuelIndexSelectOptionsGraphQL}
      popoutLink="/billing/fuel-management"
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
              {option.label}
              {region && (
                <span className="bg-muted text-2xs text-muted-foreground rounded px-1 py-0.5">
                  {region}
                </span>
              )}
            </span>
            {option?.description && (
              <span className="text-2xs text-muted-foreground w-full truncate">
                {option.description}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/fuel-surcharge-programs/select-options/"
      graphql={fuelSurchargeProgramSelectOptionsGraphQL}
      popoutLink="/billing/fuel-management"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option?.description}
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
      link="/shipment-types/select-options/"
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
      link="/service-types/select-options/"
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
      link="/commodities/select-options/"
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
      link="/equipment-types/select-options/"
      graphql={equipmentTypeSelectOptionsGraphQL}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label || ""}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={selectOptionMetaString(option, "color")} value={option.label} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/detention-policies/select-options/"
      graphql={detentionPolicySelectOptionsGraphQL}
      popoutLink="/detention/configuration-files/detention-policies"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {selectOptionMetaString(option, "code") && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/accessorial-charges/select-options/"
      graphql={accessorialChargeSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/accessorial-charges"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {option?.description && (
            <span className="text-2xs text-muted-foreground w-full truncate">
              {option.description}
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/rate-zones/select-options/"
      graphql={rateZoneSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/rate-zones"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {(option.description || selectOptionMetaString(option, "code")) && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/rate-matrices/select-options/"
      graphql={rateMatrixSelectOptionsGraphQL}
      popoutLink="/billing/configuration-files/rate-matrices"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          {(option.description || selectOptionMetaString(option, "code")) && (
            <span className="text-2xs text-muted-foreground w-full truncate">
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
  return (
    <AutocompleteField<GraphQLSelectOption, T>
      link="/rate-agreements/select-options/"
      graphql={rateAgreementSelectOptionsGraphQL}
      popoutLink="/billing/rate-agreements"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.label}</span>
          <span className="text-2xs text-muted-foreground w-full truncate">
            {option.description || selectOptionMetaString(option, "code")}
          </span>
        </div>
      )}
      {...props}
    />
  );
}
