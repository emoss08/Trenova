import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useHazardousMaterial } from "@/hooks/useQueries";
import {
  UnitOfMeasureChoices,
  statusChoices,
  yesAndNoBooleanChoices,
} from "@/lib/choices";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { type CommodityFormValues as FormValues } from "@/types/commodities";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useEffect } from "react";
import { Control, useForm } from "react-hook-form";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";
import { Form, FormControl, FormGroup } from "./ui/form";

export function CommodityForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const { selectHazardousMaterials, isLoading, isError } =
    useHazardousMaterial(open);

  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Commodity"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Name for the Commodity"
          />
        </FormControl>
      </FormGroup>
      <div className="my-2 grid w-full items-center gap-0.5">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Commodity"
        />
      </div>
      <FormGroup className="grid gap-2 md:grid-cols-2 lg:grid-cols-2">
        <FormControl>
          <InputField
            name="minTemp"
            control={control}
            label="Min. Temp"
            placeholder="Min. Temp"
            description="Minimum Temperature of the Commodity"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="maxTemp"
            label="Max. Temp"
            placeholder="Max. Temp"
            description="Maximum Temperature of the Commodity"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="hazardousMaterialId"
            control={control}
            label="Hazardous Material"
            options={selectHazardousMaterials}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Hazardous Material"
            description="The Hazardous Material associated with the Commodity"
            isClearable
            hasPopoutWindow
            popoutLink="/shipments/hazardous-materials/"
            popoutLinkLabel="Hazardous Material"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="isHazmat"
            control={control}
            label="Is Hazmat"
            options={yesAndNoBooleanChoices}
            placeholder="Is Hazmat"
            description="Is the Commodity a Hazardous Material?"
            isClearable
          />
        </FormControl>
        <FormControl className="col-span-full">
          <SelectInput
            name="unitOfMeasure"
            control={control}
            label="Unit of Measure"
            options={UnitOfMeasureChoices}
            placeholder="Unit of Measure"
            description="Unit of Measure of the Commodity"
            isClearable
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function CommodityDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, handleSubmit, watch, reset, setValue } = useForm<FormValues>(
    {
      resolver: yupResolver(commoditySchema),
      defaultValues: {
        status: "A",
        name: "",
        description: undefined,
        minTemp: undefined,
        maxTemp: undefined,
        unitOfMeasure: undefined,
        hazardousMaterialId: null,
        isHazmat: false,
      },
    },
  );

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "hazardousMaterialId" && value.hazardousMaterialId) {
        setValue("isHazmat", true);
      } else if (name === "hazardousMaterialId" && !value.hazardousMaterialId) {
        setValue("isHazmat", false);
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/commodities/",
    successMessage: "Commodity created successfully.",
    queryKeysToInvalidate: "commodities",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new commodity.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Commodity</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Commodity.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <CommodityForm control={control} open={open} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={mutation.isPending}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
