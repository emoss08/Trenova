import { z } from "zod";
import { optionalStringSchema } from "./helpers";

export const payeeClassificationSchema = z.enum(["CompanyDriver", "OwnerOperator"]);
export type PayeeClassification = z.infer<typeof payeeClassificationSchema>;

export const payComponentKindSchema = z.enum([
  "Linehaul",
  "FuelSurcharge",
  "StopPay",
  "Detention",
  "Layover",
  "Breakdown",
  "Tarp",
  "Hazmat",
  "Bonus",
  "Custom",
]);
export type PayComponentKind = z.infer<typeof payComponentKindSchema>;

export const payCalcMethodSchema = z.enum([
  "PerLoadedMile",
  "PerEmptyMile",
  "PerTotalMile",
  "PercentOfRevenue",
  "FlatPerShipment",
  "PerStop",
  "PerHour",
  "PerDay",
  "PerEvent",
]);
export type PayCalcMethod = z.infer<typeof payCalcMethodSchema>;

export const payRevenueBasisSchema = z.enum([
  "Linehaul",
  "LinehaulPlusFuelSurcharge",
  "TotalRevenue",
]);
export type PayRevenueBasis = z.infer<typeof payRevenueBasisSchema>;

export const payCodeDirectionSchema = z.enum(["Earning", "Deduction"]);
export type PayCodeDirection = z.infer<typeof payCodeDirectionSchema>;

export const recurringDeductionFrequencySchema = z.enum(["EverySettlement", "Monthly"]);
export type RecurringDeductionFrequency = z.infer<typeof recurringDeductionFrequencySchema>;

export const recurringDeductionStatusSchema = z.enum(["Active", "Paused", "Completed"]);
export type RecurringDeductionStatus = z.infer<typeof recurringDeductionStatusSchema>;

export const recurringEarningFrequencySchema = z.enum(["EverySettlement", "Monthly"]);
export type RecurringEarningFrequency = z.infer<typeof recurringEarningFrequencySchema>;

export const recurringEarningStatusSchema = z.enum(["Active", "Paused", "Completed"]);
export type RecurringEarningStatus = z.infer<typeof recurringEarningStatusSchema>;

export const payAdvanceStatusSchema = z.enum([
  "Outstanding",
  "PartiallyRecovered",
  "Recovered",
  "WrittenOff",
]);
export type PayAdvanceStatus = z.infer<typeof payAdvanceStatusSchema>;

export const payAdvanceSourceSchema = z.enum([
  "Cash",
  "EFSMoneyCode",
  "ComdataCode",
  "FuelCard",
  "Other",
]);
export type PayAdvanceSource = z.infer<typeof payAdvanceSourceSchema>;

export const escrowAccountStatusSchema = z.enum(["Active", "Closed"]);
export type EscrowAccountStatus = z.infer<typeof escrowAccountStatusSchema>;

export const escrowTransactionTypeSchema = z.enum([
  "Contribution",
  "InterestAccrual",
  "Application",
  "Refund",
  "Adjustment",
]);
export type EscrowTransactionType = z.infer<typeof escrowTransactionTypeSchema>;

export const driverSettlementStatusSchema = z.enum([
  "Draft",
  "PendingApproval",
  "Approved",
  "Posted",
  "Paid",
  "Voided",
]);
export type DriverSettlementStatus = z.infer<typeof driverSettlementStatusSchema>;

export const settlementLineCategorySchema = z.enum([
  "Earning",
  "Reimbursement",
  "Deduction",
  "AdvanceRecovery",
  "EscrowContribution",
  "GuaranteeTopUp",
  "CarryForward",
  "Adjustment",
]);
export type SettlementLineCategory = z.infer<typeof settlementLineCategorySchema>;

export const settlementBatchStatusSchema = z.enum(["Open", "Completed", "Canceled"]);
export type SettlementBatchStatus = z.infer<typeof settlementBatchStatusSchema>;

export const driverPayEventStatusSchema = z.enum(["Accrued", "Settled", "Voided"]);
export type DriverPayEventStatus = z.infer<typeof driverPayEventStatusSchema>;

export const payPeriodFrequencySchema = z.enum(["Weekly", "Biweekly", "Monthly"]);
export type PayPeriodFrequency = z.infer<typeof payPeriodFrequencySchema>;

export const settlementPayTriggerSchema = z.enum([
  "MoveCompleted",
  "ShipmentDelivered",
  "PODReceived",
  "ShipmentInvoiced",
]);
export type SettlementPayTrigger = z.infer<typeof settlementPayTriggerSchema>;

