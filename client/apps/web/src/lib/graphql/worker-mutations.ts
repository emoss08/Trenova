import {
  ApproveWorkerPtoDocument,
  PatchWorkerDocument,
  RejectWorkerPtoDocument,
  type WorkerPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Worker, WorkerPTO } from "@trenova/shared/types/worker";

type PatchWorkerResponse = {
  patchWorker: unknown;
};

type ApproveWorkerPTOResponse = {
  approveWorkerPTO: unknown;
};

type RejectWorkerPTOResponse = {
  rejectWorkerPTO: unknown;
};

export async function patchWorker(
  id: Worker["id"],
  input: WorkerPatchInput,
): Promise<Worker> {
  const data = await requestGraphQL<PatchWorkerResponse>({
    document: PatchWorkerDocument,
    operationName: "PatchWorker",
    variables: {
      id,
      input,
    },
  });

  return data.patchWorker as Worker;
}

export async function approveWorkerPTO(id: WorkerPTO["id"]): Promise<WorkerPTO> {
  const data = await requestGraphQL<ApproveWorkerPTOResponse>({
    document: ApproveWorkerPtoDocument,
    operationName: "ApproveWorkerPto",
    variables: {
      id,
    },
  });

  return data.approveWorkerPTO as WorkerPTO;
}

export async function rejectWorkerPTO(
  id: WorkerPTO["id"],
  reason: string,
): Promise<WorkerPTO> {
  const data = await requestGraphQL<RejectWorkerPTOResponse>({
    document: RejectWorkerPtoDocument,
    operationName: "RejectWorkerPto",
    variables: {
      id,
      reason,
    },
  });

  return data.rejectWorkerPTO as WorkerPTO;
}
