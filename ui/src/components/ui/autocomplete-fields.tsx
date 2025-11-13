import { holdTypeChoices } from "@/lib/choices";
import type { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { categoryToHumanReadable } from "@/lib/schemas/account-type-schema";
import type { CommoditySchema } from "@/lib/schemas/commodity-schema";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import type { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import type { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import type { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import type { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import type { FormulaTemplateSchema } from "@/lib/schemas/formula-template-schema";
import type { HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";
import type { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import type { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import type { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import {
  EquipmentStatus,
  type TractorSchema,
} from "@/lib/schemas/tractor-schema";
import type { TrailerSchema } from "@/lib/schemas/trailer-schema";
import type { RoleSchema } from "@/lib/schemas/user-schema";
import type { WorkerSchema } from "@/lib/schemas/worker-schema";
import { formatLocation, truncateText, USDollarFormat } from "@/lib/utils";
import {
  AccountTypeSelectOptionResponse,
  GLAccountSelectOptionResponse,
  UserSelectOptionResponse,
} from "@/types/auto-complete-fields";
import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import type {
  Control,
  FieldPath,
  FieldValues,
  Path,
  RegisterOptions,
} from "react-hook-form";
import { MultiSelectAutocompleteField } from "../fields/async-multi-select";
import { AutocompleteField } from "../fields/autocomplete";
import { ColorOptionValue } from "../fields/select-components";
import { ExpressionHighlight } from "../formula-templates/expression-highliter";
import { PackingGroupBadge } from "../status-badge";
import { LazyImage } from "./image";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

type BaseAutocompleteFieldProps<TOption, TForm extends FieldValues> = {
  control: Control<TForm>;
  name: Path<TForm>;
  label?: string;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  description?: string;
  clearable?: boolean;
  placeholder?: string;
  extraSearchParams?: Record<string, string | string[]>;
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

export function HazardousMaterialAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<HazardousMaterialSchema, T>) {
  return (
    <AutocompleteField<HazardousMaterialSchema, T>
      link="/hazardous-materials/"
      getOptionValue={(option) => option.id ?? ""}
      getDisplayValue={(option) => `${option.name}`}
      placeholder="Select a hazardous material"
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <div className="flex flex-row items-center">
            <span className="text-xs font-medium">
              {truncateText(option.name, 10)}
            </span>
            <PackingGroupBadge
              group={option.packingGroup}
              className="h-4 border-none bg-transparent text-2xs"
            />
          </div>
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {truncateText(option?.description, 35)}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function UserAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<UserSelectOptionResponse, T>) {
  return (
    <AutocompleteField<UserSelectOptionResponse, T>
      link="/users/select-options/"
      getOptionValue={(option) => option.id ?? ""}
      getDisplayValue={(option) => (
        <div className="flex shrink-0 flex-row items-center gap-1.5">
          <LazyImage
            src={
              option.profilePicUrl ||
              `https://avatar.vercel.sh/${option.name}.svg`
            }
            alt={option.name}
            className="size-3 shrink-0 rounded-full"
          />
          <span className="text-xs font-medium">
            {truncateText(option.name, 20)}
          </span>
        </div>
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <div className="flex w-full flex-row items-center gap-1.5">
            <LazyImage
              src={
                option.profilePicUrl ||
                `https://avatar.vercel.sh/${option.name}.svg`
              }
              alt={option.name}
              className="size-4 shrink-0 rounded-full"
            />
            <span className="w-full truncate text-xs font-medium">
              {option.name}
            </span>
          </div>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {truncateText(option.emailAddress, 20)}
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function LocationCategoryAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<LocationCategorySchema, T>) {
  return (
    <AutocompleteField<LocationCategorySchema, T>
      link="/location-categories/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue
          color={option.color}
          value={option.name}
          textClassName="font-normal"
        />
      )}
      renderOption={(option) => (
        <div className="flex flex-col items-start">
          <ColorOptionValue color={option.color} value={option.name} />
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
}: BaseAutocompleteFieldProps<GLAccountSelectOptionResponse, T>) {
  return (
    <AutocompleteField<GLAccountSelectOptionResponse, T>
      link="/gl-accounts/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.accountCode}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="text-xs font-medium">{option.accountCode}</span>
          {option.name && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option.name}
            </span>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function AccountTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<AccountTypeSelectOptionResponse, T>) {
  return (
    <AutocompleteField<AccountTypeSelectOptionResponse, T>
      link="/account-types/select-options/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code}`}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start">
          <span className="text-xs font-medium">{option.code}</span>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {truncateText(option.name, 20)} (
            {categoryToHumanReadable(option.category)})
          </span>
        </div>
      )}
      {...props}
    />
  );
}

export function EquipmentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<EquipmentTypeSchema, T>) {
  return (
    <AutocompleteField<EquipmentTypeSchema, T>
      link="/equipment-types/"
      popoutLink="/equipment/configurations/equipment-types"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color} value={option.code} />
      )}
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
}: BaseAutocompleteFieldProps<EquipmentManufacturerSchema, T>) {
  return (
    <AutocompleteField<EquipmentManufacturerSchema, T>
      link="/equipment-manufacturers/"
      popoutLink="/equipment/configurations/equipment-manufacturers"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => option.name}
      {...props}
    />
  );
}

export function FleetCodeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FleetCodeSchema, T>) {
  return (
    <AutocompleteField<FleetCodeSchema, T>
      link="/fleet-codes/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color} value={option.code} />
      )}
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
}: BaseAutocompleteFieldProps<WorkerSchema, T>) {
  return (
    <AutocompleteField<WorkerSchema, T>
      link="/workers/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.firstName} ${option.lastName}`}
      renderOption={(option) => `${option.firstName} ${option.lastName}`}
      extraSearchParams={{
        status: [Status.Active], // * Always filter by active workers
      }}
      {...props}
    />
  );
}

export function TractorAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<TractorSchema, T>) {
  return (
    <AutocompleteField<TractorSchema, T>
      link="/tractors/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code}`}
      renderOption={(option) => `${option.code}`}
      extraSearchParams={{
        status: EquipmentStatus.Available,
      }}
      {...props}
    />
  );
}

export function TrailerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<TrailerSchema, T>) {
  return (
    <AutocompleteField<TrailerSchema, T>
      link="/trailers/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.code}`}
      renderOption={(option) => `${option.code}`}
      extraSearchParams={{
        status: EquipmentStatus.Available,
      }}
      {...props}
    />
  );
}

export function ShipmentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<ShipmentTypeSchema, T>) {
  return (
    <AutocompleteField<ShipmentTypeSchema, T>
      link="/shipment-types/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color} value={option.code} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      extraSearchParams={{
        status: [Status.Active],
      }}
      {...props}
    />
  );
}

