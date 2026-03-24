import { z } from "zod";

export const databaseSessionChainSchema = z.object({
  blockedPid: z.number(),
  blockingPid: z.number(),
  databaseName: z.string(),
  blockedState: z.string(),
  blockingState: z.string(),
  blockedWaitEventType: z.string(),
  blockedWaitEvent: z.string(),
  blockedApplicationName: z.string(),
  blockingApplicationName: z.string(),
  blockedUser: z.string(),
  blockingUser: z.string(),
  blockedQueryPreview: z.string(),
  blockingQueryPreview: z.string(),
  blockedTransactionAgeSeconds: z.number(),
  blockingTransactionAgeSeconds: z.number(),
  blockedQueryAgeSeconds: z.number(),
  blockingQueryAgeSeconds: z.number(),
});

export const listDatabaseSessionsResponseSchema = z.object({
  items: z.array(databaseSessionChainSchema).nullable().transform((v) => v ?? []),
});

export const terminateDatabaseSessionResponseSchema = z.object({
  pid: z.number(),
  terminated: z.boolean(),
  message: z.string(),
});

export type DatabaseSessionChain = z.infer<typeof databaseSessionChainSchema>;
export type ListDatabaseSessionsResponse = z.infer<
  typeof listDatabaseSessionsResponseSchema
>;
export type TerminateDatabaseSessionResponse = z.infer<
  typeof terminateDatabaseSessionResponseSchema
>;
