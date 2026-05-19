import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { AutocompleteCommandContent } from "@/components/fields/autocomplete/autocomplete-content";
import { AutocompleteTrigger } from "@/components/fields/autocomplete/autocomplete-input";
import { FieldWrapper } from "@/components/fields/field-components";
import { SelectField } from "@/components/fields/select-field";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type {
  EDIDocumentType,
  EDIPartner,
  EDIPartnerDocumentProfile,
  EDIPartnerSettingField,
  EDISourceContextField,
  EDITemplate,
} from "@/types/edi";
import type { SelectOption } from "@/types/fields";
import type { API_ENDPOINTS, SELECT_OPTIONS_ENDPOINTS } from "@/types/server";
import { SearchIcon } from "lucide-react";
import { useEffect, useId, useState, type ReactNode } from "react";
import { useForm, type FieldValues } from "react-hook-form";
import { insertPathReference } from "../utils/edi-designer-utils";

type ControlledSelectFormValues = {
  value: string;
};

export function ControlledSelectField({
  label,
  value,
  options,
  onValueChange,
  placeholder = "Select",
  disabled,
  clearable = true,
}: {
  label: string;
  value: string;
  options: SelectOption[];
  onValueChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  clearable?: boolean;
}) {
  const { control, getValues, setValue } = useForm<ControlledSelectFormValues>({
    defaultValues: { value: value ?? "" },
  });

  useEffect(() => {
    const nextValue = value ?? "";
    if (getValues("value") === nextValue) return;
    setValue("value", nextValue, {
      shouldDirty: false,
      shouldTouch: false,
      shouldValidate: false,
    });
  }, [getValues, setValue, value]);

  return (
    <SelectField
      control={control}
      name="value"
      label={label}
      options={options}
      placeholder={placeholder}
      isReadOnly={disabled}
      isClearable={clearable}
      onValueChange={onValueChange}
    />
  );
}

