import type { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import type { CommoditySchema } from "@/lib/schemas/commodity-schema";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import type { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import type { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import type { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import type { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import type { HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import type { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import type { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import type { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import {
  EquipmentStatus,
  type TractorSchema,
} from "@/lib/schemas/tractor-schema";
import type { TrailerSchema } from "@/lib/schemas/trailer-schema";
import type { RoleSchema, UserSchema } from "@/lib/schemas/user-schema";
import type { WorkerSchema } from "@/lib/schemas/worker-schema";
import { formatLocation, truncateText } from "@/lib/utils";
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
import { PackingGroupBadge } from "../status-badge";
import { LazyImage } from "./image";

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
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.name}`}
      renderOption={(option) => (
        <div className="flex flex-col gap-0.5 items-start size-full">
          <div className="flex flex-row items-center">
            <span className="text-xs font-medium">
              {truncateText(option.name, 10)}
            </span>
            <PackingGroupBadge
              group={option.packingGroup}
              className="h-4 text-2xs bg-transparent border-none"
            />
          </div>
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
}: BaseAutocompleteFieldProps<UserSchema, T>) {
  return (
    <AutocompleteField<UserSchema, T>
      link="/users/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <div className="flex flex-row items-center gap-1.5">
          <LazyImage
            src={
              option.profilePicUrl ||
              `https://avatar.vercel.sh/${option.name}.svg`
            }
            alt={option.name}
            className="size-3 rounded-full"
          />
          <span className="text-xs font-medium">
            {truncateText(option.name, 20)}
          </span>
        </div>
      )}
      renderOption={(option) => (
        <div className="flex flex-col gap-1 items-start size-full">
          <div className="flex flex-row items-center gap-1.5 w-full shrink-0">
            <LazyImage
              src={
                option.profilePicUrl ||
                `https://avatar.vercel.sh/${option.name}.svg`
              }
              alt={option.name}
              className="size-4 rounded-full shrink-0"
            />
            <span className="w-full truncate text-xs font-medium">
              {option.name}
            </span>
          </div>
          <span className="text-2xs text-muted-foreground truncate w-full">
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
            <span className="text-2xs text-muted-foreground truncate w-full">
              {option.description}
            </span>
          )}
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
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => (
        <ColorOptionValue color={option.color} value={option.code} />
      )}
      renderOption={(option) => (
        <div className="flex flex-col items-start size-full">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
        <ColorOptionValue color={option.color} value={option.name} />
      )}
      renderOption={(option) => (
        <div className="flex flex-col items-start size-full">
          <ColorOptionValue color={option.color} value={option.name} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
        <div className="flex flex-col gap-0.5 items-start size-full">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
        <div className="flex flex-col gap-0.5 items-start size-full">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
      getDisplayValue={(option) => option.name}
      renderOption={(option) => (
        <div className="flex flex-col gap-0.5 items-start size-full">
          <span className="text-sm font-normal">{option.name}</span>
          <span className="text-2xs text-muted-foreground truncate w-full">
            {formatLocation(option)}
          </span>
        </div>
      )}
      extraSearchParams={{
        status: [Status.Active],
      }}
      {...props}
    />
  );
}

export function AccessorialChargeAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<AccessorialChargeSchema, T>) {
  return (
    <AutocompleteField<AccessorialChargeSchema, T>
      link="/accessorial-charges/"
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => (
        <div className="flex flex-col gap-0.5 items-start size-full">
          <p className="text-sm font-medium">{option.code}</p>
          {option.description && (
            <p className="text-xs text-muted-foreground truncate w-full">
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

export function CustomerAutocompleteField<T extends FieldValues>({
  ...props
}: BaseAutocompleteFieldProps<CustomerSchema, T>) {
  return (
    <AutocompleteField<CustomerSchema, T>
      link="/customers/"
      getOptionValue={(option) => option.id ?? ""}
      getDisplayValue={(option) => option.code}
      renderOption={(option) => (
        <div className="flex flex-col gap-0.5 items-start size-full">
          <p className="text-sm font-medium">{option.code}</p>
          {option.name && (
            <p className="text-xs text-muted-foreground truncate w-full">
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
        <div className="flex flex-col gap-0.5 items-start size-full">
          <ColorOptionValue color={option.color} value={option.code} />
          {option?.description && (
            <span className="text-2xs text-muted-foreground truncate w-full">
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
