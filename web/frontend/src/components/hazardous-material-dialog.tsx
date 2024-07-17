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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  hazardousClassChoices,
  packingGroupChoices,
  statusChoices,
} from "@/lib/choices";
import { usePopoutWindow } from "@/lib/popout-window-hook";
import { hazardousMaterialSchema } from "@/lib/validations/CommoditiesSchema";
import { type HazardousMaterialFormValues as FormValues } from "@/types/commodities";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
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

export function HazardousMaterialForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { t } = useTranslation(["pages.hazardousmaterial", "common"]);

  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label={t("fields.status.label")}
            options={statusChoices}
            placeholder={t("fields.status.placeholder")}
            description={t("fields.status.description")}
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label={t("fields.name.label")}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder={t("fields.name.placeholder")}
            description={t("fields.name.description")}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label={t("fields.description.label")}
            placeholder={t("fields.description.placeholder")}
            description={t("fields.description.description")}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="hazardClass"
            rules={{ required: true }}
            control={control}
            label={t("fields.hazardClass.label")}
            options={hazardousClassChoices}
            placeholder={t("fields.hazardClass.placeholder")}
            description={t("fields.hazardClass.description")}
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="packingGroup"
            control={control}
            label={t("fields.packingGroup.label")}
            options={packingGroupChoices}
            placeholder={t("fields.packingGroup.placeholder")}
            description={t("fields.packingGroup.description")}
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <InputField
            control={control}
            name="ergNumber"
            label={t("fields.ergNumber.label")}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder={t("fields.ergNumber.placeholder")}
            description={t("fields.ergNumber.description")}
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="properShippingName"
            control={control}
            label={t("fields.properShippingName.label")}
            placeholder={t("fields.properShippingName.placeholder")}
            description={t("fields.properShippingName.description")}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function HazardousMaterialDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const { t } = useTranslation(["pages.hazardousmaterial", "common"]);
  const { isPopout, closePopout } = usePopoutWindow();
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(hazardousMaterialSchema),
    defaultValues: {
      status: "Active",
      name: "",
      description: undefined,
      hazardClass: undefined,
      packingGroup: undefined,
      ergNumber: undefined,
      properShippingName: undefined,
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/hazardous-materials/",
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "hazardousMaterials",
    closeModal: true,
    reset,
    errorMessage: t("formErrorMessage"),
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{t("title")}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>{t("description")}</CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <HazardousMaterialForm control={control} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={mutation.isPending}>
                Save {isPopout ? "and Close" : "Changes"}
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
