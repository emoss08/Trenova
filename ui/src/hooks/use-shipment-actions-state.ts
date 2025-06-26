import { parseAsBoolean, type inferParserType } from "nuqs";

export const shipmentActionsParser = {
  auditDialogOpen: parseAsBoolean.withDefault(false),
  documentDialogOpen: parseAsBoolean.withDefault(false),
  addDocumentDialogOpen: parseAsBoolean.withDefault(false),
  unCancelDialogOpen: parseAsBoolean.withDefault(false),
  cancellationDialogOpen: parseAsBoolean.withDefault(false),
  duplicateDialogOpen: parseAsBoolean.withDefault(false),
  transferDialogOpen: parseAsBoolean.withDefault(false),
};

export type ShipmentActionState = inferParserType<typeof shipmentActionsParser>;
