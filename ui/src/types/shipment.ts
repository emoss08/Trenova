import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { type ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { ShipmentMove } from "./move";

export enum ShipmentStatus {
  New = "New",
  InTransit = "InTransit",
  Delayed = "Delayed",
  Completed = "Completed",
  Billed = "Billed",
  Canceled = "Canceled",
}

export enum RatingMethod {
  FlatRate = "FlatRate",
  PerMile = "PerMile",
  PerStop = "PerStop",
  PerPound = "PerPound",
  PerPallet = "PerPallet",
  PerLinearFoot = "PerLinearFoot",
  Other = "Other",
}

export enum EntryMethod {
  Manual = "Manual",
  Electronic = "Electronic",
}

export type Shipment = ShipmentSchema & {
  serviceType: ServiceTypeSchema;
  shipmentType: ShipmentTypeSchema;
  customer: CustomerSchema;
  tractorType?: EquipmentTypeSchema | null;
  trailerType?: EquipmentTypeSchema | null;
  moves: ShipmentMove[];
};

export type ShipmentQueryParams = {
  pageIndex: number;
  pageSize: number;
  expandShipmentDetails: boolean;
  query?: string;
};

export type ShipmentDetailsQueryParams = {
  shipmentId: string;
};

export type ShipmentPaginationProps = {
  totalCount: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  pageSizeOptions: readonly number[];
  isLoading: boolean;
};
