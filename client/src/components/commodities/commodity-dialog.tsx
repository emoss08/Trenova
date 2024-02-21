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
import { useHazardousMaterial } from "@/hooks/useQueries";
import { UnitOfMeasureChoices, statusChoices } from "@/lib/choices";
import { yesAndNoChoices } from "@/lib/constants";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { CommodityFormValues as FormValues } from "@/types/commodities";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React, { useEffect } from "react";
import { Control, useForm } from "react-hook-form";
import { Form, FormControl, FormGroup } from "../ui/form";

export function CommodityForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const { selectHazardousMaterials, isLoading, isError } =
    useHazardousMaterial(open);

  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Commodity"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Name for the Commodity"
          />
        </FormControl>
      </FormGroup>
      <div className="my-2 grid w-full items-center gap-0.5">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Commodity"
        />
      </div>
      <FormGroup className="grid gap-2 md:grid-cols-2 lg:grid-cols-2">
        <FormControl>
          <InputField
            name="minTemp"
            control={control}
            label="Min. Temp"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Min. Temp"
            description="Minimum Temperature of the Commodity"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="maxTemp"
            label="Max. Temp"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Max. Temp"
            description="Maximum Temperature of the Commodity"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="hazardousMaterial"
            control={control}
            label="Hazardous Material"
            options={selectHazardousMaterials}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Hazardous Material"
            description="The Hazardous Material associated with the Commodity"
            isClearable
            hasPopoutWindow
            popoutLink="/shipment-management/hazardous-materials/"
            popoutLinkLabel="Hazardous Material"
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="isHazmat"
            control={control}
            label="Is Hazmat"
            options={yesAndNoChoices}
            placeholder="Is Hazmat"
            description="Is the Commodity a Hazardous Material?"
            isClearable
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="unitOfMeasure"
            control={control}
            label="Unit of Measure"
            options={UnitOfMeasureChoices}
            placeholder="Unit of Measure"
            description="Unit of Measure of the Commodity"
            isClearable
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function CommodityDialog({ onOpenChange, open }: TableSheetProps) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit, watch, setValue } = useForm<FormValues>(
    {
      resolver: yupResolver(commoditySchema),
      defaultValues: {
        status: "A",
        name: "",
        description: undefined,
        minTemp: undefined,
        maxTemp: undefined,
        setPointTemp: undefined,
        unitOfMeasure: undefined,
        hazardousMaterial: undefined,
        isHazmat: "N",
      },
    },
  );

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "hazardousMaterial" && value.hazardousMaterial) {
        setValue("isHazmat", "Y");
      } else if (name === "hazardousMaterial" && !value.hazardousMaterial) {
        setValue("isHazmat", "N");
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "POST",
      path: "/commodities/",
      successMessage: "Commodity created successfully.",
      queryKeysToInvalidate: ["commodity-table-data"],
      closeModal: true,
      errorMessage: "Failed to create new commodity.",
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
          <DialogTitle>Create New Commodity</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Please fill out the form below to create a new Commodity.
        </DialogDescription>
        <form onSubmit={handleSubmit(onSubmit)}>
          <CommodityForm control={control} open={open} />
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
