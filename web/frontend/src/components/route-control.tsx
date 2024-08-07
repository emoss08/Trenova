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

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useRouteControl } from "@/hooks/useQueries";
import { distanceMethodChoices, routeDistanceUnitChoices } from "@/lib/choices";
import { routeControlSchema } from "@/lib/validations/RouteSchema";
import type {
  RouteControlFormValues,
  RouteControl as RouteControlType,
} from "@/types/route";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ErrorLoadingData } from "./common/table/data-table-components";
import { ComponentLoader } from "./ui/component-loader";

function RouteControlForm({
  routeControl,
}: {
  routeControl: RouteControlType;
}) {
  const { control, handleSubmit, reset } = useForm<RouteControlFormValues>({
    resolver: yupResolver(routeControlSchema),
    defaultValues: routeControl,
  });

  const mutation = useCustomMutation<RouteControlFormValues>(control, {
    method: "PUT",
    path: `/route-control/${routeControl.id}/`,
    successMessage: "Route Control updated successfully.",
    queryKeysToInvalidate: "routeControl",
    reset,
    errorMessage: "Failed to update route control.",
  });

  const onSubmit = (values: RouteControlFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="m-4 border border-border bg-card sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="distanceMethod"
              control={control}
              label="Distance Method"
              options={distanceMethodChoices}
              rules={{ required: true }}
              placeholder="Distance Method"
              description="Choose the preferred method for calculating distances for route planning."
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="mileageUnit"
              control={control}
              label="Mileage Unit"
              options={routeDistanceUnitChoices}
              rules={{ required: true }}
              placeholder="Mileage Unit"
              description="Select the unit of measurement for mileage, such as miles or kilometers."
            />
          </div>
          <div className="col-span-full">
            <CheckboxInput
              name="generateRoutes"
              control={control}
              label="Generate Routes"
              description="Enable automatic generation of shipment routes based on optimal pathing algorithms."
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-x-4 border-t border-muted p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="outline"
          disabled={mutation.isPending}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
          Save
        </Button>
      </div>
    </form>
  );
}

export default function RouteControl() {
  const { data, isLoading, isError } = useRouteControl();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-base font-semibold leading-7 text-foreground">
          Route Control
        </h2>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          Streamline your route planning and management with our Routing
          Optimization Panel. This module is engineered to enhance efficiency
          and precision in route selection, ensuring optimal pathing and
          distance calculations for your transportation needs.
        </p>
      </div>
      {isLoading ? (
        <div className="m-4 bg-background ring-1 ring-muted sm:rounded-xl md:col-span-2">
          <ComponentLoader className="h-[30em]" />
        </div>
      ) : isError ? (
        <ErrorLoadingData />
      ) : (
        data && <RouteControlForm routeControl={data} />
      )}
    </div>
  );
}
