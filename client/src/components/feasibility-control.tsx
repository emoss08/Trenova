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
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useFeasibilityControl } from "@/hooks/useQueries";
import { feasibilityOperatorChoices } from "@/lib/choices";
import { feasibilityControlSchema } from "@/lib/validations/DispatchSchema";
import type {
  FeasibilityToolControlFormValues,
  FeasibilityToolControl as FeasibilityToolControlType,
} from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { ErrorLoadingData } from "./common/table/data-table-components";

function FeasibilityControlForm({
  feasibilityControl,
}: {
  feasibilityControl: FeasibilityToolControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit, reset } =
    useForm<FeasibilityToolControlFormValues>({
      resolver: yupResolver(feasibilityControlSchema),
      defaultValues: feasibilityControl,
    });

  const mutation = useCustomMutation<FeasibilityToolControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/feasibility-tool-control/${feasibilityControl.id}/`,
      successMessage: "Feasibility Control updated successfully.",
      queryKeysToInvalidate: ["feasibilityControl"],
      errorMessage: "Failed to update feasibility control.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FeasibilityToolControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="mpwOperator"
              control={control}
              label="Miles Per Week Operator"
              options={feasibilityOperatorChoices}
              rules={{ required: true }}
              placeholder="Miles Per Week Operator"
              description="Select the operator (like greater than or less than) for evaluating miles per week."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="mpwValue"
              control={control}
              type="number"
              label="Miles Per Week Value"
              rules={{ required: true }}
              placeholder="Miles Per Week Value"
              description="Specify the mileage threshold for the miles per week value."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="mpdOperator"
              control={control}
              label="Miles Per Day Operator"
              options={feasibilityOperatorChoices}
              rules={{ required: true }}
              placeholder="Miles Per Day Operator"
              description="Choose the operator for daily mileage evaluation in worker assignment."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="mpdValue"
              control={control}
              type="number"
              label="Miles Per Day Value"
              rules={{ required: true }}
              placeholder="Miles Per Day Value"
              description="Set the mileage limit for the miles per day value."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="mpgOperator"
              control={control}
              label="Miles Per Gallon Operator"
              options={feasibilityOperatorChoices}
              rules={{ required: true }}
              placeholder="Miles Per Gallon Operator"
              description="Determine the operator for assessing miles per gallon in feasibility checks."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="mpgValue"
              control={control}
              type="number"
              label="Miles Per Gallon Value"
              rules={{ required: true }}
              placeholder="Miles Per Gallon Value"
              description="Define the value for miles per gallon evaluations."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="otpOperator"
              control={control}
              label="On-Time Percentage Operator"
              options={feasibilityOperatorChoices}
              rules={{ required: true }}
              placeholder="Miles Per Gallon Operator"
              description="Select the operator for on-time percentage assessment."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="otpValue"
              control={control}
              type="number"
              label="On-Time Percentage Value"
              rules={{ required: true }}
              placeholder="On-Time Percentage Value"
              description="Establish the threshold for on-time percentage in worker assignments."
            />
          </div>
        </div>
      </div>
      <div className="border-muted flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={isSubmitting}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function FeasibilityControl() {
  const { data, isLoading, isError } = useFeasibilityControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-foreground text-base font-semibold leading-7">
          Feasibility Control
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Optimize your workforce allocation with our Worker Feasibility Tool
          Panel. This tool dynamically assesses the suitability of workers for
          shipments based on their performance metrics and Hours of Service
          (HOS) data, ensuring efficient and compliant assignment decisions.
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="bg-background ring-muted m-4 p-8 ring-1 sm:rounded-xl md:col-span-2">
          <ErrorLoadingData message="Failed to load feasibility control." />
        </div>
      ) : (
        data && <FeasibilityControlForm feasibilityControl={data} />
      )}
    </div>
  );
}
