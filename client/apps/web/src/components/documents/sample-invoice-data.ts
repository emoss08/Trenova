/** The shipment and charges every sample invoice in the app prints. */
export const SAMPLE_INVOICE = {
  invoiceNumber: "INV-2026-1042",
  billTo: {
    name: "Halstead Grocery Group",
    lines: ["Accounts Payable", "900 Distribution Pkwy", "Fort Worth, TX 76106"],
  },
  shipper: { name: "Rio Grande Produce", lines: ["1200 Airport Rd", "Pharr, TX 78577"] },
  consignee: { name: "Halstead DC 14", lines: ["900 Distribution Pkwy", "Fort Worth, TX 76106"] },
  refs: [
    ["Pro Number", "TRV-884120"],
    ["BOL", "BOL-77401"],
    ["Customer PO", "PO-5591004"],
  ],
  charges: [
    { description: "Linehaul", quantity: "612 mi", rate: "2.85", amountMinor: 174_420 },
    { description: "Fuel surcharge", quantity: "612 mi", rate: "0.42", amountMinor: 25_704 },
    {
      description: "Detention — Halstead DC 14",
      quantity: "2.5 hr",
      rate: "65.00",
      amountMinor: 16_250,
    },
    { description: "Lumper", quantity: "1", rate: "175.00", amountMinor: 17_500 },
  ],
} as const;

export const SAMPLE_INVOICE_TOTAL_MINOR = SAMPLE_INVOICE.charges.reduce(
  (sum, charge) => sum + charge.amountMinor,
  0,
);

/** The classes that size a logo in the invoice's masthead: fitted whole, never cut. */
export const INVOICE_LOGO_CLASS = "block max-h-11 max-w-30 object-contain";
