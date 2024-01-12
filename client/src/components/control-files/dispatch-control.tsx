/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
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

import { Button } from "@/components/ui/button";
import { useDispatchControl } from "@/hooks/useQueries";
import React from "react";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { CheckboxInput } from "@/components/common/fields/checkbox";
import { yupResolver } from "@hookform/resolvers/yup";
import { Skeleton } from "@/components/ui/skeleton";
import { SelectInput } from "@/components/common/fields/select-input";
import { serviceIncidentControlChoices } from "@/lib/choices";
import {
  DispatchControl as DispatchControlType,
  DispatchControlFormValues,
} from "@/types/dispatch";
import { dispatchControlSchema } from "@/lib/validations/DispatchSchema";
import { InputField } from "@/components/common/fields/input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";

function DispatchControlForm({
  dispatchControl,
}: {
  dispatchControl: DispatchControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit, reset } = useForm<DispatchControlFormValues>({
    resolver: yupResolver(dispatchControlSchema),
    defaultValues: dispatchControl,
  });

  const mutation = useCustomMutation<DispatchControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/dispatch_control/${dispatchControl.id}/`,
      successMessage: "Dispatch Control updated successfully.",
      queryKeysToInvalidate: ["dispatchControl"],
      errorMessage: "Failed to update dispatch control.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: DispatchControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="recordServiceIncident"
              control={control}
              options={serviceIncidentControlChoices}
              rules={{ required: true }}
              label="Record Service Incident"
              placeholder="Record Service Incident"
              description="Option to log service incidents automatically for delayed shipments."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="gracePeriod"
              control={control}
              type="number"
              rules={{ required: true }}
              label="Grace Period"
              placeholder="Grace Period"
              description="Enter the number of minutes to wait before recording a service incident."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="deadheadTarget"
              type="number"
              control={control}
              rules={{ required: true }}
              label="Deadhead Target"
              placeholder="Deadhead Target"
              description="Specify the maximum miles a driver can travel unloaded to optimize route efficiency."
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="maxShipmentWeightLimit"
              type="number"
              control={control}
              rules={{ required: true }}
              label="Max Shipment Weight Limit"
              placeholder="Max Shipment Weight Limit"
              description="Sets the maximum allowable weight (in pounds) for any shipment. Dispatch is prevented if the weight exceeds this limit."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceWorkerAssign"
              control={control}
              label="Enforce Worker Assignment"
              description="Mandate specific worker assignments for each shipment for better accountability."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="trailerContinuity"
              control={control}
              label="Enforce Trailer Continuity"
              description="Ensure the same trailer is used throughout a shipment for consistency."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="dupeTrailerCheck"
              control={control}
              label="Enforce Duplicate Trailer Check"
              description="Activate checks against using the same trailer for multiple simultaneous shipments."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="regulatoryCheck"
              control={control}
              label="Enforce Regulatory Check"
              description="Implement a mandatory check to ensure all shipments comply with regulations."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="prevShipmentsOnHold"
              control={control}
              label="Prevent Shipments On Hold"
              description="Prevent allocation of shipments to drivers who have ongoing shipments on hold."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="workerTimeAwayRestriction"
              control={control}
              label="Enforce Worker Time Away"
              description="Disallow assignments to workers currently on approved time away."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="tractorWorkerFleetConstraint"
              control={control}
              label="Enforce Tractor and Worker Fleet Constraint"
              description="Restrict dispatch assignments to workers and tractors from the same fleet."
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="maintenanceCompliance"
              control={control}
              label="Vehicle Maintenance Compliance"
              description="Ensures that all vehicles are compliant with maintenance standards before dispatch."
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-6 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
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

export default function DispatchControl() {
  const { dispatchControlData, isLoading, isError } = useDispatchControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Dispatch Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Elevate your dispatch operations with our comprehensive Dispatch
          Control Panel. This module is designed to streamline dispatch
          processes, enforce compliance, and optimize operational workflows,
          ensuring a smooth and efficient management of your transportation
          services.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="m-4 bg-background p-8 ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <ErrorLoadingData message="Failed to load dispatch control." />
        </div>
      ) : (
        dispatchControlData && (
          <DispatchControlForm dispatchControl={dispatchControlData} />
        )
      )}
    </div>
  );
}
