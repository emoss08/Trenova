import type { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import type { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import type { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import type { HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import type { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import type { WorkerSchema } from "@/lib/schemas/worker-schema";
import { truncateText } from "@/lib/utils";
import { Status } from "@/types/common";
import type { User } from "@/types/user";
import type {
  Control,
  FieldValues,
  Path,
  RegisterOptions,
} from "react-hook-form";
import { AutocompleteField } from "../fields/autocomplete";
import { ColorOptionValue } from "../fields/select-components";
import { PackingGroupBadge } from "../status-badge";
import { LazyImage } from "./image";

type BaseAutocompleteFieldProps<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  label: string;
  rules?: RegisterOptions<T, Path<T>>;
  description: string;
  clearable?: boolean;
  placeholder?: string;
  extraSearchParams?: Record<string, string | string[]>;
};

export function HazardousMaterialAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<HazardousMaterialSchema, T>
      name={name}
      control={control}
      rules={rules}
      link="/hazardous-materials/"
      label={label}
      clearable={clearable}
      placeholder={placeholder || "Select Hazardous Material"}
      description={description}
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
    />
  );
}

export function UserAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<User, T>
      name={name}
      control={control}
      rules={rules}
      link="/users/"
      label={label}
      clearable={clearable}
      placeholder={placeholder || "Select User"}
      description={description}
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
            layout="fullWidth"
          />
          <span className="text-xs font-medium">
            {truncateText(option.name, 20)}
          </span>
        </div>
      )}
      renderOption={(option) => (
        <div className="flex flex-col gap-1 items-start size-full">
          <div className="flex flex-row items-center gap-1.5">
            <LazyImage
              src={
                option.profilePicUrl ||
                `https://avatar.vercel.sh/${option.name}.svg`
              }
              alt={option.name}
              className="size-4 rounded-full"
              layout="fullWidth"
            />
            <span className="text-xs font-medium">
              {truncateText(option.name, 15)}
            </span>
          </div>
          <span className="text-2xs text-muted-foreground truncate w-full">
            {truncateText(option.emailAddress, 20)}
          </span>
        </div>
      )}
    />
  );
}

export function LocationCategoryAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<LocationCategorySchema, T>
      name={name}
      control={control}
      rules={rules}
      link="/location-categories/"
      label={label}
      clearable={clearable}
      placeholder={placeholder || "Select Location Category"}
      description={description}
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
    />
  );
}

export function EquipmentTypeAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  extraSearchParams,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<EquipmentTypeSchema, T>
      name={name}
      control={control}
      rules={rules}
      label={label}
      link="/equipment-types/"
      clearable={clearable}
      placeholder={placeholder || "Select Equipment Type"}
      description={description}
      getOptionValue={(option) => option.id || ""}
      extraSearchParams={extraSearchParams}
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
    />
  );
}
export function EquipmentManufacturerAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<EquipmentManufacturerSchema, T>
      name={name}
      control={control}
      rules={rules}
      link="/equipment-manufacturers/"
      label={label}
      placeholder={placeholder || "Select Equipment Manufacturer"}
      description={description}
      clearable={clearable}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => option.name}
      renderOption={(option) => option.name}
    />
  );
}

export function FleetCodeAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<FleetCodeSchema, T>
      name={name}
      control={control}
      rules={rules}
      link="/fleet-codes/"
      label={label}
      placeholder={placeholder || "Select Fleet Code"}
      description={description}
      clearable={clearable}
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
    />
  );
}

export function WorkerAutocompleteField<T extends FieldValues>({
  control,
  name,
  label,
  clearable,
  description,
  placeholder,
  rules,
  extraSearchParams,
}: BaseAutocompleteFieldProps<T>) {
  return (
    <AutocompleteField<WorkerSchema, T>
      name={name}
      control={control}
      rules={rules}
      link="/workers/"
      label={label}
      placeholder={placeholder || "Select Worker"}
      description={description}
      extraSearchParams={{
        status: [Status.Active], // * By default, only active workers are shown
        ...extraSearchParams, // * Include any additional search parameters
      }}
      clearable={clearable}
      getOptionValue={(option) => option.id || ""}
      getDisplayValue={(option) => `${option.firstName} ${option.lastName}`}
      renderOption={(option) => `${option.firstName} ${option.lastName}`}
    />
  );
}
