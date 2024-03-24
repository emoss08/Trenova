/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import { SelectInput } from "@/components/common/fields/select-input";
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
import { severityChoices, statusChoices } from "@/lib/choices";
import { commentTypeSchema } from "@/lib/validations/DispatchSchema";
import { CommentTypeFormValues as FormValues } from "@/types/dispatch";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";

export function CommentTypeForm({ control }: { control: Control<FormValues> }) {
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
            description="Status of the Comment Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="severity"
            rules={{ required: true }}
            control={control}
            label="Severity"
            options={severityChoices}
            placeholder="Select Severity Level"
            description="Severity level of the Comment Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            maxLength={10}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            autoComplete="name"
            description="Unique name for the Comment Type"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Comment Type"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function CommentTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(commentTypeSchema),
    defaultValues: {
      status: "A",
      name: "",
      severity: "Low",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/comment-types/",
      successMessage: "Comment Type created successfully.",
      queryKeysToInvalidate: ["comment-types-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new comment type.",
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
          <DialogTitle>Create New Comment Type</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Comment Type.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <CommentTypeForm control={control} />
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
