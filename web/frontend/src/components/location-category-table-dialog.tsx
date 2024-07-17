/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