export function ControlledAutocompleteField<TOption>({
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
}: {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  onOptionChange?: (option: TOption | null) => void;
  description?: string;
  link: SELECT_OPTIONS_ENDPOINTS;
  selectedValueLink?: API_ENDPOINTS;
  renderOption: (option: TOption) => React.ReactNode;
  getOptionValue: (option: TOption) => string | number;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder?: string;
  disabled?: boolean;
  clearable?: boolean;
  extraSearchParams?: Record<string, string | string[]>;
  initialLimit?: number;
  noResultsMessage?: string;
}) {
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

export function EDIDocumentTypeAutocompleteField({
  value,
  onValueChange,
  onOptionChange,
  disabled,
}: {
  value: string;
  onValueChange: (value: string) => void;
  onOptionChange?: (option: EDIDocumentType | null) => void;
  disabled?: boolean;
}) {
  return (
    <ControlledAutocompleteField<EDIDocumentType>
      label="Document Type"
      value={value}
      onValueChange={onValueChange}
      onOptionChange={onOptionChange}
      link="/edi/document-types/select-options/"
      renderOption={(option) => (
        <OptionStack
          primary={`${option.code} - ${option.name}`}
          secondary={`${option.transactionSet} / ${option.direction} / ${option.defaultVersion}`}
        />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      disabled={disabled}
    />
  );
}

export function EDIPartnerAutocompleteField({
  value,
  onValueChange,
  disabled,
  label = "Partner",
  placeholder,
}: {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
  label?: string;
  placeholder?: string;
}) {
  return (
    <ControlledAutocompleteField<EDIPartner>
      label={label}
      value={value}
      onValueChange={onValueChange}
      link="/edi/partners/select-options/"
      selectedValueLink="/edi/partners/"
      extraSearchParams={{ kind: "External" }}
      initialLimit={50}
      placeholder={placeholder}
      renderOption={(option) => (
        <OptionStack primary={`${option.code} - ${option.name}`} secondary={option.kind} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => `${option.code} - ${option.name}`}
      disabled={disabled}
    />
  );
}

export function EDITemplateAutocompleteField({
  value,
  onValueChange,
  disabled,
  transactionSet,
  direction,
}: {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
  transactionSet?: string;
  direction?: string;
}) {
  return (
    <ControlledAutocompleteField<EDITemplate>
      label="Template"
      value={value}
      onValueChange={onValueChange}
      link="/edi/templates/select-options/"
      selectedValueLink="/edi/templates/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
      }}
      renderOption={(option) => (
        <OptionStack primary={option.name} secondary={option.description ?? option.status} />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.name}
      disabled={disabled}
    />
  );
}

export function EDIDocumentProfileAutocompleteField({
  value,
  onValueChange,
  onOptionChange,
  disabled,
  description,
  placeholder,
  noResultsMessage,
  partnerId,
  transactionSet,
  direction,
}: {
  value: string;
  onValueChange: (value: string) => void;
  onOptionChange?: (option: EDIPartnerDocumentProfile | null) => void;
  disabled?: boolean;
  description?: string;
  placeholder?: string;
  noResultsMessage?: string;
  partnerId?: string;
  transactionSet?: string;
  direction?: string;
}) {
  return (
    <ControlledAutocompleteField<EDIPartnerDocumentProfile>
      label="Document Profile"
      value={value}
      onValueChange={onValueChange}
      onOptionChange={onOptionChange}
      link="/edi/document-profiles/select-options/"
      selectedValueLink="/edi/document-profiles/"
      extraSearchParams={{
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...(partnerId ? { partnerId } : {}),
      }}
      renderOption={(option) => (
        <OptionStack
          primary={option.name}
          secondary={option.partner ? `${option.partner.code} - ${option.partner.name}` : undefined}
        />
      )}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => option.name}
      disabled={disabled}
      description={description}
      placeholder={placeholder}
      noResultsMessage={noResultsMessage}
    />
  );
}

export function sourceContextFieldDisplayText(field: EDISourceContextField) {
  return `${field.path} (${field.dataType})`;
}

export function sourceContextFieldSearchText(field: EDISourceContextField) {
  return [field.path, field.displayName, field.description, field.dataType]
    .filter(Boolean)
    .join(" ");
}

export function partnerSettingFieldDisplayText(field: EDIPartnerSettingField) {
  return `${field.path} (${field.dataType})`;
}

export function partnerSettingFieldSearchText(field: EDIPartnerSettingField) {
  return [field.path, field.label, field.description, field.groupKey].filter(Boolean).join(" ");
}

export function toSourcePathReference(field: EDISourceContextField) {
  return field.path;
}

export function toPartnerSettingPath(field: EDIPartnerSettingField) {
  return field.path;
}

export function toPartnerConditionPathReference(field: EDIPartnerSettingField) {
  return `partner.${field.path}`;
}

export function toPartnerTransformPathReference(field: EDIPartnerSettingField) {
  return `$${toPartnerConditionPathReference(field)}`;
}

export function PathReferenceField({
  label,
  value,
  onChange,
  disabled,
  sourceOnlyRepeated,
  partner,
  transactionSet,
  direction,
  documentTypeId,
  x12Version,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  sourceOnlyRepeated?: boolean;
  partner?: boolean;
  transactionSet?: string;
  direction?: string;
  documentTypeId?: string;
  x12Version?: string;
}) {
  return (
    <div className="space-y-1">
      <PathInput label={label} value={value} onChange={onChange} disabled={disabled} />
      {partner ? (
        <PartnerSettingPicker
          disabled={disabled}
          onPick={onChange}
          transactionSet={transactionSet}
          direction={direction}
          documentTypeId={documentTypeId}
          x12Version={x12Version}
        />
      ) : (
        <SourceContextPicker
          disabled={disabled}
          onPick={onChange}
          extraSearchParams={{
            ...(sourceOnlyRepeated ? { repeated: "true" } : {}),
            ...(transactionSet ? { transactionSet } : {}),
            ...(direction ? { direction } : {}),
            ...(x12Version ? { x12Version } : {}),
          }}
        />
      )}
    </div>
  );
}

export function PathInsertField({
  label,
  value,
  placeholder,
  disabled,
  onChange,
  transactionSet,
  direction,
  documentTypeId,
  x12Version,
}: {
  label: string;
  value: string;
  placeholder?: string;
  disabled: boolean;
  onChange: (value: string) => void;
  transactionSet?: string;
  direction?: string;
  documentTypeId?: string;
  x12Version?: string;
}) {
  return (
    <div className="space-y-1">
      <PathInput
        label={label}
        value={value}
        onChange={onChange}
        disabled={disabled}
        placeholder={placeholder}
      />
      <div className="grid grid-cols-2 gap-1">
        <SourceContextPicker
          disabled={disabled}
          placeholder="Source"
          extraSearchParams={{
            ...(transactionSet ? { transactionSet } : {}),
            ...(direction ? { direction } : {}),
            ...(x12Version ? { x12Version } : {}),
          }}
          onPick={(path) => onChange(insertPathReference(value, path))}
        />
        <PartnerSettingPicker
          disabled={disabled}
          placeholder="Partner"
          transactionSet={transactionSet}
          direction={direction}
          documentTypeId={documentTypeId}
          x12Version={x12Version}
          getPickValue={toPartnerTransformPathReference}
          onPick={(path) => onChange(insertPathReference(value, path, false))}
        />
      </div>
    </div>
  );
}

function SourceContextPicker({
  disabled,
  onPick,
  placeholder = "Browse source fields",
  extraSearchParams,
}: {
  disabled?: boolean;
  onPick: (path: string) => void;
  placeholder?: string;
  extraSearchParams?: Record<string, string>;
}) {
  return (
    <PathPicker<EDISourceContextField>
      link="/edi/source-context/fields/select-options/"
      label="Source Fields"
      disabled={disabled}
      placeholder={placeholder}
      getOptionValue={(field) => field.path}
      getDisplayValue={sourceContextFieldDisplayText}
      getPickValue={toSourcePathReference}
      extraSearchParams={{
        status: "Active",
        ...extraSearchParams,
      }}
      renderOption={(field) => (
        <OptionStack primary={sourceContextFieldDisplayText(field)} secondary={field.description} />
      )}
      onPick={onPick}
    />
  );
}

function PartnerSettingPicker({
  disabled,
  onPick,
  placeholder = "Browse partner settings",
  transactionSet,
  direction,
  documentTypeId,
  x12Version,
  getPickValue = toPartnerSettingPath,
}: {
  disabled?: boolean;
  onPick: (path: string) => void;
  placeholder?: string;
  transactionSet?: string;
  direction?: string;
  documentTypeId?: string;
  x12Version?: string;
  getPickValue?: (field: EDIPartnerSettingField) => string;
}) {
  return (
    <PathPicker<EDIPartnerSettingField>
      link="/edi/partner-settings/fields/select-options/"
      label="Partner Settings"
      disabled={disabled}
      placeholder={placeholder}
      getOptionValue={(field) => field.path}
      getDisplayValue={partnerSettingFieldDisplayText}
      getPickValue={getPickValue}
      extraSearchParams={{
        status: "Active",
        ...(transactionSet ? { transactionSet } : {}),
        ...(direction ? { direction } : {}),
        ...(documentTypeId ? { documentTypeId } : {}),
        ...(x12Version ? { x12Version } : {}),
      }}
      renderOption={(field) => (
        <OptionStack
          primary={partnerSettingFieldDisplayText(field)}
          secondary={field.description ?? field.groupKey ?? undefined}
        />
      )}
      onPick={onPick}
    />
  );
}

function PathPicker<TOption>({
  link,
  label,
  disabled,
  placeholder,
  renderOption,
  getOptionValue,
  getDisplayValue,
  getPickValue,
  extraSearchParams,
  onPick,
}: {
  link: SELECT_OPTIONS_ENDPOINTS;
  label: string;
  disabled?: boolean;
  placeholder: string;
  renderOption: (option: TOption) => React.ReactNode;
  getOptionValue: (option: TOption) => string | number;
  getDisplayValue: (option: TOption) => React.ReactNode;
  getPickValue: (option: TOption) => string;
  extraSearchParams?: Record<string, string>;
  onPick: (path: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [selectedOption, setSelectedOption] = useState<TOption | null>(null);
  const listboxId = useId();

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <AutocompleteTrigger
            open={open}
            disabled={!!disabled}
            triggerClassName="font-mono"
            clearable={false}
            currentValue=""
            selectedOption={selectedOption}
            getDisplayValue={getDisplayValue}
            placeholder={placeholder}
            handleClear={() => setSelectedOption(null)}
            listboxId={listboxId}
          />
        }
      />
      <PopoverContent sideOffset={7} className="dark w-(--anchor-width) rounded-md p-0">
        <AutocompleteCommandContent<TOption>
          open={open}
          link={link}
          preload={false}
          label={label}
          getOptionValue={getOptionValue}
          renderOption={renderOption}
          setOpen={setOpen}
          setSelectedOption={setSelectedOption}
          selectedOption={selectedOption}
          onOptionChange={(option) => {
            if (option) onPick(getPickValue(option));
          }}
          onChange={() => undefined}
          clearable={false}
          value=""
          extraSearchParams={extraSearchParams}
          initialLimit={20}
          listboxId={listboxId}
        />
      </PopoverContent>
    </Popover>
  );
}

function PathInput({
  label,
  value,
  onChange,
  disabled,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  return (
    <FieldWrapper label={label}>
      <div className="relative">
        <SearchIcon className="absolute top-1/2 left-2 size-3 -translate-y-1/2 text-muted-foreground" />
        <input
          value={value}
          disabled={disabled}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
          className="h-8 w-full rounded-md border border-input bg-background px-7 text-sm outline-none focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/30 disabled:opacity-50"
        />
      </div>
    </FieldWrapper>
  );
}

function OptionStack({ primary, secondary }: { primary: ReactNode; secondary?: ReactNode }) {
  return (
    <div className="flex size-full min-w-0 flex-col items-start pr-4">
      <span className="w-full truncate">{primary}</span>
      {secondary ? (
        <span className="w-full truncate text-2xs text-muted-foreground">{secondary}</span>
      ) : null}
    </div>
  );
}
