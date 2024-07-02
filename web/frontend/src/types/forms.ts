/** Types for Export Model */
export type ExportModelChoices = "csv" | "xlsx" | "pdf";

export type DeliveryMethodChoices = "email" | "local";

export type TExportModelFormValues = {
  fileFormat: ExportModelChoices;
  deliveryMethod: DeliveryMethodChoices;
  emailRecipients?: string;
  columns: string[];
};
