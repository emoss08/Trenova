import type {
  DistanceMethodChoiceProps,
  RouteDistanceUnitProps,
} from "@/lib/choices";

export type RouteControl = {
  id: string;
  organization: string;
  distanceMethod: DistanceMethodChoiceProps;
  mileageUnit: RouteDistanceUnitProps;
  generateRoutes: boolean;
};

export type RouteControlFormValues = Omit<RouteControl, "id" | "organization">;
