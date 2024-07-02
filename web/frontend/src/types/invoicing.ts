import type { DateFormatChoiceProps } from "@/lib/choices";

export type InvoiceControl = {
  id: string;
  organizationId: string;
  invoiceNumberPrefix: string;
  creditMemoNumberPrefix: string;
  invoiceDueAfterDays: number;
  invoiceTerms?: string;
  invoiceFooter?: string;
  invoiceLogoUrl?: any;
  invoiceLogoWidth: number;
  showInvoiceDueDate: boolean;
  invoiceDateFormat: DateFormatChoiceProps;
  showAmountDue: boolean;
  attachPdf: boolean;
};
export type InvoiceControlFormValues = Omit<
  InvoiceControl,
  "id" | "organizationId"
>;
