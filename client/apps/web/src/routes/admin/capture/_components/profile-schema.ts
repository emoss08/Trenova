import type {
  CapturePixelType,
  CaptureProfile,
  CaptureProfileInput,
  CaptureProfileStatus,
  CaptureSeparatorStrategy,
} from "@/lib/graphql/capture";
import { z } from "zod";

export const PIXEL_TYPES = [
  "BlackWhite",
  "Grayscale",
  "Color",
] as const satisfies readonly CapturePixelType[];

export const PROFILE_STATUSES = [
  "Active",
  "Inactive",
] as const satisfies readonly CaptureProfileStatus[];

export const SEPARATOR_STRATEGIES = [
  "PatchCode",
  "CoverSheet",
  "BlankPage",
  "FixedPageCount",
] as const satisfies readonly CaptureSeparatorStrategy[];

/** The resolutions offered: what document scanners support and what paperwork needs. */
export const PROFILE_RESOLUTIONS = [150, 200, 300, 400, 600] as const;

/** The server's bounds (`capture.CaptureProfile.Validate`), checked here first. */
const MAX_NAME = 100;
const MAX_DESCRIPTION = 500;
const MAX_FIXED_PAGE_COUNT = 500;

export const profileFormSchema = z
  .object({
    name: z.string().trim().min(1, { error: "Name is required" }).max(MAX_NAME),
    description: z.string().max(MAX_DESCRIPTION),
    status: z.enum(PROFILE_STATUSES),
    isDefault: z.boolean(),
    dpi: z.number().int().min(100).max(600),
    pixelType: z.enum(PIXEL_TYPES),
    duplex: z.boolean(),
    useFeeder: z.boolean(),
    discardBlankPages: z.boolean(),
    jpegQuality: z
      .number()
      .int()
      .min(30, { error: "At least 30" })
      .max(95, { error: "At most 95" }),
    showDriverUi: z.boolean(),
    // The checkbox group stores an empty choice as null.
    separatorStrategies: z
      .array(z.enum(SEPARATOR_STRATEGIES))
      .nullable()
      .transform((value) => value ?? []),
    fixedPageCount: z.number().int().min(0).max(MAX_FIXED_PAGE_COUNT),
  })
  .refine((value) => !value.isDefault || value.status === "Active", {
    error: "An inactive profile cannot be the default",
    path: ["isDefault"],
  })
  .refine(
    (value) =>
      !value.separatorStrategies.includes("FixedPageCount") ||
      (value.fixedPageCount >= 1 && value.fixedPageCount <= MAX_FIXED_PAGE_COUNT),
    {
      error: `Pages per document must be between 1 and ${MAX_FIXED_PAGE_COUNT}`,
      path: ["fixedPageCount"],
    },
  );

export type ProfileFormInput = z.input<typeof profileFormSchema>;
export type ProfileFormValues = z.output<typeof profileFormSchema>;

/**
 * What a new profile starts as: the setting scanner vendors recommend for
 * paperwork that is mostly text, the same one the server applies to a profile
 * saved with nothing set.
 */
export function newProfileDefaults(): ProfileFormInput {
  return {
    name: "",
    description: "",
    status: "Active",
    isDefault: false,
    dpi: 300,
    pixelType: "BlackWhite",
    duplex: true,
    useFeeder: true,
    discardBlankPages: true,
    jpegQuality: 80,
    showDriverUi: false,
    separatorStrategies: ["PatchCode", "CoverSheet"],
    fixedPageCount: 0,
  };
}

export function profileFormValues(profile: CaptureProfile): ProfileFormInput {
  return {
    name: profile.name,
    description: profile.description,
    status: profile.status,
    isDefault: profile.isDefault,
    dpi: profile.dpi,
    pixelType: profile.pixelType,
    duplex: profile.duplex,
    useFeeder: profile.useFeeder,
    discardBlankPages: profile.discardBlankPages,
    jpegQuality: profile.jpegQuality,
    showDriverUi: profile.showDriverUi,
    separatorStrategies: [...profile.separatorStrategies],
    fixedPageCount: profile.fixedPageCount,
  };
}

/**
 * The profile as the server takes it. A page count kept while splitting by
 * count is off would be refused, so it is cleared rather than sent.
 */
export function profileInput(values: ProfileFormValues): CaptureProfileInput {
  const byCount = values.separatorStrategies.includes("FixedPageCount");

  return {
    name: values.name.trim(),
    description: values.description.trim() === "" ? null : values.description.trim(),
    status: values.status,
    isDefault: values.isDefault,
    dpi: values.dpi,
    pixelType: values.pixelType,
    duplex: values.duplex,
    useFeeder: values.useFeeder,
    discardBlankPages: values.discardBlankPages,
    jpegQuality: values.jpegQuality,
    showDriverUi: values.showDriverUi,
    separatorStrategies: values.separatorStrategies,
    fixedPageCount: byCount ? values.fixedPageCount : 0,
  };
}
