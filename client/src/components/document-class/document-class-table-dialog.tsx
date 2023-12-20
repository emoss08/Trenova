/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

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
import { documentClassSchema } from "@/lib/validations/BillingSchema";
import { DocumentClassificationFormValues as FormValues } from "@/types/billing";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";

export function DocumentClassForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  return (
    <Form>
      <FormGroup className="grid md:grid-cols-1 lg:grid-cols-1 xl:grid-cols-1 gap-6">
        <FormControl className="w-full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            autoComplete="name"
            description="Enter a unique identifier for the document classification."
            maxLength={10}
          />
        </FormControl>
        <FormControl className="w-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the document classification."
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
      name: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/document_classifications/",
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
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
            >
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
