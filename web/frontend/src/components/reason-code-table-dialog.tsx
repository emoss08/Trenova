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
  const { control, handleSubmit, reset } = useForm<FormValues>({
    resolver: yupResolver(reasonCodeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      codeType: "Voided",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/reason-codes/",
    successMessage: "Reason Codes created successfully.",
    queryKeysToInvalidate: "reasonCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new reason code.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
    reset();
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
