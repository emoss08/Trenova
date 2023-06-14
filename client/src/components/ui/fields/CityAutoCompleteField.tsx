/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import React, { useRef, useState } from "react";
import { createStyles } from "@mantine/styles";
import { ValidatedTextInputProps } from "@/types/fields";
import { Autocomplete, Loader } from "@mantine/core";
import { stateData } from "@/components/ui/fields/StateSelect";
import { IconAlertTriangle } from "@tabler/icons-react";

interface CityAutoCompleteField<TFormValues extends object>
  extends ValidatedTextInputProps<TFormValues> {
  stateSelection: string;
}

const useStyles = createStyles((theme) => {
  return {
    invalid: {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.fn.rgba(theme.colors.red[8], 0.15)
          : theme.colors.red[0],
    },
    invalidIcon: {
      color: theme.colors.red[theme.colorScheme === "dark" ? 7 : 6],
    },
  };
});

export const CityAutoCompleteField = <TFormValues extends object>({
  form,
  stateSelection,
  name,
  ...rest
}: CityAutoCompleteField<TFormValues>) => {
  const { classes } = useStyles();
  const error = form.errors[name as string];
  const timeoutRef = useRef<number>(-1);
  const [loading, setLoading] = useState<boolean>(false);
  const [data, setData] = useState<string[]>([]);

  const getCity = async () => {
    const selectedState = stateData.find(
      (state) => state.value === stateSelection
    );
    const stateLabel = selectedState ? selectedState.label : "";
    const response = await fetch(
      "https://countriesnow.space/api/v0.1/countries/state/cities",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          country: "United States",
          state: stateLabel,
        }),
      }
    );
    return await response.json();
  };

  const handleChange = (val: string) => {
    window.clearTimeout(timeoutRef.current);
    form.setFieldValue(name as string, val); // Update form value for the city
    setData([]);

    if (val.trim().length === 0 || val.includes("@")) {
      setLoading(false);
    } else {
      setLoading(true);
      timeoutRef.current = window.setTimeout(() => {
        setLoading(false);
        getCity().then((res) => {
          setData(res.data);
        });
      }, 500);
    }
  };

  return (
    <Autocomplete
      {...rest}
      {...form.getInputProps(name as string)}
      data={data ?? []}
      error={error}
      classNames={{
        input: error ? classes.invalid : "",
      }}
      onChange={handleChange}
      rightSection={
        loading ? (
          <Loader size={24} />
        ) : (
          error && (
            <IconAlertTriangle
              stroke={1.5}
              size="1.1rem"
              className={classes.invalidIcon}
            />
          )
        )
      }
    />
  );
};
