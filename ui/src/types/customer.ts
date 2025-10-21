/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export type CustomerDocumentRequirement = {
  name: string;
  docId: string;
  description: string;
  color: string;
};

export type GetCustomerByIDParams = {
  customerId: string;
  includeBillingProfile?: boolean;
  enabled?: boolean;
};
