import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { checkForDuplicateBOLs } from "@/services/shipment";
import { APIError } from "@/types/errors";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useDebouncedEffect } from "@wojtekmaj/react-hooks";
import { useFormContext, useWatch } from "react-hook-form";

export default function ShipmentGeneralInformation() {
  return (
    <ShipmentGeneralInformationInner>
      <GeneralInformationFormGroup>
        <BOLField />
        <TemperatureFields />
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
      <h3 className="text-sm font-medium">General Information</h3>
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
      return await checkForDuplicateBOLs(bol, getValues("id"));
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
          label="Temperature Min"
          type="number"
          description="The minimum temperature for the shipment."
          placeholder="Enter Temperature Min"
          sideText="°F"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="temperatureMax"
          label="Temperature Max"
          type="number"
          description="The maximum temperature for the shipment."
          placeholder="Enter Temperature Max"
          sideText="°F"
        />
      </FormControl>
    </>
  );
}
