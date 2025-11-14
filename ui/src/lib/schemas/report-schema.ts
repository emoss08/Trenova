import * as z from "zod";
import { FilterStateSchema } from "./table-configuration-schema";

export const ReportFormatSchema = z.enum(["Csv", "Excel"]);

export const DeliveryMethodSchema = z.enum(["Download", "Email"]);

export const ReportStatusSchema = z.enum([
  "Pending",
  "Processing",
  "Completed",
  "Failed",
]);

export const generateReportRequestSchema = z.object({
  resourceType: z.string().min(1, {
    message: "Resource type is required",
  }),
  name: z.string().min(1, {
    message: "Name is required",
  }),
  format: ReportFormatSchema,
  deliveryMethod: DeliveryMethodSchema,
  filterState: FilterStateSchema,
});

export const generateReportResponseSchema = z.object({
  reportId: z.string().min(1, {
    message: "Report ID is required",
  }),
});

export type GenerateReportResponseSchema = z.infer<
  typeof generateReportResponseSchema
>;
export type GenerateReportRequestSchema = z.infer<
  typeof generateReportRequestSchema
>;
export type ReportFormatSchema = z.infer<typeof ReportFormatSchema>;
export type DeliveryMethodSchema = z.infer<typeof DeliveryMethodSchema>;
export type ReportStatusSchema = z.infer<typeof ReportStatusSchema>;
