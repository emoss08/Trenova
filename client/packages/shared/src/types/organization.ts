import { z } from "zod";
import { capabilityFlagSchema } from "./helpers";
import { userSchema } from "./user";
import { translate } from "@trenova/shared/i18n/runtime";

export const userOrganizationSchema = z.object({
  id: z.string().min(1, { error: () => translate("Organization ID is required") }),
  name: z.string().min(1, { error: () => translate("Organization name is required") }),
  city: z.string().min(1, { error: () => translate("City is required") }),
  state: z.string().min(1, { error: () => translate("State is required") }),
  logoUrl: z.string().nullish(),
  iconUrl: z.string().nullish(),
  isDefault: z.boolean(),
  isCurrent: z.boolean(),
});

export type UserOrganization = z.infer<typeof userOrganizationSchema>;

export const userOrganizationsResponseSchema = z.array(userOrganizationSchema);

export const switchOrganizationRequestSchema = z.object({
  organizationId: z.string(),
});

export type SwitchOrganizationRequest = z.infer<typeof switchOrganizationRequestSchema>;

export const switchOrganizationResponseSchema = userSchema;

export type SwitchOrganizationResponse = z.infer<typeof switchOrganizationResponseSchema>;

/** The values the server keeps when an organization is made before anyone has filled it in. */
const PLACEHOLDER_SCAC = "TBDX";
const PLACEHOLDER_DOT_NUMBER = "0";
const PLACEHOLDER_CITY = "Pending";
const PLACEHOLDER_POSTAL_CODE = "00000";

const LOGIN_SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

const organizationSettingsFields = z.object({
  id: z.string().min(1, { error: () => translate("Organization ID is required") }),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
  bucketName: z.string().nullish(),
  businessUnitId: z.string().nullish(),
  loginSlug: z
    .string()
    .max(100, { error: () => translate("Sign-in address must be 100 characters or fewer") })
    .regex(LOGIN_SLUG_PATTERN, {
      error: () =>
        translate("Sign-in address may only contain lowercase letters, numbers, and hyphens"),
    })
    .or(z.literal(""))
    .nullish(),

  name: z
    .string()
    .min(1, { error: () => translate("Name is required") })
    .max(100, { error: () => translate("Name must be 100 characters or fewer") }),
  scacCode: z
    .string()
    .min(1, { error: () => translate("SCAC code is required") })
    .length(4, { error: () => translate("SCAC code must be 4 characters") })
    .refine((value) => value.trim().toUpperCase() !== PLACEHOLDER_SCAC, {
      error: () => translate("Enter your company's SCAC code"),
    }),
  dotNumber: z
    .string()
    .min(1, { error: () => translate("DOT number is required") })
    .max(8, { error: () => translate("DOT number must be between 1 and 8 digits") })
    .regex(/^\d+$/, { error: () => translate("DOT number must be numeric") })
    .refine((value) => value !== PLACEHOLDER_DOT_NUMBER, {
      error: () => translate("Enter your company's DOT number"),
    }),
  logoUrl: z.string().nullish(),
  iconUrl: z.string().nullish(),
  addressLine1: z
    .string()
    .min(1, { error: () => translate("Address line 1 is required") })
    .max(150, { error: () => translate("Address line 1 must be 150 characters or fewer") }),
  addressLine2: z.string().nullish(),
  city: z
    .string()
    .min(1, { error: () => translate("City is required") })
    .refine((value) => value !== PLACEHOLDER_CITY, {
      error: () => translate("Enter the city of your registered address"),
    }),
  stateId: z.string().min(1, { error: () => translate("State is required") }),
  postalCode: z
    .string()
    .min(1, { error: () => translate("ZIP code is required") })
    .refine((value) => value !== PLACEHOLDER_POSTAL_CODE, {
      error: () => translate("Enter the ZIP code of your registered address"),
    }),
  timezone: z.string().min(1, { error: () => translate("Timezone is required") }),
  taxId: z.string().nullish(),
  brokerageEnabled: capabilityFlagSchema,
  assetOperationsEnabled: capabilityFlagSchema,
  state: z
    .object({
      id: z.string(),
      name: z.string(),
      abbreviation: z.string().nullish(),
      countryName: z.string().nullish(),
    })
    .nullish(),
});

/**
 * What the settings form accepts: the organization validator's rules
 * (`tenant.Organization.Validate` and the service's operating-capability rule), so a save the
 * server would refuse is refused before it is sent, on the field the server names.
 */
export const organizationSettingsSchema = organizationSettingsFields.superRefine((values, ctx) => {
  if (values.brokerageEnabled || values.assetOperationsEnabled) {
    return;
  }

  const message = translate(
    "Turn on brokerage or asset operations. An organization with neither cannot move freight.",
  );
  ctx.addIssue({ code: "custom", path: ["brokerageEnabled"], message });
  ctx.addIssue({ code: "custom", path: ["assetOperationsEnabled"], message });
});

export type OrganizationSettings = z.infer<typeof organizationSettingsSchema>;

// What the server may hold, as opposed to what the settings form accepts. An
// organization made for a managed instance starts with only its name: the address is
// blank until an administrator fills it in. Reading it with the form's rules would
// refuse the response, and the settings page that fixes it could never load.
export const storedOrganizationSettingsSchema = organizationSettingsFields.extend({
  name: z.string(),
  loginSlug: z.string().nullish(),
  scacCode: z.string(),
  dotNumber: z.string(),
  addressLine1: z.string(),
  city: z.string(),
  stateId: z.string(),
  postalCode: z.string(),
  timezone: z.string(),
});

export type OrganizationSelectOption = {
  id: string;
  name: string;
  scacCode?: string | null;
  city?: string | null;
  state?: {
    name?: string | null;
    abbreviation?: string | null;
  } | null;
};

/**
 * An organization's two pictures. The logo is its full mark, at the top of the documents and
 * emails it sends; the icon is its square mark, drawn where a logo has no room.
 */
export type OrganizationBrandImage = "logo" | "icon";

/** Where each picture can be read from; an empty string for one never uploaded. */
export const organizationBrandingSchema = z.object({
  logoUrl: z.string(),
  iconUrl: z.string(),
});

export type OrganizationBranding = z.infer<typeof organizationBrandingSchema>;

export const tenantLoginMetadataSchema = z.object({
  organizationId: z.string(),
  organizationName: z.string(),
  organizationSlug: z.string(),
  enabledProviders: z.array(z.string()).default([]),
  passwordEnabled: z.boolean(),
  enforceSso: z.boolean(),
});

export type TenantLoginMetadata = z.infer<typeof tenantLoginMetadataSchema>;
