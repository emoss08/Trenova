import { z } from "zod";
import {
  billingCycleSchema,
  invoiceDetailSchema,
  invoiceSectionKeySchema,
  invoiceSplitKeySchema,
} from "./customer";
import { decimalStringSchema, nullableStringSchema } from "./helpers";

export const statementShipmentSchema = z.object({
  billingQueueItemId: z.string(),
  shipmentId: z.string(),
  orderId: nullableStringSchema,
  proNumber: nullableStringSchema,
  bol: nullableStringSchema,
  poNumber: nullableStringSchema,
  orderNumber: nullableStringSchema,
  serviceDate: z.number().int().nullish(),
  amount: decimalStringSchema,
});
export type StatementShipment = z.infer<typeof statementShipmentSchema>;

export const statementGroupSchema = z.object({
  key: z.string(),
  label: z.string(),
  shipmentCount: z.number().int().default(0),
  totalAmount: decimalStringSchema,
  belowMinimum: z.boolean().default(false),
  /** Absent on the list read, which does not need the members. */
  shipments: z.array(statementShipmentSchema).nullish(),
});
export type StatementGroup = z.infer<typeof statementGroupSchema>;

export const openStatementSchema = z.object({
  customerId: z.string(),
  customerName: z.string(),
  customerCode: nullableStringSchema,
  customerStatus: nullableStringSchema,

  cycle: billingCycleSchema,
  billingCycleAnchorDay: z.number().int().default(1),
  billingCycleTimezone: z.string().default("UTC"),
  periodStart: z.number().int(),
  periodEnd: z.number().int(),
  lastBilledPeriodEnd: z.number().int().nullish(),

  shipmentCount: z.number().int().default(0),
  invoiceCount: z.number().int().default(0),
  totalAmount: decimalStringSchema,
  currencyCode: z.string().default("USD"),

  splitBy: invoiceSplitKeySchema,
  sectionBy: invoiceSectionKeySchema,
  detail: invoiceDetailSchema,
  minimumAmount: decimalStringSchema.nullish(),
  autoBill: z.boolean().default(false),
  belowMinimum: z.boolean().default(false),

  groups: z.array(statementGroupSchema).nullish(),
});
export type OpenStatement = z.infer<typeof openStatementSchema>;

export const openStatementListSchema = z.object({
  results: z.array(openStatementSchema).default([]),
  count: z.number().int().default(0),
});
