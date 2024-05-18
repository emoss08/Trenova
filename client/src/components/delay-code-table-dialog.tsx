import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { delayCodeSchema } from "@/lib/validations/DispatchSchema";
import { type DelayCodeFormValues } from "@/types/dispatch";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { GradientPicker } from "./common/fields/color-field";
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

export function DelayCodeForm({
  control,
}: {
  control: Control<DelayCodeFormValues>;
}) {
  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Delay code"
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
            description="Unique Code for the Delay Code"
            maxLength={20}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Delay Code"
          />
        </FormControl>
        <FormControl className="col-span-full min-h-0">
          <CheckboxInput
            control={control}
            label="Fault of Carrier or Driver?"
            name="fCarrierOrDriver"
            description="Indicates if the delay is the fault of the carrier or driver."
          />
        </FormControl>
        <FormControl className="col-span-full">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Delay Code"
            control={control}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function DelayCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<DelayCodeFormValues>({
    resolver: yupResolver(delayCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      fCarrierOrDriver: false,
      color: "",
    },
  });

  const mutation = useCustomMutation<DelayCodeFormValues>(control, {
    method: "POST",
    path: "/delay-codes/",
    successMessage: "Delay Code created successfully.",
    queryKeysToInvalidate: "delayCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new delay code.",
  });

  const onSubmit = (values: DelayCodeFormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Delay Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Delay Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <DelayCodeForm control={control} />
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
