/**
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



import { GradientPicker } from "@/components/common/fields/color-field";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { LocationCategorySchema as formSchema } from "@/lib/validations/LocationSchema";
import { type LocationCategoryFormValues as FormValues } from "@/types/location";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
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

export function LCForm({ control }: { control: Control<FormValues> }) {
  return (
    <div className="flex items-center justify-center">
      <div className="mb-2 grid min-w-full content-stretch justify-items-center gap-2">
        <div className="w-full max-w-md">
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Cateogry Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Official Name for Location Category"
          />
        </div>
        <div className="grid w-full max-w-md">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Detailed Description of the Location Category"
          />
        </div>
        <div className="grid w-full max-w-md">
          <GradientPicker
            name="color"
            label="Color"
            description="Color Code of the Location Category"
            control={control}
          />
        </div>
      </div>
    </div>
  );
}

export function LocationCategoryDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(formSchema),
    defaultValues: {
      name: "",
      description: "",
      color: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/location-categories/",
    successMessage: "Location Category created successfully.",
    queryKeysToInvalidate: "locationCategories",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new location category.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Location Category</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Location Category.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <LCForm control={control} />
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
