import { optionalIdSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";

/** How a script that showed the person something ended. */
export const analysisRunStatusSchema = z.enum([
  "Succeeded",
  "Errored",
  "Stopped",
  "TimedOut",
  "Failed",
]);

export const analysisDatasetSourceSchema = z.enum(["ReportRun", "Attachment"]);

/** A report run or attached file a script read, by the name it read it by. */
export const analysisDatasetSchema = z.object({
  name: z.string(),
  source: analysisDatasetSourceSchema.catch("ReportRun"),
  sourceId: z.string().optional().default(""),
  label: z.string().optional().default(""),
  rows: z.number().int().optional().default(0),
});

export const analysisDatasetsSchema = z
  .array(analysisDatasetSchema)
  .nullish()
  .transform((value) => value ?? []);

/** How an analysis value reads: a percent is a fraction, a date ISO 8601 text. */
export const analysisValueKindSchema = z
  .enum(["text", "number", "money", "count", "percent", "date", "datetime", "bool"])
  .catch("text");

const analysisCellSchema = z.union([z.string(), z.number(), z.boolean(), z.null()]).catch(null);

export const analysisTableSchema = z.object({
  title: z.string().optional().default(""),
  columns: z.array(
    z.object({
      name: z.string(),
      kind: analysisValueKindSchema,
      currency: z.string().optional().default(""),
    }),
  ),
  rows: z.array(z.array(analysisCellSchema)),
  totalRows: z.number().int(),
});

export const analysisFigureSchema = z.object({
  label: z.string(),
  value: analysisCellSchema,
  kind: analysisValueKindSchema,
  currency: z.string().optional().default(""),
  note: z.string().optional().default(""),
});

export const analysisChartKindSchema = z.enum(["bar", "line", "area", "scatter", "pie"]);

/**
 * A chart a script showed with show_chart, as what it plots: the values along its axis and a
 * number, or nothing, for each of them in each series. The client draws it in the reader's theme.
 */
export const analysisChartSchema = z.object({
  kind: analysisChartKindSchema,
  title: z.string().optional().default(""),
  x: z.object({
    label: z.string().optional().default(""),
    kind: analysisValueKindSchema,
    values: z.array(analysisCellSchema),
  }),
  series: z.array(
    z.object({
      name: z.string(),
      values: z.array(z.number().nullable().catch(null)),
    }),
  ),
  valueKind: analysisValueKindSchema,
  currency: z.string().optional().default(""),
  stacked: z.boolean().optional().default(false),
  horizontal: z.boolean().optional().default(false),
});

export const analysisBlockSchema = z.object({
  kind: z.enum(["chart", "table"]),
  index: z.number().int(),
});

export const analysisTablesSchema = z
  .array(analysisTableSchema)
  .nullish()
  .transform((value) => value ?? []);

export const analysisFiguresSchema = z
  .array(analysisFigureSchema)
  .nullish()
  .transform((value) => value ?? []);

export const analysisChartsSchema = z
  .array(analysisChartSchema)
  .nullish()
  .transform((value) => value ?? []);

export const analysisLayoutSchema = z
  .array(analysisBlockSchema)
  .nullish()
  .transform((value) => value ?? []);

/** One analysis, whole, as the server keeps it. */
export const analysisRunSchema = z.object({
  id: z.string(),
  status: analysisRunStatusSchema,
  title: z.string().default(""),
  purpose: z.string().default(""),
  code: z.string().default(""),
  datasets: analysisDatasetsSchema,
  output: z.string().default(""),
  tables: analysisTablesSchema,
  figures: analysisFiguresSchema,
  charts: analysisChartsSchema,
  layout: analysisLayoutSchema,
  errorName: z.string().default(""),
  errorValue: z.string().default(""),
  traceback: z.string().default(""),
  failure: z.string().default(""),
  durationMs: z.number().int(),
  agentDefinitionId: optionalIdSchema,
  agentRunId: optionalIdSchema,
  threadId: optionalIdSchema,
  requestedById: optionalIdSchema,
  startedAt: z.number().int(),
  finishedAt: z.number().int(),
});

/** An analysis as the history lists it: how it ended, what it was for and when it ran. */
export const analysisRunSummarySchema = analysisRunSchema.pick({
  id: true,
  status: true,
  title: true,
  purpose: true,
  durationMs: true,
  agentDefinitionId: true,
  agentRunId: true,
  threadId: true,
  requestedById: true,
  startedAt: true,
  finishedAt: true,
});

export const analysisRunsResponseSchema = z.object({
  items: z
    .array(analysisRunSummarySchema)
    .nullish()
    .transform((value) => value ?? []),
  nextBefore: z.number().int(),
  nextBeforeId: optionalIdSchema,
});

export type AnalysisRun = z.infer<typeof analysisRunSchema>;
export type AnalysisRunSummary = z.infer<typeof analysisRunSummarySchema>;
export type AnalysisRunStatus = z.infer<typeof analysisRunStatusSchema>;
export type AnalysisRunsResponse = z.infer<typeof analysisRunsResponseSchema>;
export type AnalysisDataset = z.infer<typeof analysisDatasetSchema>;
export type AnalysisValueKind = z.infer<typeof analysisValueKindSchema>;
export type AnalysisTable = z.infer<typeof analysisTableSchema>;
export type AnalysisFigure = z.infer<typeof analysisFigureSchema>;
export type AnalysisChart = z.infer<typeof analysisChartSchema>;
export type AnalysisChartKind = z.infer<typeof analysisChartKindSchema>;
export type AnalysisBlock = z.infer<typeof analysisBlockSchema>;
