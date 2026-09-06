import { z } from "zod";

export const policyAudienceSchema = z.enum(["All", "Employees", "Contractors"]);
export type PolicyAudienceValue = z.infer<typeof policyAudienceSchema>;

export const profileChangeStatusSchema = z.enum(["Pending", "Approved", "Rejected", "Withdrawn"]);
export type ProfileChangeStatusValue = z.infer<typeof profileChangeStatusSchema>;

/**
 * A policy the carrier publishes. Either text or an attached document, never
 * neither: a signature has to be on something. The version label is what an
 * acknowledgement records, so the form asks for it up front rather than
 * inventing one.
 */
export const workerPolicyFormSchema = z
  .object({
    code: z.string().min(1, "A code is required").max(30),
    title: z.string().min(1, "A title is required").max(150),
    summary: z.string().max(1000).nullable(),
    body: z.string().max(20000).nullable(),
    documentId: z.string().nullable(),
    versionLabel: z.string().min(1, "A version is required").max(30),
    requiresSignature: z.boolean(),
    appliesTo: policyAudienceSchema,
    effectiveFrom: z.number().int().positive("An effective date is required"),
    status: z.enum(["Active", "Inactive"]),
  })
  .refine((values) => Boolean(values.documentId) || Boolean(values.body?.trim()), {
    message:
      "A policy needs either text or an attached document — a signature has to be on something",
    path: ["body"],
  });
export type WorkerPolicyFormValues = z.infer<typeof workerPolicyFormSchema>;

/**
 * Deciding a driver's request to change their own record. A rejection needs a
 * reason: a driver whose record did not change deserves to know why.
 */
export const decideProfileChangeFormSchema = z
  .object({
    approve: z.boolean(),
    note: z.string().max(500).nullable(),
  })
  .refine((values) => values.approve || Boolean(values.note?.trim()), {
    message: "Turning a request down needs a reason",
    path: ["note"],
  });
export type DecideProfileChangeFormValues = z.infer<typeof decideProfileChangeFormSchema>;

/** Signing a policy from the portal: a typed full name and an explicit agreement. */
export const acknowledgePolicyFormSchema = z.object({
  signatureName: z.string().max(150),
  agreed: z.literal(true, { error: "Tick the box to confirm you have read it" }),
});
export type AcknowledgePolicyFormValues = z.infer<typeof acknowledgePolicyFormSchema>;
