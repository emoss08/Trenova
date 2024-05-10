import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { documentClassSchema } from "@/lib/validations/BillingSchema";
import { type DocumentClassificationFormValues as FormValues } from "@/types/billing";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { GradientPicker } from "./common/fields/color-field";
import { SelectInput } from "./common/fields/select-input";
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

export function DocumentClassForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  return (
    <Form>
      <FormGroup className="lg:grid-cols-2">
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
            placeholder="Code"
            description="Enter a unique identifier for the document classification."
            maxLength={10}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the document classification."
          />
        </FormControl>
        <FormControl className="col-span-full">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Location Category"
            control={control}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function DocumentClassDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(documentClassSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      color: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/document-classifications/",
    successMessage: "Document Classification created successfully.",
    queryKeysToInvalidate: "documentClassifications",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new document classification.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Document Classification</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Document
          Classification.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <DocumentClassForm control={control} />
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
