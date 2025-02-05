import { CommoditySchema } from "@/lib/schemas/commodity-schema";
import { CustomerSchema } from "@/lib/schemas/customer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { type ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
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

export const mapToRatingMethod = (ratingMethod: RatingMethod) => {
  const ratingMethodLabels = {
    FlatRate: "Flat Rate",
    PerMile: "Per Mile",
    PerStop: "Per Stop",
    PerPound: "Per Pound",
    PerPallet: "Per Pallet",
    PerLinearFoot: "Per Linear Foot",
    Other: "Other",
  };

  return ratingMethodLabels[ratingMethod];
};

export enum EntryMethod {
  Manual = "Manual",
  Electronic = "Electronic",
}

export type ShipmentCommodity = ShipmentCommoditySchema & {
  commodity: CommoditySchema;
  shipment: ShipmentSchema;
};

export type Shipment = ShipmentSchema & {
  serviceType: ServiceTypeSchema;
  shipmentType: ShipmentTypeSchema;
  customer: CustomerSchema;
  commodities: ShipmentCommodity[];
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

export type ShipmentCardProps = {
  shipment: Shipment;
  isSelected: boolean;
  onSelect: (shipmentId: string) => void;
  inputValue?: string;
};

export type ShipmentListProps = {
  displayData: (Shipment | undefined)[];
  isLoading: boolean;
  selectedShipmentId?: string | null;
  onShipmentSelect: (shipmentId: string) => void;
  inputValue?: string;
};
