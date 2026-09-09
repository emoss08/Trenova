import { z } from "zod";
import { fleetCodeRelationSchema } from "./fleet-code";
import {
  customFieldsSchema,
  nullableEnumSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";
import { usStateRelationSchema } from "./us-state";
import { drugAlcoholStatusSchema, returnToDutyStatusSchema } from "./worker-drug-alcohol-status";
import { safetyRatingSchema } from "./worker-safety";
import { workerTrainingHealthSchema } from "./worker-training-health";

// function validatePhoneNumber(value: string): string | undefined {
//   try {
//     const phone = parsePhoneNumber(value);
//     if (!phone) {
//       return "Invalid phone number format";
//     }
//     if (!phone.isPossible()) {
//       return "Phone number has an incorrect number of digits";
//     }
//     if (!phone.isValid()) {
//       return (
//         "Phone number is not a valid number for " +
//         (phone.country ?? "the given country code")
//       );
//     }
//     return undefined;
//   } catch {
//     return "Invalid phone number format";
//   }
// }

// const phoneNumberSchema = z
//   .string()
//   .superRefine((val, ctx) => {
//     const error = validatePhoneNumber(val);
//     if (error) {
//       ctx.addIssue({ code: "custom", message: error });
//     }
//   })
//   .or(z.literal(""));

export const workerTypeSchema = z.enum(["Employee", "Contractor"]);
export type WorkerType = z.infer<typeof workerTypeSchema>;

export const genderSchema = z.enum(["Male", "Female"]);
export type Gender = z.infer<typeof genderSchema>;

export const driverTypeSchema = z.enum(["Local", "Regional", "OTR", "Team"]);
export type DriverType = z.infer<typeof driverTypeSchema>;

export const workerLeaveTypeSchema = z.enum([
  "FMLA",
  "Medical",
  "Military",
  "Parental",
  "Personal",
  "Other",
]);
export type WorkerLeaveType = z.infer<typeof workerLeaveTypeSchema>;

export const WORKER_LEAVE_TYPE_LABELS: Record<WorkerLeaveType, string> = {
  FMLA: "FMLA",
  Medical: "Medical",
  Military: "Military",
  Parental: "Parental",
  Personal: "Personal",
  Other: "Other",
};

export const cdlClassSchema = z.enum(["A", "B", "C"]);
export type CDLClass = z.infer<typeof cdlClassSchema>;

export const endorsementTypeSchema = z.enum(["O", "N", "H", "X", "P", "T"]);
export type EndorsementType = z.infer<typeof endorsementTypeSchema>;

export const complianceStatusSchema = z.enum(["Compliant", "NonCompliant", "Pending"]);
export type ComplianceStatus = z.infer<typeof complianceStatusSchema>;

export const ptoStatusSchema = z.enum(["Requested", "Approved", "Rejected", "Cancelled"]);
export type PTOStatus = z.infer<typeof ptoStatusSchema>;

export const ptoTypeSchema = z.enum([
  "Personal",
  "Vacation",
  "Sick",
  "Holiday",
  "Bereavement",
  "Maternity",
  "Paternity",
]);
export type PTOType = z.infer<typeof ptoTypeSchema>;

export const workerProfileSchema = z.object({
  id: optionalStringSchema,
  workerId: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  licenseStateId: nullableStringSchema,
  dob: z.number().int().positive({
    message: "Date of birth is required",
  }),
  licenseNumber: z.string().min(1, {
    message: "License number is required",
  }),
  cdlClass: cdlClassSchema,
  cdlRestrictions: nullableStringSchema,
  endorsement: endorsementTypeSchema,
  hazmatExpiry: nullableIntegerSchema,
  licenseExpiry: z.number().int().positive({
    message: "License expiry is required",
  }),
  medicalCardExpiry: nullableIntegerSchema,
  medicalExaminerName: nullableStringSchema,
  medicalExaminerNpi: nullableStringSchema,
  twicCardNumber: nullableStringSchema,
  twicExpiry: nullableIntegerSchema,
  hireDate: z.number().int().positive({
    message: "Hire date is required",
  }),
  terminationDate: nullableIntegerSchema,
  physicalDueDate: nullableIntegerSchema,
  mvrDueDate: nullableIntegerSchema,
  complianceStatus: complianceStatusSchema,
  /**
   * Roster roll-ups. The server keeps these on the profile so the workers list
   * can filter and sort on training and safety without replaying every record
   * for every row. They are a cache; the worker overview has the live answer.
   */
  trainingHealth: workerTrainingHealthSchema.default("Current"),
  safetyRating: safetyRatingSchema.default("Excellent"),
  safetyScore: z.number().int().default(100),
  nextCredentialExpiry: nullableIntegerSchema,
  nextTrainingDue: nullableIntegerSchema,
  drugAlcoholStatus: drugAlcoholStatusSchema.default("Unknown"),
  returnToDutyStatus: returnToDutyStatusSchema.default("NotRequired"),
  lastClearinghouseQueryAt: nullableIntegerSchema,
  nextClearinghouseQueryDue: nullableIntegerSchema,
  isQualified: z.boolean().default(false),
  disqualificationReason: nullableStringSchema,
  lastComplianceCheck: z.number().int().default(0),
  lastMvrCheck: z.number().int().default(0),
  lastDrugTest: z.number().int().default(0),
  eldExempt: z.boolean().default(false),
  shortHaulExempt: z.boolean().default(false),
  version: z.number().int().optional(),
  createdAt: z.number().int().optional(),
  updatedAt: z.number().int().optional(),

  licenseState: usStateRelationSchema,
});

export type WorkerProfile = z.infer<typeof workerProfileSchema>;

export const ptoActorSchema = z.object({
  id: z.string(),
  name: z.string(),
});

export type PTOActor = z.infer<typeof ptoActorSchema>;

export const workerPtoSchema = z.object({
  id: optionalStringSchema,
  workerId: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  approverId: nullableStringSchema,
  rejectorId: nullableStringSchema,
  cancelledById: nullableStringSchema,
  status: ptoStatusSchema,
  type: ptoTypeSchema,
  startDate: z.number().int().positive({
    message: "Start date is required",
  }),
  endDate: z.number().int().positive({
    message: "End date is required",
  }),
  reason: z.string().min(1, {
    message: "Reason is required",
  }),
  rejectionReason: nullableStringSchema,
  cancellationReason: nullableStringSchema,
  days: z.string().optional(),
  balanceAfterDays: nullableStringSchema,
  autoApproved: z.boolean().optional(),
  version: z.number().int().optional(),
  createdAt: z.number().int().optional(),
  updatedAt: z.number().int().optional(),
  approver: ptoActorSchema.nullish(),
  rejector: ptoActorSchema.nullish(),
  cancelledBy: ptoActorSchema.nullish(),
  get worker() {
    return workerSchema.nullish();
  },
});

export type WorkerPTO = z.infer<typeof workerPtoSchema>;

export const workerSchema = z.object({
  ...tenantInfoSchema.shape,
  stateId: z.string().min(1, {
    message: "State is required",
  }),
  fleetCodeId: nullableStringSchema,
  managerId: nullableStringSchema,
  positionId: nullableStringSchema,
  status: statusSchema,
  type: workerTypeSchema,
  driverType: driverTypeSchema,
  leaveType: nullableEnumSchema(workerLeaveTypeSchema),
  profilePicUrl: nullableStringSchema,
  firstName: z.string().min(1, {
    message: "First name is required",
  }),
  lastName: z.string().min(1, {
    message: "Last name is required",
  }),
  wholeName: optionalStringSchema,
  addressLine1: z.string().min(1, {
    message: "Address is required",
  }),
  addressLine2: nullableStringSchema,
  city: z.string().min(1, {
    message: "City is required",
  }),
  postalCode: z
    .string()
    .min(1, {
      message: "Postal code is required",
    })
    .regex(/^\d{5}(-\d{4})?$/, {
      message: "Invalid US postal code (e.g., 12345 or 12345-6789)",
    }),
  email: nullableStringSchema,
  phoneNumber: z.string().min(1, {
    error: "Phone number is required",
  }),
  emergencyContactName: nullableStringSchema,
  emergencyContactPhone: z.string().nullish(),
  externalId: nullableStringSchema,
  assignmentBlocked: nullableStringSchema,
  gender: genderSchema,
  canBeAssigned: z.boolean().default(false),
  availableForDispatch: z.boolean().default(true),

  state: usStateRelationSchema,
  fleetCode: fleetCodeRelationSchema,
  profile: workerProfileSchema.nullish(),
  pto: z.array(workerPtoSchema).nullish(),
  customFields: customFieldsSchema,
});

export type Worker = z.infer<typeof workerSchema>;

export const ptoChartDataRequestSchema = z.object({
  startDateFrom: z.number().int().positive(),
  startDateTo: z.number().int().positive(),
  type: ptoTypeSchema.optional(),
  timezone: z.string().optional(),
  workerId: workerSchema.shape.id.optional(),
});

export type PTOChartDataRequest = z.infer<typeof ptoChartDataRequestSchema>;

export const ptoChartDataPointSchema = z.object({
  date: z.string(),
  vacation: z.number().int().positive(),
  sick: z.number().int().positive(),
  holiday: z.number().int().positive(),
  bereavement: z.number().int().positive(),
  maternity: z.number().int().positive(),
  paternity: z.number().int().positive(),
  personal: z.number().int().positive(),
  workers: z.record(
    z.string(),
    z.array(
      z.object({
        id: z.string(),
        firstName: z.string(),
        lastName: z.string(),
        ptoType: z.string(),
      }),
    ),
  ),
});

export type PTOChartDataPoint = z.infer<typeof ptoChartDataPointSchema>;

export const PTO_FILTER_MAX_RANGE_DAYS = 120;

export const ptoFilterSchema = z
  .object({
    type: z.string().optional(),
    startDate: z.number().min(1, { error: "Start date is required" }),
    endDate: z.number().min(1, { error: "End date is required" }),
    workerId: z.string().optional(),
    fleetCodeId: z.string().optional(),
  })
  .refine((data) => data.startDate <= data.endDate, {
    message: "Start date must be before end date",
    path: ["endDate"],
  })
  .refine(
    (data) => {
      const diffInSeconds = data.endDate - data.startDate;
      return diffInSeconds <= PTO_FILTER_MAX_RANGE_DAYS * 24 * 60 * 60;
    },
    {
      message: `Date range cannot exceed ${PTO_FILTER_MAX_RANGE_DAYS} days`,
      path: ["endDate"],
    },
  );

export type PTOFilter = z.infer<typeof ptoFilterSchema>;

export type ListUpcomingPTORequest = {
  filter: any;
  type?: PTOType;
  status?: PTOStatus;
  startDate?: number;
  endDate?: number;
  workerId?: string;
  fleetCodeId?: string;
  timezone?: string;
};

export const ptoFormSchema = z
  .object({
    workerId: z.string().min(1, { message: "Worker is required" }),
    type: ptoTypeSchema,
    startDate: z.number().int().positive({ message: "Start date is required" }),
    endDate: z.number().int().positive({ message: "End date is required" }),
    reason: z
      .string()
      .trim()
      .min(1, { message: "Reason is required" })
      .max(255, { message: "Reason must be 255 characters or fewer" }),
  })
  .refine((data) => data.endDate > data.startDate, {
    message: "End date must be after start date",
    path: ["endDate"],
  });

export type PTOFormValues = z.infer<typeof ptoFormSchema>;

export const ptoBulkActionSchema = z.enum(["Approve", "Reject", "Cancel"]);
export type PTOBulkAction = z.infer<typeof ptoBulkActionSchema>;

export const ptoReasonRequestSchema = z.object({
  ptoIds: z.array(z.string().min(1)).min(1, { message: "Select at least one PTO request" }),
  reason: z.string().trim().min(1, { message: "Reason is required" }).max(255, {
    message: "Reason must be 255 characters or fewer",
  }),
});

export type PTOReasonRequest = z.infer<typeof ptoReasonRequestSchema>;

export const ptoCancelRequestSchema = ptoReasonRequestSchema.extend({
  reason: z.string().trim().max(255, { message: "Reason must be 255 characters or fewer" }),
});

export type PTOCancelRequest = z.infer<typeof ptoCancelRequestSchema>;

export const ptoBulkActionResultSchema = z.object({
  ptoId: z.string(),
  success: z.boolean(),
  error: z.string(),
});

export const ptoBulkActionPayloadSchema = z.object({
  results: z.array(ptoBulkActionResultSchema),
  successCount: z.number().int(),
  failureCount: z.number().int(),
});

export type PTOBulkActionPayload = z.infer<typeof ptoBulkActionPayloadSchema>;
