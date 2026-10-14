import { defineLabels } from "@trenova/shared/i18n/labels";
import type { EDIMappingEntityType } from "@trenova/shared/types/edi";

/** What each kind of record an EDI mapping ties a partner's code to is called. */
export const MAPPING_ENTITY_LABELS: Record<EDIMappingEntityType, string> = defineLabels({
  Customer: "Customer",
  ServiceType: "Service type",
  ShipmentType: "Shipment type",
  FormulaTemplate: "Formula template",
  Location: "Location",
  Commodity: "Commodity",
  AccessorialCharge: "Accessorial charge",
  ServiceFailureReasonCode: "Service failure reason code",
});
