import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { QueryOptions } from "./common";

export type ShipmentQueryParams = {
  expandShipmentDetails?: boolean;
  enabled?: boolean;
} & QueryOptions;

export type ShipmentDetailsQueryParams = {
  shipmentId: ShipmentSchema["id"];
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
  shipment?: ShipmentSchema;
  onSelect: (shipmentId: string) => void;
  inputValue?: string;
};

export type ShipmentListProps = {
  displayData: ShipmentSchema[];
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
