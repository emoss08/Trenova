import { z } from "zod";

/**
 * The two testing roll-ups the server keeps on the worker profile. They live in
 * their own leaf module because the worker schema imports them and the fuller
 * drug-and-alcohol types import the worker schema — putting them together would
 * make that a cycle.
 */
export const drugAlcoholStatusSchema = z.enum(["Unknown", "Clear", "Pending", "Prohibited"]);
export type DrugAlcoholStatus = z.infer<typeof drugAlcoholStatusSchema>;

export const returnToDutyStatusSchema = z.enum([
  "NotRequired",
  "SAPEvaluation",
  "RTDTestRequired",
  "FollowUpTesting",
  "Complete",
]);
export type ReturnToDutyStatus = z.infer<typeof returnToDutyStatusSchema>;
