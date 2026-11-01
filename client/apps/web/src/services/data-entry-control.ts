import {
  DataEntryControlDocument,
  UpdateDataEntryControlDocument,
  type UpdateDataEntryControlInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { dataEntryControlSchema, type DataEntryControl } from "@/types/data-entry-control";

export function toDataEntryControlInput(data: DataEntryControl): UpdateDataEntryControlInput {
  return {
    version: data.version ?? 0,
    codeCase: data.codeCase,
    nameCase: data.nameCase,
    emailCase: data.emailCase,
    cityCase: data.cityCase,
  };
}

export class DataEntryControlService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: DataEntryControlDocument,
      operationName: "DataEntryControl",
      signal: options?.signal,
    });

    return safeParse(dataEntryControlSchema, response.dataEntryControl, "Data Entry Control");
  }

  public async update(data: DataEntryControl) {
    const response = await requestGraphQL({
      document: UpdateDataEntryControlDocument,
      operationName: "UpdateDataEntryControl",
      variables: { input: toDataEntryControlInput(data) },
    });

    return safeParse(dataEntryControlSchema, response.updateDataEntryControl, "Data Entry Control");
  }
}
