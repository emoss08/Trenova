import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import { formatLocation } from "@/lib/utils";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import type { BatchSourceOption } from "@/types/bank-receipt-batch";
import type { FiscalPeriod } from "@/types/fiscal-period";
import type { FiscalYear } from "@/types/fiscal-year";
import type { AccountType } from "@/types/account-type";
import type { Commodity } from "@/types/commodity";
import type { Customer } from "@/types/customer";
import type { Document } from "@/types/document";
import type { DocumentType } from "@/types/document-type";
import type {
  EDICommunicationProfile,
  EDIDocumentType,
  EDIMappingProfile,
  EDIPartner,
  EDIPartnerDocumentProfile,
  EDITemplate,
} from "@/types/edi";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import type { EquipmentType } from "@/types/equipment-type";
import type { FleetCode } from "@/types/fleet-code";
import type { FormulaTemplate } from "@/types/formula-template";
import type { GLAccount } from "@/types/gl-account";
import type { HazardousMaterial } from "@/types/hazardous-material";
import type { Location } from "@/types/location";
import type { LocationCategory } from "@/types/location-category";
import type { OrganizationSelectOption } from "@/types/organization";
import type { Role } from "@/types/role";
import type { API_ENDPOINTS, SELECT_OPTIONS_ENDPOINTS } from "@/types/server";
import type { ServiceType } from "@/types/service-type";
import type { ShipmentType } from "@/types/shipment-type";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";
import type { UsState } from "@/types/us-state";
import type { User } from "@/types/user";
import type { Worker } from "@/types/worker";
import type { ReactNode } from "react";
import type { Control, FieldPath, FieldValues, Path, RegisterOptions } from "react-hook-form";
import type { SelectOption } from "@/types/fields";
import { Autocomplete, AutocompleteField } from "./fields/autocomplete/autocomplete";
import { FieldWrapper } from "./fields/field-components";
import { MultiSelectAutocompleteField } from "./fields/multi-select-field";
import { ColorOptionValue } from "./fields/select-components";
import { SelectField } from "./fields/select-field";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./ui/tooltip";

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
};

type ControlledAutocompleteFieldProps<TOption> = {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  onOptionChange?: (option: TOption | null) => void;
  description?: string;
  link: SELECT_OPTIONS_ENDPOINTS;
  selectedValueLink?: API_ENDPOINTS;
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
): SelectOption[] {
  return definitions.map((definition) => ({
    value: definition.value,
    label: definition.label,
    description: definition.description,
  }));
}

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
        <span className="w-full truncate text-2xs text-muted-foreground">{secondary}</span>
      ) : null}
    </div>
  );
}

export function RoleAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<Role, T>) {
  return (
    <MultiSelectAutocompleteField<Role, T>
      link="/roles/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      renderOption={(option) => option.name || ""}
      getOptionLabel={(option) => option.name || ""}
      {...props}
    />
  );
}

export function RoleSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<Role, T>) {
  return (
    <AutocompleteField<Role, T>
      link="/roles/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      renderOption={(option) => (
        <EDIOptionStack primary={option.name || ""} secondary={option.description} />
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
}: BaseAutocompleteFieldProps<User, T>) {
  return (
    <AutocompleteField<User, T>
      link="/users/select-options/"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      renderOption={(option) => option.name || ""}
      {...props}
    />
  );
}

export function UserMultiSelectAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<User, T>) {
  return (
    <MultiSelectAutocompleteField<User, T>
      link="/users/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      renderOption={(option) => option.name || ""}
      {...props}
    />
  );
}

export function UsStateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<UsState, T>) {
  return (
    <AutocompleteField<UsState, T>
      link="/us-states/select-options/"
      initialLimit={100}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      renderOption={(option) => option.name || ""}
      {...props}
    />
  );
}

export function EquipmentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EquipmentType, T>) {
  return (
    <AutocompleteField<EquipmentType, T>
      link="/equipment-types/select-options/"
      popoutLink="/equipment/configuration-files/equipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => <ColorOptionValue color={option.color} value={option.code} />}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color} value={option.code} />
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
}: BaseAutocompleteFieldProps<EquipmentManufacturer, T>) {
  return (
    <AutocompleteField<EquipmentManufacturer, T>
      link="/equipment-manufacturers/select-options/"
      popoutLink="/equipment/configuration-files/equipment-manufacturers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => option.name}
      {...props}
    />
  );
}

export function TractorAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<Tractor, T>) {
  return (
    <AutocompleteField<Tractor, T>
      link="/tractors/select-options/"
      popoutLink="/equipment/tractors/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => option.code}
      {...props}
    />
  );
}

export function TrailerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<Trailer, T>) {
  return (
    <AutocompleteField<Trailer, T>
      link="/trailers/select-options/"
      popoutLink="/equipment/trailers/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => option.code}
      {...props}
    />
  );
}