const decimalStringSchema = z
  .string()
  .refine((value) => value === "" || !Number.isNaN(Number(value)), {
    message: "Must be a valid number",
  });

export const mileageBandFormSchema = z.object({
  minMiles: z.number().int().min(0, "Minimum miles cannot be negative"),
  maxMiles: z.number().int().min(0, "Maximum miles cannot be negative"),
  rate: decimalStringSchema.refine((value) => value !== "", "Rate is required"),
});
export type MileageBandFormValues = z.infer<typeof mileageBandFormSchema>;

export const payProfileComponentFormSchema = z
  .object({
    kind: payComponentKindSchema,
    method: payCalcMethodSchema,
    description: optionalStringSchema,
    rate: decimalStringSchema.refine((value) => value !== "", "Rate is required"),
    revenueBasis: payRevenueBasisSchema.optional().nullable(),
    bands: z.array(mileageBandFormSchema).optional(),
    freeTimeMinutes: z.number().int().min(0).optional(),
    minAmount: z.number().min(0).optional().nullable(),
    maxAmount: z.number().min(0).optional().nullable(),
    isActive: z.boolean(),
  })
  .superRefine((component, ctx) => {
    if (component.method === "PercentOfRevenue" && !component.revenueBasis) {
      ctx.addIssue({
        code: "custom",
        path: ["revenueBasis"],
        message: "Revenue basis is required for percentage components",
      });
    }
    if (component.kind === "Custom" && !component.description) {
      ctx.addIssue({
        code: "custom",
        path: ["description"],
        message: "Description is required for custom components",
      });
    }
    if (component.kind === "Detention" && component.method !== "PerHour") {
      ctx.addIssue({
        code: "custom",
        path: ["method"],
        message: "Detention components must use the Per Hour method",
      });
    }
  });
export type PayProfileComponentFormValues = z.infer<typeof payProfileComponentFormSchema>;

export const payProfileFormSchema = z.object({
  status: z.enum(["Active", "Inactive"]),
  name: z.string().min(1, "Name is required").max(100),
  description: optionalStringSchema,
  classification: payeeClassificationSchema,
  guaranteedPeriodMinimum: z.number().min(0).optional().nullable(),
  perDiemRatePerMile: decimalStringSchema.optional(),
  perDiemDailyCap: z.number().min(0).optional().nullable(),
  components: z.array(payProfileComponentFormSchema).min(1, "Add at least one pay component"),
});
export type PayProfileFormValues = z.infer<typeof payProfileFormSchema>;

export const assignPayProfileFormSchema = z.object({
  workerId: z.string().min(1, "Worker is required"),
  payProfileId: z.string().min(1, "Pay profile is required"),
  effectiveFrom: z.number().int().min(1, "Effective from date is required"),
  effectiveTo: z.number().int().optional().nullable(),
  splitPercent: z
    .number()
    .gt(0, "Split percent must be greater than zero")
    .max(100, "Split percent cannot exceed 100"),
  notes: optionalStringSchema,
});
export type AssignPayProfileFormValues = z.infer<typeof assignPayProfileFormSchema>;

export const recurringDeductionFormSchema = z
  .object({
    workerId: z.string().min(1, "Worker is required"),
    payCodeId: z.string().min(1, "Pay code is required"),
    escrowContribution: z.boolean(),
    status: recurringDeductionStatusSchema,
    frequency: recurringDeductionFrequencySchema,
    description: z.string().min(1, "Description is required").max(255),
    amount: z.number().gt(0, "Amount must be greater than zero"),
    totalCap: z.number().gt(0).optional().nullable(),
    startDate: z.number().int().min(1, "Start date is required"),
    endDate: z.number().int().optional().nullable(),
  })
  .superRefine((deduction, ctx) => {
    if (
      deduction.endDate != null &&
      deduction.endDate !== 0 &&
      deduction.endDate <= deduction.startDate
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["endDate"],
        message: "End date must be after the start date",
      });
    }
  });
export type RecurringDeductionFormValues = z.infer<typeof recurringDeductionFormSchema>;

export const recurringEarningFormSchema = z
  .object({
    workerId: z.string().min(1, "Worker is required"),
    payCodeId: z.string().min(1, "Pay code is required"),
    status: recurringEarningStatusSchema,
    frequency: recurringEarningFrequencySchema,
    description: z.string().min(1, "Description is required").max(255),
    amount: z.number().gt(0, "Amount must be greater than zero"),
    totalCap: z.number().gt(0).optional().nullable(),
    startDate: z.number().int().min(1, "Start date is required"),
    endDate: z.number().int().optional().nullable(),
  })
  .superRefine((earning, ctx) => {
    if (earning.endDate != null && earning.endDate !== 0 && earning.endDate <= earning.startDate) {
      ctx.addIssue({
        code: "custom",
        path: ["endDate"],
        message: "End date must be after the start date",
      });
    }
  });
