import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { documentClassSchema } from "@/lib/validations/BillingSchema";
import { type DocumentClassificationFormValues as FormValues } from "@/types/billing";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { GradientPicker } from "./common/fields/color-field";
import { SelectInput } from "./common/fields/select-input";

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
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(documentClassSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
      color: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/document-classifications/",
      successMessage: "Document Classification created successfully.",
      queryKeysToInvalidate: ["document-classification-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new document classification.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Document Classification</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Document
          Classification.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DocumentClassForm control={control} />
          <DialogFooter className="mt-6">
            <Button type="submit" isLoading={isSubmitting}>
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