export function ServiceTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<ServiceTypeSchema, T>) {
  return (
    <AutocompleteField<ServiceTypeSchema, T>
      link="/service-types/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color} value={option.code} />
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      extraSearchParams={{
        status: [Status.Active],
      }}
      {...props}
    />
  );
}

export function LocationAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<LocationSchema, T>) {
  return (
    <AutocompleteField<LocationSchema, T>
      link="/locations/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <span className="w-full truncate text-left">{option.name}</span>
      )}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <div className="flex w-full items-center gap-1">
            <span className="w-prose truncate text-sm font-normal">
              {option.name}
            </span>
            <span className="w-prose truncate text-2xs text-muted-foreground">
              {option.locationCategory?.name}
            </span>
          </div>
          <span className="w-full truncate text-2xs text-muted-foreground">
            {formatLocation(option)}
          </span>
        </div>
      )}
      extraSearchParams={{
        status: [Status.Active],
        includeState: "true",
        includeCategory: "true",
      }}
      {...props}
    />
  );
}

export function AccessorialChargeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<AccessorialChargeSchema, T>) {
  const mapAmount = (method: AccessorialChargeMethod, amount: number) => {
    switch (method) {
      case AccessorialChargeMethod.Flat:
        return USDollarFormat(amount);
      case AccessorialChargeMethod.Distance:
        return `${USDollarFormat(amount)} per mile`;
      case AccessorialChargeMethod.Percentage:
        return `${amount}%`;
    }
  };
  return (
    <AutocompleteField<AccessorialChargeSchema, T>
      link="/accessorial-charges/"
      getOptionValue={(option) => option.id || ""}
      placeholder="Select Accessorial Charge"
      getDisplayValue={(option) => option.code}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <div className="flex w-full flex-row items-start gap-1">
            <p className="text-sm font-medium">{option.code}</p>
            <p className="text-sm font-medium text-muted-foreground">
              {mapAmount(option.method, option.amount)}
            </p>
          </div>
          {option.description && (
            <p className="w-full truncate text-xs text-muted-foreground">
              {option.description}
            </p>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function CommodityAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<CommoditySchema, T>) {
  return (
    <AutocompleteField<CommoditySchema, T>
      link="/commodities/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => option.name}
      {...props}
    />
  );
}

export function HoldReasonAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<HoldReasonSchema, T>) {
  return (
    <AutocompleteField<HoldReasonSchema, T>
      link="/hold-reasons/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.label}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <p className="flex flex-row items-center gap-1">
            <span className="max-w-prose truncate text-sm font-medium">
              {option.label}
            </span>
            <span className="truncate text-2xs text-muted-foreground">
              {
                holdTypeChoices.find((choice) => choice.value === option.type)
                  ?.label
              }
            </span>
          </p>
          {option.description && (
            <p className="max-w-prose truncate text-xs text-muted-foreground">
              {option.description}
            </p>
          )}
        </div>
      )}
      {...props}
    />
  );
}

