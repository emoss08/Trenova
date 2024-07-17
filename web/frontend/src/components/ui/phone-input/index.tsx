/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { FieldDescription } from "@/components/common/fields/components";
import { ErrorMessage } from "@/components/common/fields/error-message";
import { Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { cn } from "@/lib/utils";

// eslint-disable-next-line import/no-named-as-default
import parsePhoneNumberFromString, {
  AsYouType,
  type CarrierCode,
  type CountryCallingCode,
  type CountryCode,
  type E164Number,
  type NationalNumber,
  type NumberType,
} from "libphonenumber-js";
import * as React from "react";
import {
  Controller,
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { countries } from "./countries";
import { useStateHistory } from "./use-state-history";

export type Country = (typeof countries)[number];

export type PhoneData = {
  phoneNumber?: E164Number;
  countryCode?: CountryCode;
  countryCallingCode?: CountryCallingCode;
  carrierCode?: CarrierCode;
  nationalNumber?: NationalNumber;
  internationalNumber?: string;
  possibleCountries?: string;
  isValid?: boolean;
  isPossible?: boolean;
  uri?: string;
  type?: NumberType;
};

interface PhoneInputProps extends React.ComponentPropsWithoutRef<"input"> {
  value?: string;
  description?: string;
  label: string;
  defaultCountry?: CountryCode;
}

export function getPhoneData(phone: string): PhoneData {
  const asYouType = new AsYouType();
  asYouType.input(phone);
  const number = asYouType.getNumber();
  return {
    phoneNumber: number?.number,
    countryCode: number?.country,
    countryCallingCode: number?.countryCallingCode,
    carrierCode: number?.carrierCode,
    nationalNumber: number?.nationalNumber,
    internationalNumber: number?.formatInternational(),
    possibleCountries: number?.getPossibleCountries().join(", "),
    isValid: number?.isValid(),
    isPossible: number?.isPossible(),
    uri: number?.getURI(),
    type: number?.getType(),
  };
}

type ExtendedPhoneInputProps<T extends FieldValues> = PhoneInputProps &
  UseControllerProps<T>;

export function PhoneInput<T extends FieldValues>({
  value: valueProp,
  defaultCountry = "US",
  ...props
}: ExtendedPhoneInputProps<T>) {
  const { fieldState } = useController(props);

  const { rules, label, name, control, className, description } = props;

  const asYouType = new AsYouType();

  const inputRef = React.useRef<HTMLInputElement>(null);

  const [value, handlers, history] = useStateHistory(valueProp);

  if (value && value.length > 0) {
    defaultCountry =
      parsePhoneNumberFromString(value)?.getPossibleCountries()[0] ||
      defaultCountry;
  }

  const [countryCode, setCountryCode] =
    React.useState<CountryCode>(defaultCountry);

  const selectedCountry = countries.find(
    (country) => country.iso2 === countryCode,
  );

  const initializeDefaultValue = () => {
    if (value) {
      return value;
    }

    return `+${selectedCountry?.phone_code}`;
  };

  const handleOnInput = (event: React.FormEvent<HTMLInputElement>) => {
    asYouType.reset();

    let value = event.currentTarget.value;
    if (!value.startsWith("+")) {
      value = `+${value}`;
    }

    const formattedValue = asYouType.input(value);
    const number = asYouType.getNumber();
    setCountryCode(number?.country || defaultCountry);
    event.currentTarget.value = formattedValue;
    handlers.set(formattedValue);
  };

  const handleOnPaste = (event: React.ClipboardEvent<HTMLInputElement>) => {
    event.preventDefault();
    asYouType.reset();

    const clipboardData = event.clipboardData;

    if (clipboardData) {
      const pastedData = clipboardData.getData("text/plain");
      const formattedValue = asYouType.input(pastedData);
      const number = asYouType.getNumber();
      setCountryCode(number?.country || defaultCountry);
      event.currentTarget.value = formattedValue;
      handlers.set(formattedValue);
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "z") {
      handlers.back();
      if (
        inputRef.current &&
        history.current > 0 &&
        history.history[history.current - 1] !== undefined
      ) {
        event.preventDefault();
        inputRef.current.value = history.history[history.current - 1] || "";
      }
    }
  };

  return (
    <>
      {label && (
        <Label
          className={cn("text-sm font-medium", rules?.required && "required")}
          htmlFor={props.id}
        >
          {label}
        </Label>
      )}
      <div className="relative">
        <Controller
          name={name}
          control={control}
          render={({ field }) => (
            <Input
              {...field}
              ref={inputRef}
              type="text"
              pattern="^(\+)?[0-9\s]*$"
              placeholder="Phone"
              defaultValue={initializeDefaultValue()}
              onInput={handleOnInput}
              onPaste={handleOnPaste}
              onKeyDown={handleKeyDown}
              className={cn(
                fieldState.invalid &&
                  "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
                className,
              )}
              {...props}
            />
          )}
        />
        {fieldState.invalid ? (
          <ErrorMessage formError={fieldState.error?.message} />
        ) : (
          <FieldDescription description={description!} />
        )}
      </div>
    </>
  );
}
