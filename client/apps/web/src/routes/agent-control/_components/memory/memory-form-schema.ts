import type {
  AgentMemoryInput,
  AgentMemoryKind,
  AgentMemorySubjectType,
} from "@trenova/graphql/generated/graphql";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

/**
 * The most characters one memory may hold. The single client copy of the
 * server's `agent.MaxMemoryContentChars`; every field and counter reads it
 * from here. A prompt shows the first 1200 and names the memory's id so the
 * agent can read the rest.
 */
export const MEMORY_CONTENT_LIMIT = 4000;

export const memoryKindValues = [
  "Instruction",
  "Fact",
  "Correction",
  "Procedure",
] as const satisfies readonly AgentMemoryKind[];
export const memorySubjectTypeValues = [
  "Customer",
  "Location",
  "Worker",
  "Carrier",
] as const satisfies readonly AgentMemorySubjectType[];

const nullableText = z.preprocess(
  (value) => (value === "" || value === undefined ? null : value),
  z.string().nullable(),
);

/**
 * What the panel edits. It is the row's own shape with the nullables kept,
 * so the edit panel can reset the form straight from a table row.
 */
export const memoryFormSchema = z
  .object({
    kind: z.enum(memoryKindValues),
    content: z
      .string()
      .trim()
      .min(1, { error: () => translate("Say what the agents should know") })
      .max(MEMORY_CONTENT_LIMIT, {
        error: () => translate("Keep it to {0} characters", MEMORY_CONTENT_LIMIT),
      }),
    subjectType: z.preprocess(
      (value) => (value === "" || value === undefined ? null : value),
      z.enum(memorySubjectTypeValues).nullable(),
    ),
    subjectId: nullableText,
    toolName: z.preprocess((value) => value ?? "", z.string().trim()),
    expiresAt: z
      .number()
      .int()
      .positive()
      .nullish()
      .transform((value) => value ?? null),
    version: z.number().int().nonnegative().default(0),
  })
  .superRefine((values, ctx) => {
    if (values.subjectType && !values.subjectId) {
      ctx.addIssue({
        code: "custom",
        path: ["subjectId"],
        message: translate("Pick the record this is about"),
      });
    }
    if (values.subjectId && !values.subjectType) {
      ctx.addIssue({
        code: "custom",
        path: ["subjectType"],
        message: translate("Say what kind of record this is about"),
      });
    }
  });

export type MemoryFormValues = z.infer<typeof memoryFormSchema>;

export const memoryFormDefaults: MemoryFormValues = {
  kind: "Instruction",
  content: "",
  subjectType: null,
  subjectId: null,
  toolName: "",
  expiresAt: null,
  version: 0,
};

/** What goes over the wire: nothing set is sent as absent, not as "". */
export function toMemoryInput(values: MemoryFormValues): AgentMemoryInput {
  const toolName = values.toolName.trim();

  return {
    kind: values.kind,
    content: values.content,
    subjectType: values.subjectType,
    subjectId: values.subjectType ? values.subjectId : null,
    toolName: toolName === "" ? null : toolName,
    expiresAt: values.expiresAt,
    version: values.version,
  };
}

/** A kept memory as the editor holds it. */
export function memoryValuesFromRow(row: {
  kind: MemoryFormValues["kind"];
  content: string;
  subjectType?: MemoryFormValues["subjectType"] | null;
  subjectId?: string | null;
  toolName?: string | null;
  expiresAt?: number | null;
  version: number;
}): MemoryFormValues {
  return {
    kind: row.kind,
    content: row.content,
    subjectType: row.subjectType ?? null,
    subjectId: row.subjectId ?? null,
    toolName: row.toolName ?? "",
    expiresAt: row.expiresAt ?? null,
    version: row.version,
  };
}
