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
import { useCustomMutation } from "@/hooks/useCustomMutation";
import {
  hazardousClassChoices,
  packingGroupChoices,
  statusChoices,
} from "@/lib/choices";
import { hazardousMaterialSchema } from "@/lib/validations/CommoditiesSchema";
import { HazardousMaterialFormValues as FormValues } from "@/types/commodities";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { Control, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Form, FormControl, FormGroup } from "../ui/form";

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
      </FormGroup>
      <div className="my-2 grid w-full items-center gap-0.5">
        <TextareaField
          name="description"
          control={control}
          label={t("fields.description.label")}
          placeholder={t("fields.description.placeholder")}
          description={t("fields.description.description")}
        />
      </div>
      <FormGroup className="grid gap-2 md:grid-cols-2 lg:grid-cols-2">
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
        <FormControl>
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
        <FormControl>
          <InputField
            control={control}
            name="additionalCost"
            label={t("fields.additionalCost.label")}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder={t("fields.additionalCost.placeholder")}
            description={t("fields.additionalCost.description")}
          />
        </FormControl>
      </FormGroup>
      <div className="my-2 grid w-full items-center gap-0.5">
        <TextareaField
          name="properShippingName"
          control={control}
          label={t("fields.properShippingName.label")}
          placeholder={t("fields.properShippingName.placeholder")}
          description={t("fields.properShippingName.description")}
        />
      </div>
    </Form>
  );
}

export function HazardousMaterialDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const { t } = useTranslation(["pages.hazardousmaterial", "common"]);

  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(hazardousMaterialSchema),
    defaultValues: {
      status: "A",
      name: "",
      description: undefined,
      hazardClass: undefined,
      packingGroup: undefined,
      ergNumber: undefined,
      additionalCost: undefined,
      properShippingName: undefined,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/hazardous_materials/",
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["hazardous-material-table-data"],
      closeModal: true,
      errorMessage: t("formErrorMessage"),
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
      <DialogContent className="max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>
        <DialogDescription>{t("description")}</DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <HazardousMaterialForm control={control} />
          <DialogFooter className="mt-6">
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving Changes..."
            >
              {t("buttons.save", { ns: "common" })}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
