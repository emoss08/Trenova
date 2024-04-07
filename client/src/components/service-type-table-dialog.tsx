import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { serviceTypeSchema } from "@/lib/validations/ShipmentSchema";
import { type ServiceTypeFormValues as FormValues } from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
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

export function ServiceTypeForm({ control }: { control: Control<FormValues> }) {
  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Service Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            maxLength={10}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            autoComplete="code"
            description="Code for the Service Type"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Service Type"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function ServiceTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(serviceTypeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/service-types/",
      successMessage: "Service Type created successfully.",
      queryKeysToInvalidate: ["service-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new service type.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Service Type</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Service Type.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <ServiceTypeForm control={control} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={isSubmitting}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
