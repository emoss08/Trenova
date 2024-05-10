import { DatepickerField } from "@/components/common/fields/date-picker";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  useEquipManufacturers,
  useEquipmentTypes,
  useFleetCodes,
  useUSStates,
} from "@/hooks/useQueries";
import { cn } from "@/lib/utils";
import { trailerSchema } from "@/lib/validations/EquipmentSchema";
import {
  equipmentStatusChoices,
  type TrailerFormValues as FormValues,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm, type Control } from "react-hook-form";
import { Form, FormControl, FormGroup } from "./ui/form";
import { Separator } from "./ui/separator";

export function TrailerForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const { selectEquipmentType, isLoading, isError } = useEquipmentTypes(100);

  const {
    selectEquipManufacturers,
    isLoading: isEquipManuLoading,
    isError: isEquipManuError,
  } = useEquipManufacturers(open);

  const {
    selectFleetCodes,
    isError: isFleetCodeError,
    isLoading: isFleetCodesLoading,
  } = useFleetCodes(open);

  const {
    selectUSStates,
    isError: isStateError,
    isLoading: isStatesLoading,
  } = useUSStates(open);

  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={equipmentStatusChoices}
            placeholder="Select Status"
            description="Select the current operational status of the trailer."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            description="Enter a unique identifier or code for the trailer."
          />
        </FormControl>
      </FormGroup>
      <Separator />
      <FormGroup>
        <FormControl>
          <SelectInput
            name="equipmentTypeId"
            rules={{ required: true }}
            control={control}
            label="Equip. Type"
            options={selectEquipmentType}
            isFetchError={isError}
            isLoading={isLoading}
            placeholder="Select Equip. Type"
            description="Select the equipment type of the trailer, to categorize it based on its functionality and usage."
            isClearable={false}
            popoutLink="/equipment/equipment-types/"
            hasPopoutWindow
            popoutLinkLabel="Equipment Type"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="equipmentManufacturerId"
            control={control}
            label="Manufacturer"
            options={selectEquipManufacturers}
            isFetchError={isEquipManuError}
            isLoading={isEquipManuLoading}
            placeholder="Select Manufacturer"
            description="Select the manufacturer of the trailer, to categorize it based on its functionality and usage."
            isClearable
            hasPopoutWindow
            popoutLink="/equipment/equipment-manufacturers/"
            popoutLinkLabel="Equipment Manufacturer"
          />
        </FormControl>
        <FormControl>
          <InputField
            name="model"
            control={control}
            label="Model"
            placeholder="Model"
            autoCapitalize="none"
            autoCorrect="off"
            description="Indicate the model of the trailer as provided by the manufacturer."
          />
        </FormControl>
        <FormControl>
          <InputField
            type="number"
            name="year"
            control={control}
            label="Year"
            placeholder="Year"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the year of manufacture of the trailer."
            maxLength={4}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="vin"
            control={control}
            label="Vin Number"
            placeholder="Vin Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Input the Vehicle Identification Number."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="fleetCodeId"
            control={control}
            rules={{ required: true }}
            label="Fleet Code"
            options={selectFleetCodes}
            isFetchError={isFleetCodeError}
            isLoading={isFleetCodesLoading}
            placeholder="Select Fleet Code"
            description="Select the code that identifies the trailer within your fleet."
            hasPopoutWindow
            popoutLink="/dispatch/fleet-codes/"
            popoutLinkLabel="Fleet Code"
          />
        </FormControl>
        <FormControl>
          <InputField
            name="licensePlateNumber"
            control={control}
            label="License Plate #"
            placeholder="License Plate Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the license plate number of the trailer, crucial for legal identification and registration."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="stateId"
            control={control}
            label="License Plate State"
            options={selectUSStates}
            isFetchError={isStateError}
            isLoading={isStatesLoading}
            isClearable
            placeholder="Select License Plate State"
            description="Select the state of registration of the trailer's license plate."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="lastInspectionDate"
            control={control}
            label="Last Inspection"
            placeholder="Last Inspection Date"
            description="Input the date of the last inspection the trailer underwent."
          />
        </FormControl>
        <FormControl>
          <InputField
            name="registrationNumber"
            control={control}
            label="Registration #"
            placeholder="Registration Number"
            autoCapitalize="none"
            autoCorrect="off"
            description="Enter the registration number assigned to the trailer by the motor vehicle department."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="registrationStateId"
            control={control}
            label="Registration State"
            options={selectUSStates}
            isFetchError={isStateError}
            isLoading={isStatesLoading}
            placeholder="Select Registration State"
            description="Select the state where the trailer is registered."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="registrationExpirationDate"
            control={control}
            placeholder="Registration Expiration Date"
            label="Registration Expiration"
            description="Choose the date when the current registration of the trailer expires."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function TrailerDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(trailerSchema),
    defaultValues: {
      code: "",
      status: "Available",
      equipmentTypeId: undefined,
      equipmentManufacturerId: undefined,
      model: "",
      year: undefined,
      vin: "",
      fleetCodeId: undefined,
      licensePlateNumber: "",
      lastInspectionDate: undefined,
      stateId: undefined,
      registrationNumber: "",
      registrationStateId: undefined,
      registrationExpirationDate: undefined,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/trailers/",
    successMessage: "Trailer created successfully.",
    queryKeysToInvalidate: "trailers",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new trailer.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New Trailer</SheetTitle>
          <SheetDescription>
            Use this form to add a new trailer to the system.
          </SheetDescription>
        </SheetHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <TrailerForm control={control} open={open} />
          <SheetFooter className="mb-12">
            <Button
              type="reset"
              variant="secondary"
              onClick={() => onOpenChange(false)}
              className="w-full"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={mutation.isPending}
              className="w-full"
            >
              Save
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
