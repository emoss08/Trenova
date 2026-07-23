import { z } from "zod";
import { optionalStringSchema, timestampSchema, versionSchema } from "./helpers";

export const caseFormatSchema = z.enum(["AsEntered", "Upper", "Lower", "TitleCase"]);

export type CaseFormat = z.infer<typeof caseFormatSchema>;

export const dataEntryControlSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  codeCase: caseFormatSchema,
  nameCase: caseFormatSchema,
  emailCase: caseFormatSchema,
  cityCase: caseFormatSchema,
});

export type DataEntryControl = z.infer<typeof dataEntryControlSchema>;
