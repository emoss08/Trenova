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
