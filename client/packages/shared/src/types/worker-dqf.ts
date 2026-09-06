import { z } from "zod";

export const employmentVerificationStatusSchema = z.enum([
  "Pending",
  "Requested",
  "Received",
  "NoResponse",
  "NotApplicable",
]);
export type EmploymentVerificationStatus = z.infer<typeof employmentVerificationStatusSchema>;

export const employmentVerificationMethodSchema = z.enum([
  "Email",
  "Fax",
  "Mail",
  "Phone",
  "Portal",
  "Other",
]);
export type EmploymentVerificationMethod = z.infer<typeof employmentVerificationMethodSchema>;

export const dqfItemStatusSchema = z.enum([
  "Satisfied",
  "ExpiringSoon",
  "Expired",
  "Missing",
  "Outstanding",
  "NotApplicable",
]);
export type DQFItemStatus = z.infer<typeof dqfItemStatusSchema>;

/**
 * One previous employer's investigation. The refinements mirror the server's:
 * a request that was never sent cannot be awaiting a response, and an answer
 * that never arrived cannot be recorded as received — either would leave a file
 * that reads as investigated and is not.
 */
export const employmentVerificationFormSchema = z
  .object({
    employerName: z.string().min(1, "Employer is required").max(150),
    employerDotNumber: z.string().max(20).nullable(),
    employerMcNumber: z.string().max(20).nullable(),
    contactName: z.string().max(100).nullable(),
    contactPhone: z.string().max(30).nullable(),
    contactEmail: z.string().max(150).nullable(),
    employedFrom: z.number().nullable(),
    employedTo: z.number().nullable(),
    wasDotRegulated: z.boolean(),
    status: employmentVerificationStatusSchema,
    method: employmentVerificationMethodSchema,
    requestedAt: z.number().nullable(),
    responseReceivedAt: z.number().nullable(),
    drugAlcoholResponseReceivedAt: z.number().nullable(),
    hadAccidents: z.boolean(),
    accidentCount: z.number().min(0),
    hadDrugAlcoholViolations: z.boolean(),
    findings: z.string().nullable(),
    notes: z.string().nullable(),
  })
  .refine(
    (values) =>
      values.employedFrom === null ||
      values.employedTo === null ||
      values.employedTo >= values.employedFrom,
    {
      message: "The end of the employment cannot pre-date its start",
      path: ["employedTo"],
    },
  )
  .refine(
    (values) =>
      values.status === "Pending" ||
      values.status === "NotApplicable" ||
      Boolean(values.requestedAt),
    {
      message: "Record when the request was sent",
      path: ["requestedAt"],
    },
  )
  .refine((values) => values.status !== "Received" || Boolean(values.responseReceivedAt), {
    message: "Record when the response arrived",
    path: ["responseReceivedAt"],
  })
  .refine((values) => !values.hadAccidents || values.accidentCount > 0, {
    message: "Record how many accidents the employer reported",
    path: ["accidentCount"],
  });
export type EmploymentVerificationFormValues = z.infer<typeof employmentVerificationFormSchema>;
