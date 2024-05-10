import { CheckboxInput } from "@/components/common/fields/checkbox";
import { SelectInput } from "@/components/common/fields/select-input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
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
    errorMessage: "Failed to update route control.",
  });

  const onSubmit = (values: RouteControlFormValues) => {
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
      <div className="border-muted flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
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
        <h2 className="text-foreground text-base font-semibold leading-7">
          Route Control
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          Streamline your route planning and management with our Routing
          Optimization Panel. This module is engineered to enhance efficiency
          and precision in route selection, ensuring optimal pathing and
          distance calculations for your transportation needs.
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="bg-background ring-muted m-4 p-8 ring-1 sm:rounded-xl md:col-span-2">
          <ErrorLoadingData />
        </div>
      ) : (
        data && <RouteControlForm routeControl={data} />
      )}
    </div>
  );
}
