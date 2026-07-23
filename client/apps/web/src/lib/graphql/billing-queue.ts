import {
  AssignBillingQueueBillerDocument,
  UpdateBillingQueueStatusDocument,
  type BillingQueueAssignInput,
  type BillingQueueUpdateStatusInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingQueueItemSchema, type BillingQueueItem } from "@trenova/shared/types/billing-queue";

type UpdateBillingQueueStatusResponse = {
  updateBillingQueueStatus: unknown;
};

type AssignBillingQueueBillerResponse = {
  assignBillingQueueBiller: unknown;
};

export async function updateBillingQueueStatusGraphQL(
  id: BillingQueueItem["id"],
  input: BillingQueueUpdateStatusInput,
): Promise<BillingQueueItem> {
  const data = await requestGraphQL<UpdateBillingQueueStatusResponse>({
    document: UpdateBillingQueueStatusDocument,
    operationName: "UpdateBillingQueueStatus",
    variables: { id, input },
  });

  return safeParse(billingQueueItemSchema, data.updateBillingQueueStatus, "BillingQueueItem");
}

export async function assignBillingQueueBillerGraphQL(
  id: BillingQueueItem["id"],
  input: BillingQueueAssignInput,
): Promise<BillingQueueItem> {
  const data = await requestGraphQL<AssignBillingQueueBillerResponse>({
    document: AssignBillingQueueBillerDocument,
    operationName: "AssignBillingQueueBiller",
    variables: { id, input },
  });

  return safeParse(billingQueueItemSchema, data.assignBillingQueueBiller, "BillingQueueItem");
}
