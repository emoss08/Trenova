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
import { Control, useForm } from "react-hook-form";
import { SelectInput } from "@/components/common/fields/select-input";
import { InputField } from "@/components/common/fields/input";
import { TextareaField } from "@/components/common/fields/textarea";
import React from "react";
import { TChoiceProps } from "@/types";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { toast } from "@/components/ui/use-toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useHazardousMaterial } from "@/hooks/useQueries";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { CommodityFormValues as FormValues } from "@/types/commodities";
import { statusChoices, UnitOfMeasureChoices } from "@/lib/choices";
import { yesAndNoChoices } from "@/lib/constants";

export function CommodityForm({
  control,
  hazardousMaterials,
  isLoading,
  isError,
}: {
  control: Control<FormValues>;
  hazardousMaterials: TChoiceProps[];
  isLoading: boolean;
  isError: boolean;
}) {
  return (
    <div className="flex-1 overflow-y-visible">
      <div className="grid md:grid-cols-2 lg:grid-cols-2 gap-2 mb-2">
        <div className="grid w-full max-w-sm items-center gap-0.5">
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
        </div>
        <div className="grid w-full items-center gap-0.5">
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
        </div>
      </div>
      <div className="grid w-full items-center gap-0.5 my-2">
        <TextareaField
          name="description"
          control={control}
          label="Description"
          placeholder="Description"
          description="Description of the Commodity"
        />
      </div>
      <div className="grid md:grid-cols-2 lg:grid-cols-2 gap-2">
        <div className="grid w-full max-w-sm items-center gap-0.5">
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
        </div>
        <div className="grid w-full items-center gap-0.5">
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
        </div>
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="hazardousMaterial"
            control={control}
            label="Hazardous Material"
            options={hazardousMaterials}
            maxOptions={10}
            isLoading={isLoading}
            isFetchError={isError}
            placeholder="Select Hazardous Material"
            description="The Hazardous Material associated with the Commodity"
            isClearable
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="isHazmat"
            control={control}
            label="Is Hazmat"
            options={yesAndNoChoices}
            placeholder="Is Hazmat"
            description="Is the Commodity a Hazardous Material?"
            isClearable
          />
        </div>
        <div className="grid w-full items-center gap-0.5">
          <SelectInput
            name="unitOfMeasure"
            control={control}
            label="Unit of Measure"
            options={UnitOfMeasureChoices}
            placeholder="Unit of Measure"
            description="Unit of Measure of the Commodity"
            isClearable
          />
        </div>
      </div>
    </div>
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

  React.useEffect(() => {
    const hazardousMaterial = watch("hazardousMaterial");

    if (hazardousMaterial) {
      setValue("isHazmat", "Y");
    } else {
      setValue("isHazmat", "N");
    }
  }, [watch("hazardousMaterial"), setValue]);

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
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

  const { selectHazardousMaterials, isLoading, isError } =
    useHazardousMaterial(open);

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
          <CommodityForm
            control={control}
            hazardousMaterials={selectHazardousMaterials}
            isLoading={isLoading}
            isError={isError}
          />
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
