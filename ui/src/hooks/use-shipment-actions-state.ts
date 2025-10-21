/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { parseAsBoolean, type inferParserType } from "nuqs";

export const shipmentActionsParser = {
  auditDialogOpen: parseAsBoolean.withDefault(false),
  documentDialogOpen: parseAsBoolean.withDefault(false),
  addDocumentDialogOpen: parseAsBoolean.withDefault(false),
  unCancelDialogOpen: parseAsBoolean.withDefault(false),
  cancellationDialogOpen: parseAsBoolean.withDefault(false),
  duplicateDialogOpen: parseAsBoolean.withDefault(false),
  transferDialogOpen: parseAsBoolean.withDefault(false),
  holdDialogOpen: parseAsBoolean.withDefault(false),
};

export type ShipmentActionState = inferParserType<typeof shipmentActionsParser>;
