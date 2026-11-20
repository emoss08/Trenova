import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";

export const caseFormatSchema = z.enum(["AsEntered", "Upper", "Lower", "TitleCase"]);

export type CaseFormat = z.infer<typeof caseFormatSchema>;

export const emailCaseFormatSchema = caseFormatSchema.exclude(["TitleCase"]);

export type EmailCaseFormat = z.infer<typeof emailCaseFormatSchema>;

export const dataEntryControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  codeCase: caseFormatSchema,
  nameCase: caseFormatSchema,
  emailCase: emailCaseFormatSchema,
  cityCase: caseFormatSchema,
});

export type DataEntryControl = z.infer<typeof dataEntryControlSchema>;
