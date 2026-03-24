import { formatLocation } from "@/lib/utils";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import type { AccountType } from "@/types/account-type";
import type { Commodity } from "@/types/commodity";
import type { Customer } from "@/types/customer";
import type { DocumentType } from "@/types/document-type";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import type { EquipmentType } from "@/types/equipment-type";
import type { FleetCode } from "@/types/fleet-code";
import type { FormulaTemplate } from "@/types/formula-template";
import type { GLAccount } from "@/types/gl-account";
import type { HazardousMaterial } from "@/types/hazardous-material";
import type { Location } from "@/types/location";
import type { LocationCategory } from "@/types/location-category";
import type { API_ENDPOINTS } from "@/types/server";
import type { ServiceType } from "@/types/service-type";
import type { ShipmentType } from "@/types/shipment-type";
import type { Tractor } from "@/types/tractor";
import type { Trailer } from "@/types/trailer";
import type { UsState } from "@/types/us-state";
import type { User, UserRoleAssignment } from "@/types/user";
import type { Worker } from "@/types/worker";
import type { Control, FieldPath, FieldValues, Path, RegisterOptions } from "react-hook-form";
import { AutocompleteField } from "./fields/autocomplete/autocomplete";
import { MultiSelectAutocompleteField } from "./fields/multi-select-field";
import { ColorOptionValue } from "./fields/select-components";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./ui/tooltip";

type BaseAutocompleteFieldProps<TOption, TForm extends FieldValues> = {
  control: Control<TForm>;
  name: Path<TForm>;
  label?: string;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  description?: string;
  clearable?: boolean;
  placeholder?: string;
  extraSearchParams?: Record<string, string | string[]>;
  selectedValueLink?: API_ENDPOINTS;
  onOptionChange?: (option: TOption | null) => void;
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
};

export function RoleAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<UserRoleAssignment, T>) {
  return (
    <MultiSelectAutocompleteField<UserRoleAssignment, T>
      link="/role-assignments/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.role?.name || ""}
      renderOption={(option) => option.role?.name || ""}
      getOptionLabel={(option) => option.role?.name || ""}
      extraSearchParams={{
        expandRoles: "true",
      }}
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
