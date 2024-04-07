import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { codeTypeChoices, statusChoices } from "@/lib/choices";
import { reasonCodeSchema } from "@/lib/validations/ShipmentSchema";
import { type ReasonCodeFormValues as FormValues } from "@/types/shipment";
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

export function ReasonCodeForm({ control }: { control: Control<FormValues> }) {
  return (
    <div className="flex-1 overflow-y-visible">
      <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <div className="grid w-full max-w-sm items-center gap-0.5">
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Reason Code"
            isClearable={false}
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
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
            description="Unique code for the Reason Code"
          />
        </div>
      </div>
      <div className="my-2">
        <TextareaField
          name="description"
          rules={{ required: true }}
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Reason Code"
        />
      </div>
      <div className="my-2">
        <SelectInput
          name="codeType"
          rules={{ required: true }}
          control={control}
          label="Code Type"
          options={codeTypeChoices}
          placeholder="Select Code Type"
          description="Code Type of the Reason Code"
          isClearable={false}
        />
      </div>
    </div>
  );
}

export function ReasonCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(reasonCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      codeType: "Voided",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/reason-codes/",
      successMessage: "Reason Codes created successfully.",
      queryKeysToInvalidate: ["reason-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new reason code.",
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
          <CredenzaTitle>Create New Reason Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Reason Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <ReasonCodeForm control={control} />
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