export function FleetCodeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FleetCode, T>) {
  return (
    <AutocompleteField<FleetCode, T>
      link="/fleet-codes/select-options/"
      popoutLink="/dispatch/configuration-files/fleet-codes"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => <ColorOptionValue color={option.color} value={option.code} />}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color} value={option.code} />
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
}: BaseAutocompleteFieldProps<ShipmentType, T>) {
  return (
    <AutocompleteField<ShipmentType, T>
      link="/shipment-types/select-options/"
      popoutLink="/shipment-management/configuration-files/shipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => <ColorOptionValue color={option.color} value={option.code} />}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color} value={option.code} />
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
}: BaseAutocompleteFieldProps<ServiceType, T>) {
  return (
    <AutocompleteField<ServiceType, T>
      link="/service-types/select-options/"
      popoutLink="/shipment-management/configuration-files/service-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => <ColorOptionValue color={option.color} value={option.code} />}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color} value={option.code} />
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
  ...props
}: BaseAutocompleteFieldProps<Worker, T>) {
  return (
    <AutocompleteField<Worker, T>
      link="/workers/select-options/"
      popoutLink="/workers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.wholeName || `${option.firstName} ${option.lastName}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.wholeName || `${option.firstName} ${option.lastName}`}</span>
          {option?.fleetCode?.code && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              Fleet: {option.fleetCode.code}
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
}: BaseAutocompleteFieldProps<Customer, T>) {
  return (
    <AutocompleteField<Customer, T>
      link="/customers/select-options/"
      popoutLink="/billing/configuration-files/customers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>
            {option.code} - {option.name}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function EDICommunicationProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EDICommunicationProfile, T>) {
  return (
    <AutocompleteField<EDICommunicationProfile, T>
      link="/edi/communication-profiles/select-options/"
      selectedValueLink="/edi/communication-profiles/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {option.method} · {option.status}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function EDIMappingProfileAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EDIMappingProfile, T>) {
  return (
    <AutocompleteField<EDIMappingProfile, T>
      link="/edi/mapping-profiles/select-options/"
      selectedValueLink="/edi/mapping-profiles/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
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

export function GLAccountAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<GLAccount, T>) {
  return (
    <AutocompleteField<GLAccount, T>
      link="/gl-accounts/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.accountCode} - ${option.name}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.accountCode}</span>
          {option?.name && (
            <span className="w-full truncate text-2xs text-muted-foreground">{option.name}</span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function AccountTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<AccountType, T>) {
  return (
    <AutocompleteField<AccountType, T>
      link="/account-types/select-options/"
      popoutLink="/billing/configuration-files/account-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => <ColorOptionValue color={option.color} value={option.code} />}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.name && (
            <span className="w-full truncate text-2xs text-muted-foreground">{option.name}</span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function LocationAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<Location, T>) {
  return (
    <AutocompleteField<Location, T>
      link="/locations/select-options/"
      popoutLink="/dispatch/locations"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>
            {option.code} - {option.name}
          </span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {formatLocation(option)}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function OrganizationAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<OrganizationSelectOption, T>) {
  return (
    <AutocompleteField<OrganizationSelectOption, T>
      link="/organizations/select-options/"
      selectedValueLink="/organizations/"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) =>
        option.scacCode ? `${option.scacCode} - ${option.name}` : option.name
      }
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.scacCode ? `${option.scacCode} - ${option.name}` : option.name}</span>
          {option.city && (
            <span className="w-full truncate text-2xs text-muted-foreground">{option.city}</span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function EDIPartnerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EDIPartner, T>) {
  return (
    <AutocompleteField<EDIPartner, T>
      link="/edi/partners/select-options/"
      selectedValueLink="/edi/partners/"
      initialLimit={50}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>
            {option.code} - {option.name}
          </span>
          {option.internalOrganization?.name && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.internalOrganization.name}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function EDIDocumentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EDIDocumentType, T>) {
  return (
    <AutocompleteField<EDIDocumentType, T>
      link="/edi/catalog/document-types/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>
            {option.code} - {option.name}
          </span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {option.transactionSet} / {option.direction} / {option.defaultVersion}
          </span>
        </div>
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
}: BaseAutocompleteFieldProps<EDITemplate, T> & {
  transactionSet?: string;
  direction?: string;
}) {
  return (
    <AutocompleteField<EDITemplate, T>
      link="/edi/templates/select-options/"
      selectedValueLink="/edi/templates/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {option.description ?? option.status}
          </span>
        </div>
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
}: BaseAutocompleteFieldProps<EDIPartnerDocumentProfile, T> & {
  partnerId?: string;
  transactionSet?: string;
  direction?: string;
}) {
  return (
    <AutocompleteField<EDIPartnerDocumentProfile, T>
      link="/edi/document-profiles/select-options/"
      selectedValueLink="/edi/document-profiles/"
      extraSearchParams={{
        ...(partnerId ? { partnerId } : {}),
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
          {option.partner ? (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.partner.code} - {option.partner.name}
            </span>
          ) : null}
        </div>
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
  ControlledAutocompleteFieldProps<EDIPartner>,
  | "label"
  | "link"
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
    <ControlledAutocompleteField<EDIPartner>
      label={label}
      link="/edi/partners/select-options/"
      selectedValueLink="/edi/partners/"
      initialLimit={50}
      placeholder={placeholder}
      renderOption={(option) => (
        <EDIOptionStack primary={`${option.code} - ${option.name}`} secondary={option.kind} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
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
  ControlledAutocompleteFieldProps<EDITemplate>,
  | "label"
  | "link"
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
  return (
    <ControlledAutocompleteField<EDITemplate>
      label="Template"
      link="/edi/templates/select-options/"
      selectedValueLink="/edi/templates/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...extraSearchParams,
      }}
      renderOption={(option) => (
        <EDIOptionStack primary={option.name} secondary={option.description ?? option.status} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.name}
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
  ControlledAutocompleteFieldProps<EDIPartnerDocumentProfile>,
  | "label"
  | "link"
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
  return (
    <ControlledAutocompleteField<EDIPartnerDocumentProfile>
      label="Document Profile"
      link="/edi/document-profiles/select-options/"
      selectedValueLink="/edi/document-profiles/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...(partnerId ? { partnerId } : {}),
        ...extraSearchParams,
      }}
      renderOption={(option) => (
        <EDIOptionStack
          primary={option.name}
          secondary={option.partner ? `${option.partner.code} - ${option.partner.name}` : undefined}
        />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.name}
      {...props}
    />
  );
}

export function FormulaTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FormulaTemplate, T>) {
  return (
    <AutocompleteField<FormulaTemplate, T>
      link="/formula-templates/select-options/"
      popoutLink="billing/configuration-files/formula-templates"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              render={
                <div className="flex size-full flex-col items-start">
                  <span>{option.name}</span>
                  {option?.description && (
                    <span className="w-full truncate text-2xs text-muted-foreground">
                      {option?.description}
                    </span>
                  )}
                </div>
              }
            />
            <TooltipContent align="center" sideOffset={20} side="left" className="size-full">
              <div className="flex size-full flex-col gap-0.5">
                <h3 className="font-semibold">Expression:</h3>
                <div className="flex w-full rounded-md border border-muted/20 bg-muted/10 p-1">
                  {option?.expression}
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
      {...props}
    />
  );
}

export function LocationCategoryAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<LocationCategory, T>) {
  return (
    <AutocompleteField<LocationCategory, T>
      link="/location-categories/select-options/"
      popoutLink="/dispatch/configuration-files/location-categories"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color ?? undefined} value={option.name} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color ?? undefined} value={option.name} />
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

export function DocumentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<DocumentType, T>) {
  return (
    <AutocompleteField<DocumentType, T>
      link="/document-types/select-options/"
      popoutLink="/billing/configuration-files/document-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color ?? undefined} value={option.code} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <ColorOptionValue color={option.color ?? undefined} value={option.code} />
          {option?.name && (
            <span className="w-full truncate text-2xs text-muted-foreground">{option.name}</span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function DocumentTypeMultiSelectField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<DocumentType, T>) {
  return (
    <MultiSelectAutocompleteField<DocumentType, T>
      link="/document-types/select-options/"
      nestedValues
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || ""}
      getOptionLabel={(option) => option.name || ""}
      renderOption={(option) => (
        <ColorOptionValue color={option.color ?? undefined} value={option.code} />
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
          <span className="w-full truncate text-2xs text-muted-foreground">
            {option.documentType?.name || option.resourceType}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function HazardousMaterialAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<HazardousMaterial, T>) {
  return (
    <AutocompleteField<HazardousMaterial, T>
      link="/hazardous-materials/select-options/"
      popoutLink="/shipment-management/configuration-files/hazardous-materials"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
          {option?.class && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              Class {option.class}
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
}: BaseAutocompleteFieldProps<Commodity, T>) {
  return (
    <AutocompleteField<Commodity, T>
      link="/commodities/select-options/"
      popoutLink="/shipment-management/configuration-files/commodities"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <div className="flex items-center gap-1.5">
            <span>{option.name}</span>
            {option?.hazardousMaterialId && (
              <span className="inline-flex items-center rounded border border-yellow-600/30 bg-yellow-600/20 px-1 py-px text-[10px] font-medium text-yellow-700 dark:text-yellow-400">
                Hazmat
              </span>
            )}
          </div>
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

export function FiscalYearAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FiscalYear, T>) {
  return (
    <AutocompleteField<FiscalYear, T>
      link="/fiscal-years/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">{option.year}</span>
        </div>
      )}
      {...props}
    />
  );
}

export function FiscalPeriodAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FiscalPeriod, T>) {
  return (
    <AutocompleteField<FiscalPeriod, T>
      link="/fiscal-periods/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name || `Period ${option.periodNumber}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.name || `Period ${option.periodNumber}`}</span>
          <span className="w-full truncate text-2xs text-muted-foreground capitalize">
            {option.periodType}
          </span>
        </div>
      )}
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

export function AccessorialChargeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<AccessorialCharge, T>) {
  return (
    <AutocompleteField<AccessorialCharge, T>
      link="/accessorial-charges/select-options/"
      popoutLink="/billing/configuration-files/accessorial-charges"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span>{option.code}</span>
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
