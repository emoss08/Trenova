import {
  CancelDotRandomDrawDocument,
  CancelDotTestDocument,
  CompleteClearinghouseQueryDocument,
  CreateDotRandomPoolDocument,
  DotRandomDrawDocument,
  DotRandomDrawsDocument,
  DotRandomPoolsDocument,
  FinalizeDotRandomDrawDocument,
  RecordClearinghouseQueryDocument,
  RecordDotTestDocument,
  RecordDotTestResultDocument,
  RecordDotViolationDocument,
  RunDotRandomDrawDocument,
  UpdateDotRandomDrawEntryDocument,
  UpdateDotRandomPoolDocument,
  UpdateDotViolationDocument,
  WorkerDrugAlcoholFileDocument,
  type CancelDotRandomDrawMutation,
  type CancelDotRandomDrawMutationVariables,
  type CancelDotTestMutation,
  type CancelDotTestMutationVariables,
  type ClearinghouseQueryFieldsFragment,
  type CompleteClearinghouseQueryInput,
  type CompleteClearinghouseQueryMutation,
  type CompleteClearinghouseQueryMutationVariables,
  type CreateDotRandomPoolMutation,
  type CreateDotRandomPoolMutationVariables,
  type DotRandomPoolInput,
  type DotRandomDrawEntryFieldsFragment,
  type DotRandomDrawFieldsFragment,
  type DotRandomDrawQuery,
  type DotRandomDrawQueryVariables,
  type DotRandomDrawsQuery,
  type DotRandomDrawsQueryVariables,
  type DotRandomPoolFieldsFragment,
  type DotRandomPoolsQuery,
  type DotRandomPoolsQueryVariables,
  type FinalizeDotRandomDrawMutation,
  type FinalizeDotRandomDrawMutationVariables,
  type RecordClearinghouseQueryInput,
  type RecordClearinghouseQueryMutation,
  type RecordClearinghouseQueryMutationVariables,
  type RecordDotTestInput,
  type RecordDotTestMutation,
  type RecordDotTestMutationVariables,
  type RecordDotTestResultInput,
  type RecordDotTestResultMutation,
  type RecordDotTestResultMutationVariables,
  type RecordDotViolationInput,
  type RecordDotViolationMutation,
  type RecordDotViolationMutationVariables,
  type RunDotRandomDrawInput,
  type RunDotRandomDrawMutation,
  type RunDotRandomDrawMutationVariables,
  type UpdateDotRandomDrawEntryInput,
  type UpdateDotRandomDrawEntryMutation,
  type UpdateDotRandomDrawEntryMutationVariables,
  type UpdateDotRandomPoolMutation,
  type UpdateDotRandomPoolMutationVariables,
  type UpdateDotViolationInput,
  type UpdateDotViolationMutation,
  type UpdateDotViolationMutationVariables,
  type WorkerDotTestFieldsFragment,
  type WorkerDotViolationFieldsFragment,
  type WorkerDrugAlcoholFileQuery,
  type WorkerDrugAlcoholFileQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DotTestRow = WorkerDotTestFieldsFragment;
export type DotViolationRow = WorkerDotViolationFieldsFragment;
export type ClearinghouseQueryRow = ClearinghouseQueryFieldsFragment;
export type RandomPoolRow = DotRandomPoolFieldsFragment;
export type RandomDrawRow = DotRandomDrawFieldsFragment;
export type RandomDrawEntryRow = DotRandomDrawEntryFieldsFragment;
export type RandomDrawListRow = DotRandomDrawsQuery["dotRandomDraws"][number];
export type DrugAlcoholStanding = WorkerDrugAlcoholFileQuery["workerDrugAlcoholStanding"];

/**
 * The whole testing file for one worker. The five reads travel together
 * because the panel shows them together and the standing is only meaningful
 * beside the records it was derived from.
 */
export type WorkerDrugAlcoholFile = {
  standing: DrugAlcoholStanding;
  tests: DotTestRow[];
  violations: DotViolationRow[];
  queries: ClearinghouseQueryRow[];
  selections: RandomDrawEntryRow[];
};

export const WORKER_DRUG_ALCOHOL_KEY = "worker-drug-alcohol";
export const DOT_RANDOM_POOLS_KEY = "dot-random-pools";
export const DOT_RANDOM_DRAWS_KEY = "dot-random-draws";
export const DOT_RANDOM_DRAW_KEY = "dot-random-draw";

export async function fetchWorkerDrugAlcoholFile(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerDrugAlcoholFile> {
  const data = await requestGraphQL<
    WorkerDrugAlcoholFileQuery,
    WorkerDrugAlcoholFileQueryVariables
  >({
    document: WorkerDrugAlcoholFileDocument,
    operationName: "WorkerDrugAlcoholFile",
    variables: { workerId },
    signal: options?.signal,
  });

  return {
    standing: data.workerDrugAlcoholStanding,
    tests: data.workerDotTests as DotTestRow[],
    violations: data.workerDotViolations as DotViolationRow[],
    queries: data.workerClearinghouseQueries as ClearinghouseQueryRow[],
    selections: data.workerRandomSelections as RandomDrawEntryRow[],
  };
}

export async function recordDotTest(input: RecordDotTestInput): Promise<DotTestRow> {
  const data = await requestGraphQL<RecordDotTestMutation, RecordDotTestMutationVariables>({
    document: RecordDotTestDocument,
    operationName: "RecordDotTest",
    variables: { input },
  });
  return data.recordDotTest as DotTestRow;
}

export async function recordDotTestResult(input: RecordDotTestResultInput): Promise<DotTestRow> {
  const data = await requestGraphQL<
    RecordDotTestResultMutation,
    RecordDotTestResultMutationVariables
  >({
    document: RecordDotTestResultDocument,
    operationName: "RecordDotTestResult",
    variables: { input },
  });
  return data.recordDotTestResult as DotTestRow;
}

export async function cancelDotTest(id: string, reason: string): Promise<DotTestRow> {
  const data = await requestGraphQL<CancelDotTestMutation, CancelDotTestMutationVariables>({
    document: CancelDotTestDocument,
    operationName: "CancelDotTest",
    variables: { id, reason },
  });
  return data.cancelDotTest as DotTestRow;
}

export async function recordDotViolation(input: RecordDotViolationInput): Promise<DotViolationRow> {
  const data = await requestGraphQL<
    RecordDotViolationMutation,
    RecordDotViolationMutationVariables
  >({
    document: RecordDotViolationDocument,
    operationName: "RecordDotViolation",
    variables: { input },
  });
  return data.recordDotViolation as DotViolationRow;
}

export async function updateDotViolation(input: UpdateDotViolationInput): Promise<DotViolationRow> {
  const data = await requestGraphQL<
    UpdateDotViolationMutation,
    UpdateDotViolationMutationVariables
  >({
    document: UpdateDotViolationDocument,
    operationName: "UpdateDotViolation",
    variables: { input },
  });
  return data.updateDotViolation as DotViolationRow;
}

export async function recordClearinghouseQuery(
  input: RecordClearinghouseQueryInput,
): Promise<ClearinghouseQueryRow> {
  const data = await requestGraphQL<
    RecordClearinghouseQueryMutation,
    RecordClearinghouseQueryMutationVariables
  >({
    document: RecordClearinghouseQueryDocument,
    operationName: "RecordClearinghouseQuery",
    variables: { input },
  });
  return data.recordClearinghouseQuery as ClearinghouseQueryRow;
}

export async function completeClearinghouseQuery(
  input: CompleteClearinghouseQueryInput,
): Promise<ClearinghouseQueryRow> {
  const data = await requestGraphQL<
    CompleteClearinghouseQueryMutation,
    CompleteClearinghouseQueryMutationVariables
  >({
    document: CompleteClearinghouseQueryDocument,
    operationName: "CompleteClearinghouseQuery",
    variables: { input },
  });
  return data.completeClearinghouseQuery as ClearinghouseQueryRow;
}

export async function fetchDotRandomPools(options?: {
  signal?: AbortSignal;
}): Promise<RandomPoolRow[]> {
  const data = await requestGraphQL<DotRandomPoolsQuery, DotRandomPoolsQueryVariables>({
    document: DotRandomPoolsDocument,
    operationName: "DotRandomPools",
    variables: { input: { first: 100 } },
    signal: options?.signal,
  });
  return data.dotRandomPools.edges.map((edge) => edge.node as RandomPoolRow);
}

export async function createDotRandomPool(input: DotRandomPoolInput): Promise<RandomPoolRow> {
  const data = await requestGraphQL<
    CreateDotRandomPoolMutation,
    CreateDotRandomPoolMutationVariables
  >({
    document: CreateDotRandomPoolDocument,
    operationName: "CreateDotRandomPool",
    variables: { input },
  });
  return data.createDotRandomPool as RandomPoolRow;
}

export async function updateDotRandomPool(
  id: string,
  version: number,
  input: DotRandomPoolInput,
): Promise<RandomPoolRow> {
  const data = await requestGraphQL<
    UpdateDotRandomPoolMutation,
    UpdateDotRandomPoolMutationVariables
  >({
    document: UpdateDotRandomPoolDocument,
    operationName: "UpdateDotRandomPool",
    variables: { id, version, input },
  });
  return data.updateDotRandomPool as RandomPoolRow;
}

export async function fetchDotRandomDraws(
  poolId?: string,
  options?: { signal?: AbortSignal },
): Promise<DotRandomDrawsQuery["dotRandomDraws"]> {
  const data = await requestGraphQL<DotRandomDrawsQuery, DotRandomDrawsQueryVariables>({
    document: DotRandomDrawsDocument,
    operationName: "DotRandomDraws",
    variables: { poolId: poolId ?? null },
    signal: options?.signal,
  });
  return data.dotRandomDraws;
}

export async function fetchDotRandomDraw(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DotRandomDrawQuery["dotRandomDraw"]> {
  const data = await requestGraphQL<DotRandomDrawQuery, DotRandomDrawQueryVariables>({
    document: DotRandomDrawDocument,
    operationName: "DotRandomDraw",
    variables: { id },
    signal: options?.signal,
  });
  return data.dotRandomDraw;
}

export async function runDotRandomDraw(
  input: RunDotRandomDrawInput,
): Promise<RunDotRandomDrawMutation["runDotRandomDraw"]> {
  const data = await requestGraphQL<RunDotRandomDrawMutation, RunDotRandomDrawMutationVariables>({
    document: RunDotRandomDrawDocument,
    operationName: "RunDotRandomDraw",
    variables: { input },
  });
  return data.runDotRandomDraw;
}

export async function finalizeDotRandomDraw(id: string): Promise<RandomDrawRow> {
  const data = await requestGraphQL<
    FinalizeDotRandomDrawMutation,
    FinalizeDotRandomDrawMutationVariables
  >({
    document: FinalizeDotRandomDrawDocument,
    operationName: "FinalizeDotRandomDraw",
    variables: { id },
  });
  return data.finalizeDotRandomDraw as RandomDrawRow;
}

export async function cancelDotRandomDraw(id: string, reason: string): Promise<RandomDrawRow> {
  const data = await requestGraphQL<
    CancelDotRandomDrawMutation,
    CancelDotRandomDrawMutationVariables
  >({
    document: CancelDotRandomDrawDocument,
    operationName: "CancelDotRandomDraw",
    variables: { id, reason },
  });
  return data.cancelDotRandomDraw as RandomDrawRow;
}

export async function updateDotRandomDrawEntry(
  input: UpdateDotRandomDrawEntryInput,
): Promise<RandomDrawEntryRow> {
  const data = await requestGraphQL<
    UpdateDotRandomDrawEntryMutation,
    UpdateDotRandomDrawEntryMutationVariables
  >({
    document: UpdateDotRandomDrawEntryDocument,
    operationName: "UpdateDotRandomDrawEntry",
    variables: { input },
  });
  return data.updateDotRandomDrawEntry as RandomDrawEntryRow;
}
