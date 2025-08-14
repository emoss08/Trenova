/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { handleMutationError } from "@/hooks/use-api-mutation";
import useDebouncedEffect from "@/hooks/use-debounce-effect";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";

export default function ShipmentGeneralInformation() {
  return (
    <ShipmentGeneralInformationInner>
      <GeneralInformationFormGroup>
        <BOLField />
        <TemperatureFields />
        <WeightAndPiecesFields />
      </GeneralInformationFormGroup>
    </ShipmentGeneralInformationInner>
  );
}

function ShipmentGeneralInformationInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium font-table">General Information</h3>
      {children}
    </div>
  );
}

function GeneralInformationFormGroup({
  children,
}: {
  children: React.ReactNode;
}) {
  return <FormGroup cols={2}>{children}</FormGroup>;
}

function BOLField() {
  const { control, getValues, setError, getFieldState } =
    useFormContext<ShipmentSchema>();

  const { isDirty: isBolDirty } = getFieldState("bol");

  const bolChanged = useWatch({
    control,
    name: "bol",
  });

  const { data: shipmentControl, isLoading: isShipmentControlLoading } =
    useQuery({
      ...queries.organization.getShipmentControl(),
    });

  const { mutate: checkBols } = useMutation({
    mutationFn: async (bol: string) => {
      return await api.shipments.checkForDuplicateBOLs(bol, getValues("id"));
    },
    onError: (error) => {
      // Standard error handling
      handleMutationError<ShipmentSchema>({
        error: error as APIError,
        setFormError: setError,
        resourceName: "BOL",
      });
    },
  });

  useDebouncedEffect(
    () => {
      if (isShipmentControlLoading || !shipmentControl?.checkForDuplicateBols) {
        return;
      }

      if (bolChanged && isBolDirty) {
        checkBols(bolChanged);
      }
    },
    [bolChanged],
    1000, // * 1 second delay
  );

  return (
    <FormControl cols="full">
      <InputField
        control={control}
        name="bol"
        label="BOL"
        rules={{ required: true }}
        description="The BOL is the bill of lading number for the shipment."
        placeholder="Enter BOL"
        maxLength={100}
      />
    </FormControl>
  );
}

function TemperatureFields() {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <>
      <FormControl>
        <InputField
          control={control}
          name="temperatureMin"
          description="The minimum temperature for the shipment."
          label="Temperature Min"
          placeholder="Enter Temperature Min"
          sideText="°F"
          type="number"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="temperatureMax"
          label="Temperature Max"
          description="The maximum temperature for the shipment."
          placeholder="Enter Temperature Max"
          sideText="°F"
          type="number"
        />
      </FormControl>
    </>
  );
}

function WeightAndPiecesFields() {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <>
      <FormControl>
        <NumberField
          inputMode="numeric"
          control={control}
          name="weight"
          label="Total Weight"
          readOnly
          tabIndex={-1}
          description="The total weight of the shipment."
          placeholder="Enter Weight"
          sideText="lbs"
        />
      </FormControl>
      <FormControl>
        <NumberField
          inputMode="numeric"
          control={control}
          name="pieces"
          readOnly
          tabIndex={-1}
          label="Total Pieces"
          description="The total number of pieces in the shipment."
          placeholder="Enter Pieces"
        />
      </FormControl>
    </>
  );
}