export function CustomerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<CustomerSchema, T>) {
  return (
    <AutocompleteField<CustomerSchema, T>
      link="/customers/"
      getOptionValue={(option) => option.id ?? ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <p className="text-sm font-medium">{option.code}</p>
          {option.name && (
            <p className="w-full truncate text-xs text-muted-foreground">
              {option.name}
            </p>
          )}
        </div>
      )}
      extraSearchParams={{
        status: [Status.Active],
      }}
      {...props}
    />
  );
}

export function RoleAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<RoleSchema, T>) {
  return (
    <MultiSelectAutocompleteField<RoleSchema, T>
      link="/roles/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => option.name}
      getOptionLabel={(option) => option.name}
      nestedValues={true}
      extraSearchParams={{
        includePermissions: "true",
      }}
      {...props}
    />
  );
}

export function DocumentTypeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseMultiSelectAutocompleteFieldProps<DocumentTypeSchema, T>) {
  return (
    <MultiSelectAutocompleteField<DocumentTypeSchema, T>
      {...props}
      link="/document-types/"
      getOptionValue={(option) => option.id || ""}
      getOptionLabel={(option) => option.name}
      nestedValues={true}
      renderOption={(option) => (
        <div className="flex size-full flex-col items-start gap-0.5">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="w-full truncate text-2xs text-muted-foreground">
              {option?.description}
            </span>
          )}
        </div>
      )}
      getDisplayValue={(option) => option.name}
      renderBadge={(option) => option.name}
    />
  );
}

export function FormulaTemplateAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<FormulaTemplateSchema, T>) {
  return (
    <TooltipProvider>
      <AutocompleteField<FormulaTemplateSchema, T>
        link="/formula-templates/"
        getOptionValue={(option) => option.id ?? ""}
        getDisplayValue={(option) => truncateText(option.name, 30)}
        renderOption={(option) => (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex size-full flex-col items-start gap-0.5">
                <p className="w-full truncate text-sm font-medium">
                  {option.name}
                </p>
                {option.description && (
                  <p className="w-full truncate text-xs text-muted-foreground">
                    {option.description}
                  </p>
                )}
              </div>
            </TooltipTrigger>
            <TooltipContent className="max-w-md" side="left" sideOffset={10}>
              <div className="flex size-full flex-col items-start gap-0.5">
                <p className="w-full truncate text-sm font-medium">
                  {option.name}
                </p>
                {option.description && (
                  <p className="text-xs dark:text-muted-foreground">
                    {option.description}
                  </p>
                )}
                <div className="flex size-full flex-col items-start gap-0.5 rounded-md border border-muted/5 bg-muted/5">
                  <h4 className="w-full truncate border-b border-muted/5 p-1 text-sm font-medium">
                    Expression
                  </h4>
                  <p className="p-1 text-xs text-wrap text-background">
                    <ExpressionHighlight expression={option.expression} />
                  </p>
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        )}
        {...props}
      />
    </TooltipProvider>
  );
}
