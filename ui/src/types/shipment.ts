import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { type EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { type ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { type ShipmentCommoditySchema } from "@/lib/schemas/shipment-commodity-schema";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { BaseModel } from "./common";
import { type ShipmentMove } from "./move";
import { type User } from "./user";

export enum ShipmentStatus {
  New = "New",
  PartiallyAssigned = "PartiallyAssigned",
  Assigned = "Assigned",
  InTransit = "InTransit",
  Delayed = "Delayed",
  PartiallyCompleted = "PartiallyCompleted",
  Completed = "Completed",
  Billed = "Billed",
  Canceled = "Canceled",
}

export const mapToShipmentStatus = (status: ShipmentStatus) => {
  const statusLabels = {
    New: "New",
    PartiallyAssigned: "Partially Assigned",
    PartiallyCompleted: "Partially Completed",
    Assigned: "Assigned",
    InTransit: "In Transit",
    Delayed: "Delayed",
    Completed: "Completed",
    Billed: "Billed",
    Canceled: "Canceled",
  };

  return statusLabels[status];
};

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

export type Shipment = BaseModel &
  ShipmentSchema & {
    serviceType: ServiceTypeSchema;
    shipmentType: ShipmentTypeSchema;
    customer: CustomerSchema;
    commodities: ShipmentCommodity[];
    tractorType?: EquipmentTypeSchema | null;
    trailerType?: EquipmentTypeSchema | null;
    moves: ShipmentMove[];
    // Cancelation Related Fields
    canceledAt?: number | null;
    canceledById?: string | null;
    cancelReason?: string | null;
    canceledBy?: User | null;
  };

export type ShipmentQueryParams = {
  pageIndex: number;
  pageSize: number;
  expandShipmentDetails: boolean;
  query?: string;
  enabled?: boolean;
};

export type ShipmentDetailsQueryParams = {
  shipmentId: string;
  enabled?: boolean;
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
  shipment?: Shipment;
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

export enum ShipmentDocumentType {
  BillOfLading = "BillOfLading",
  ProofOfDelivery = "ProofOfDelivery",
  Invoice = "Invoice",
  DeliveryReceipt = "DeliveryReceipt",
  Other = "Other",
}
