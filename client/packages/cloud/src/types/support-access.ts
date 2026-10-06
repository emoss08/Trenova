import { z } from "zod";

export const AccessMode = z.enum(["read_only", "read_write"]);
export type AccessMode = z.infer<typeof AccessMode>;

const optionalString = z.string().nullish().transform((value) => value ?? "");
const optionalTimestamp = z
  .number()
  .int()
  .nullish()
  .transform((value) => value ?? null);

export const supportGrantSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  grantedById: z.string(),
  accessMode: AccessMode,
  note: optionalString,
  startsAt: z.number().int(),
  expiresAt: z.number().int(),
  revokedAt: optionalTimestamp,
  revokedById: optionalString,
  createdAt: z.number().int(),
});

export type SupportGrant = z.infer<typeof supportGrantSchema>;

export const SessionStatus = z.enum(["active", "ended", "expired"]);
export type SessionStatus = z.infer<typeof SessionStatus>;

export const customerSessionSchema = z.object({
  id: z.string(),
  staffName: z.string(),
  status: SessionStatus,
  mode: AccessMode,
  reason: z.string(),
  ticketReference: optionalString,
  startedAt: z.number().int(),
  expiresAt: z.number().int(),
  lastSeenAt: z.number().int(),
  elevationCount: z.number().int(),
  elevatedUntil: optionalTimestamp,
  elevationReason: optionalString,
  endedAt: optionalTimestamp,
  endReason: optionalString,
});

export type CustomerSession = z.infer<typeof customerSessionSchema>;

export const grantStateSchema = z.object({
  grant: supportGrantSchema.nullable(),
  sessions: z.array(customerSessionSchema),
  history: z.array(supportGrantSchema),
  durationHours: z.array(z.number().int()),
  maxDurationHours: z.number().int(),
  accessModes: z.array(AccessMode),
  serverTime: z.number().int(),
});

export type GrantState = z.infer<typeof grantStateSchema>;

export const createGrantRequestSchema = z.object({
  durationHours: z.number().int().positive(),
  accessMode: AccessMode,
  note: z.string().max(500, { error: "Note must be at most 500 characters" }),
});

export type CreateGrantRequest = z.infer<typeof createGrantRequestSchema>;

export const sessionViewSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  organizationName: optionalString,
  staffName: z.string(),
  mode: AccessMode,
  grantMode: AccessMode,
  reason: z.string(),
  ticketReference: optionalString,
  startedAt: z.number().int(),
  expiresAt: z.number().int(),
  elevatedUntil: optionalTimestamp,
  canElevate: z.boolean(),
  serverTime: z.number().int(),
});

export type SessionView = z.infer<typeof sessionViewSchema>;

export const currentSessionSchema = z.object({
  active: z.boolean(),
  session: sessionViewSchema.nullish().transform((value) => value ?? null),
  endedReason: optionalString,
});

export type CurrentSession = z.infer<typeof currentSessionSchema>;

export const staffProfileSchema = z.object({
  isStaff: z.boolean(),
  role: optionalString,
  mfaEnrolled: z.boolean(),
  sessionVerified: z.boolean(),
  openSessions: z.array(sessionViewSchema),
  maxSessionHours: z.number(),
  elevationMinutes: z.number(),
});

export type StaffProfile = z.infer<typeof staffProfileSchema>;

export const grantedOrganizationSchema = z.object({
  grantId: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  organizationName: z.string(),
  accessMode: AccessMode,
  note: optionalString,
  startsAt: z.number().int(),
  expiresAt: z.number().int(),
});

export type GrantedOrganization = z.infer<typeof grantedOrganizationSchema>;

export const grantedOrganizationsSchema = z.object({
  items: z.array(grantedOrganizationSchema),
});

export const startSessionRequestSchema = z.object({
  organizationId: z.string().min(1),
  businessUnitId: z.string().min(1),
  reason: z
    .string()
    .trim()
    .min(10, { error: "Describe what you are doing in at least 10 characters" })
    .max(500, { error: "Reason must be at most 500 characters" }),
  ticketReference: z.string().trim().max(100, { error: "Ticket must be at most 100 characters" }),
});

export type StartSessionRequest = z.infer<typeof startSessionRequestSchema>;

export const elevateRequestSchema = z.object({
  password: z.string().min(1, { error: "Password is required" }),
  code: z.string().trim().min(6, { error: "Enter the six-digit code from your authenticator app" }),
  reason: z
    .string()
    .trim()
    .min(10, { error: "Describe the change in at least 10 characters" })
    .max(500, { error: "Reason must be at most 500 characters" }),
  ticketReference: z
    .string()
    .trim()
    .min(1, { error: "A ticket reference is required to change customer data" })
    .max(100, { error: "Ticket must be at most 100 characters" }),
});

export type ElevateRequest = z.infer<typeof elevateRequestSchema>;
