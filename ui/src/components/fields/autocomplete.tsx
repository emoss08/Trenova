/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import {
  AutocompleteFieldProps,
  BaseAutocompleteFieldProps,
} from "@/types/fields";
import { useCallback, useEffect, useState } from "react";
import { Controller, FieldValues } from "react-hook-form";
import { AutocompleteCommandContent } from "./autocompete/autocomplete-content";
import { AutocompleteTrigger } from "./autocompete/autocomplete-input";
import { FieldWrapper } from "./field-components";

export interface Option {
  value: string;
  label: string;
  disabled?: boolean;
  description?: string;
  icon?: React.ReactNode;
}

export function Autocomplete<TOption, TForm extends FieldValues>({
  link,
  preload = false,
  renderOption,
  getOptionValue,
  getDisplayValue,
  label,
  placeholder = "Select...",
  value,
  onChange,
  disabled = false,
  className,
  triggerClassName,
  noResultsMessage,
  onOptionChange,
  isInvalid,
  clearable = false,
  extraSearchParams,
  popoutLink,
}: BaseAutocompleteFieldProps<TOption, TForm>) {
  const [open, setOpen] = useState(false);
  const [selectedOption, setSelectedOption] = useState<TOption | null>(null);

  const handleClear = useCallback(() => {
    onChange("");
    setSelectedOption(null);
  }, [onChange]);

  useEffect(() => {
    if (selectedOption && value !== getOptionValue(selectedOption)) {
      setSelectedOption(null);
    }
  }, [value, selectedOption, getOptionValue]);

  return (
    <div className="relative">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <AutocompleteTrigger
            open={open}
            disabled={disabled}
            isInvalid={isInvalid}
            triggerClassName={triggerClassName}
            clearable={clearable}
            value={value}
            selectedOption={selectedOption}
            getDisplayValue={getDisplayValue}
            placeholder={placeholder}
            handleClear={handleClear}
            setSelectedOption={setSelectedOption}
            link={link}
          />
        </PopoverTrigger>
        <PopoverContent
          sideOffset={7}
          className={cn(
            "p-0 rounded-md w-[var(--radix-popover-trigger-width)]",
            className,
          )}
        >
          <AutocompleteCommandContent
            open={open}
            link={link}
            preload={preload}
            label={label}
            getOptionValue={getOptionValue}
            renderOption={renderOption}
            setOpen={setOpen}
            setSelectedOption={setSelectedOption}
            onOptionChange={onOptionChange}
            onChange={onChange}
            clearable={clearable}
            value={value}
            noResultsMessage={noResultsMessage}
            extraSearchParams={extraSearchParams}
            popoutLink={popoutLink}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

export function AutocompleteField<TOption, TForm extends FieldValues>({
  label,
  name,
  control,
  rules,
  className,
  link,
  preload,
  renderOption,
  description,
  getOptionValue,
  getDisplayValue,
  onOptionChange,
  clearable,
  extraSearchParams,
  placeholder,
  ...props
}: AutocompleteFieldProps<TOption, TForm>) {
  return (
    <Controller<TForm>
      name={name}
      control={control}
      rules={rules}
      render={({ field: { onChange, value, disabled }, fieldState }) => {
        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <Autocomplete<TOption, TForm>
              link={link}
              preload={preload}
              renderOption={renderOption}
              getOptionValue={getOptionValue}
              getDisplayValue={getDisplayValue}
              onOptionChange={onOptionChange}
              clearable={clearable}
              extraSearchParams={extraSearchParams}
              label={label}
              placeholder={placeholder}
              value={value}
              onChange={onChange}
              disabled={disabled}
              isInvalid={fieldState.invalid}
              {...props}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
