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

import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { hazardousClassChoices } from "@/lib/choices";
import { useHazmatSegRulesForm } from "@/lib/validations/ShipmentSchema";
import { type HazardousMaterialSegregationRuleFormValues as FormValues } from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { FormProvider, useFormContext } from "react-hook-form";
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

const segregationTypeChoices = [
  {
    value: "AllowedWithConditions",
    label: "Allowed With Conditions",
    color: "#15803d",
  },
  { value: "NotAllowed", label: "Not Allowed", color: "#b91c1c" },
];

export function HazmatSegRulesForm() {
  const { control } = useFormContext<FormValues>();

  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="classA"
            rules={{ required: true }}
            control={control}
            label="Class A"
            options={hazardousClassChoices}
            placeholder="Select Class A"
            description="First hazardous material class or division."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="classB"
            rules={{ required: true }}
            control={control}
            label="Class B"
            options={hazardousClassChoices}
            placeholder="Select Class B"
            description="Second hazardous material class or division."
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-2">
          <SelectInput
            name="segregationType"
            rules={{ required: true }}
            control={control}
            label="Segregation Type"
            options={segregationTypeChoices}
            placeholder="Select Segregation Type"
            description="Indicates if the materials are allowed to be transported together."
            isClearable={false}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function HazmatSegRulesDialog({ onOpenChange, open }: TableSheetProps) {
  const { hazmatSegRulesForm } = useHazmatSegRulesForm();
  const { control, handleSubmit, reset } = hazmatSegRulesForm;

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/hazardous-material-segregations/",
    successMessage: "Hazardous material segregation rule created successfully.",
    queryKeysToInvalidate: "hazardousMaterialsSegregations",
    closeModal: true,
    reset,
    errorMessage: "Failed to create hazardous material segregation rule.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Hazmat Seg. Rule</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Hazmat Segregation
          Rule.
        </CredenzaDescription>
        <CredenzaBody>
          <FormProvider {...hazmatSegRulesForm}>
            <form onSubmit={handleSubmit(onSubmit)}>
              <HazmatSegRulesForm />
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
          </FormProvider>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
