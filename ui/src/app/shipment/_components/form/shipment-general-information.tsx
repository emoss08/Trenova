import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { checkForDuplicateBOLs } from "@/services/shipment";
import { APIError } from "@/types/errors";
import { useMutation, useQuery } from "@tanstack/react-query";
import debounce from "lodash/debounce";
import { useEffect, useRef } from "react";
import { useFormContext } from "react-hook-form";

export default function ShipmentGeneralInformation() {
  const { control, watch, getValues, setError, getFieldState } =
    useFormContext();

  const { isDirty } = getFieldState("bol");

  // Create refs for state that shouldn't trigger re-renders
  const debouncedFnRef = useRef<ReturnType<typeof debounce> | null>(null);
  const lastCheckedBolRef = useRef<string | null>(null);

  // If the user fills out the BOL, and tabs to the next field
  // We need to send a request to the server to check if the BOL is valid
  // But only if the Shipment Control has duplicate BOLs checking enabled
  const { data: shipmentControl } = useQuery({
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

  // Create the debounced function once when component mounts
  useEffect(() => {
    debouncedFnRef.current = debounce((bol: string) => {
      if (bol && bol !== lastCheckedBolRef.current) {
        lastCheckedBolRef.current = bol;
        checkBols(bol);
      }
    }, 500);

    return () => {
      debouncedFnRef.current?.cancel();
    };
  }, [checkBols]);

  // Use watch subscription to observe BOL changes
  useEffect(() => {
    // Only set up subscription if duplicate BOL checking is enabled
    if (!shipmentControl?.checkForDuplicateBOLs) {
      return;
    }

    // Subscribe to BOL field changes
    const subscription = watch((formValues, { name }) => {
      // Only react to BOL field changes
      if (name === "bol" || name === undefined) {
        const bol = formValues.bol as string | undefined;

        if (bol && isDirty && debouncedFnRef.current) {
          debouncedFnRef.current(bol);
        }
      }
    });

    // Cleanup subscription when component unmounts
    return () => subscription.unsubscribe();
  }, [watch, isDirty, shipmentControl?.checkForDuplicateBOLs]);

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">General Information</h3>
      <FormGroup cols={2}>
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
      </FormGroup>
    </div>
  );
}