export type RecurringEarningFormValues = z.infer<typeof recurringEarningFormSchema>;

export const payCodeFormSchema = z.object({
  direction: payCodeDirectionSchema,
  code: z
    .string()
    .min(1, "Code is required")
    .max(20)
    .regex(/^[A-Za-z0-9][A-Za-z0-9_-]*$/, "Use letters, digits, dashes, or underscores only"),
  name: z.string().min(1, "Name is required").max(100),
  description: z.string().max(500).optional().nullable(),
  status: z.enum(["Active", "Inactive"]),
  taxable: z.boolean(),
  countsTowardGuarantee: z.boolean(),
  glAccountId: z.string().optional().nullable(),
  defaultAmount: z.number().gt(0).optional().nullable(),
});
export type PayCodeFormValues = z.infer<typeof payCodeFormSchema>;

export const issuePayAdvanceFormSchema = z.object({
  workerId: z.string().min(1, "Worker is required"),
  source: payAdvanceSourceSchema,
  reference: optionalStringSchema,
  issuedDate: z.number().int().min(1, "Issued date is required"),
  amount: z.number().gt(0, "Amount must be greater than zero"),
  notes: optionalStringSchema,
});
export type IssuePayAdvanceFormValues = z.infer<typeof issuePayAdvanceFormSchema>;

export const openEscrowAccountFormSchema = z.object({
  workerId: z.string().min(1, "Worker is required"),
  targetAmount: z.number().min(0, "Target amount cannot be negative"),
  annualInterestRate: z
    .number()
    .min(0, "Interest rate cannot be negative")
    .max(100, "Interest rate cannot exceed 100")
    .optional()
    .nullable(),
  openedDate: z.number().int().optional().nullable(),
});
export type OpenEscrowAccountFormValues = z.infer<typeof openEscrowAccountFormSchema>;

export const settlementControlFormSchema = z.object({
  payPeriodFrequency: payPeriodFrequencySchema,
  periodEndDayOfWeek: z.number().int().min(0).max(6),
  payDelayDays: z.number().int().min(0).max(30),
  payTrigger: settlementPayTriggerSchema,
  autoGenerateBatches: z.boolean(),
  autoApproveClean: z.boolean(),
  autoAttachAccruals: z.boolean(),
  autoPostOnApprove: z.boolean(),
  allowNegativeNet: z.boolean(),
  varianceThresholdPct: z.number().min(0),
  varianceLookbackWeeks: z.number().int().min(1).max(52),
  defaultEscrowInterestRate: z.number().min(0).max(100),
  escrowInterestFrequencyMonths: z.number().int().min(1).max(3),
});
export type SettlementControlFormValues = z.infer<typeof settlementControlFormSchema>;

export const dashControlFormSchema = z
  .object({
    requireLoadAcknowledgment: z.boolean(),
    allowLoadRefusals: z.boolean(),
    allowStopActions: z.boolean(),
    allowLoadDocumentUpload: z.boolean(),
    allowLoadComments: z.boolean(),
    showLoadPay: z.boolean(),
    showPayEstimates: z.boolean(),
    allowExpenseSubmission: z.boolean(),
    requireExpenseReceipt: z.boolean(),
    allowSettlementDisputes: z.boolean(),
    allowProfileDocumentUpload: z.boolean(),
    allowContactInfoEdit: z.boolean(),
    allowPtoRequests: z.boolean(),
    sendCredentialReminders: z.boolean(),
    enableDetentionAlerts: z.boolean(),
    detentionAlertThresholdMinutes: z.number().int().min(15).max(1440),
  })
  .superRefine((values, ctx) => {
    if (values.allowLoadRefusals && !values.requireLoadAcknowledgment) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["allowLoadRefusals"],
        message: "Load refusals require load acknowledgment to be enabled",
      });
    }
    if (values.showPayEstimates && !values.showLoadPay) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["showPayEstimates"],
        message: "Pay estimates require per-load pay visibility to be enabled",
      });
    }
    if (values.requireExpenseReceipt && !values.allowExpenseSubmission) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["requireExpenseReceipt"],
        message: "Receipt requirement only applies when expense submission is enabled",
      });
    }
  });
export type DashControlFormValues = z.infer<typeof dashControlFormSchema>;

export const generateBatchFormSchema = z.object({
  name: optionalStringSchema,
  notes: optionalStringSchema,
});
export type GenerateBatchFormValues = z.infer<typeof generateBatchFormSchema>;
