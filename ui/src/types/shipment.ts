/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { QueryOptions } from "./common";

export type ShipmentQueryParams = {
  expandShipmentDetails?: boolean;
  enabled?: boolean;
} & QueryOptions;

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
