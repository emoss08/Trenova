import { SystemUpdateStatusDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { updateStatusSchema, type UpdateStatus } from "@/types/update";

export const updateService = {
  getUpdateStatus: async (): Promise<UpdateStatus> => {
    const response = await requestGraphQL({
      document: SystemUpdateStatusDocument,
      operationName: "SystemUpdateStatus",
    });
    return safeParse(updateStatusSchema, response.systemUpdateStatus, "Update Status");
  },
};
