import { DeactivateStoredMileageDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export class StoredMileageService {
  public async deactivate(id: string) {
    await requestGraphQL({
      document: DeactivateStoredMileageDocument,
      operationName: "DeactivateStoredMileage",
      variables: { id },
    });
  }
}
