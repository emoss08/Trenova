import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";

export type ShipmentQueryParams = {
  pageIndex?: number;
  pageSize?: number;
  expandShipmentDetails?: boolean;
  query?: string;
  enabled?: boolean;
  status?: ShipmentSchema["status"];
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
