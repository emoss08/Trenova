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

import { ColorField } from "@/components/common/fields/color-field";
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
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { locationCategorySchema as formSchema } from "@/lib/validations/location";
import { LocationCategoryFormValues as FormValues } from "@/types/location";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import {
  Control,
  UseFormSetValue,
  UseFormWatch,
  useForm,
} from "react-hook-form";

export function LCForm({
  control,
}: {
  control: Control<FormValues>;
  watch: UseFormWatch<FormValues>;
  setValue: UseFormSetValue<FormValues>;
}) {
  return (
    <div className="flex items-center justify-center">
      <div className="grid gap-2 mb-2 content-stretch justify-items-center min-w-full">
        <div className="w-full max-w-md">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Name for Location Category"
          />
        </div>
        <div className="grid w-full max-w-md">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Location Category"
          />
        </div>
        <div className="grid w-full max-w-md">
          <ColorField
            name="color"
            label="color"
            control={control}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Color (Hex)"
            description="Color Code of the Location Category"
          />
        </div>
      </div>
    </div>
  );
}

export function LCTableSheet({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit, setValue, watch } = useForm<FormValues>(
    {
      resolver: yupResolver(formSchema),
      defaultValues: {
        name: "",
        description: "",
        color: "",
      },
    },
  );

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/location_categories/",
      successMessage: "Location Category created successfully.",
      queryKeysToInvalidate: ["location-categories-table-data"],
      additionalInvalidateQueries: ["locationCategories"],
      closeModal: true,
      errorMessage: "Failed to create new location category.",
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
          <DialogTitle>Create New Location Category</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Location Category.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <LCForm control={control} watch={watch} setValue={setValue} />
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
