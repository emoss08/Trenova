import {
  DispatchStyleDocument,
  ResetDispatchStyleDocument,
  type DispatchStyleQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DispatchStyle = DispatchStyleQuery["dispatchStyle"];
export type DispatchStyleWeight = DispatchStyle["weights"][number];
export type DispatchStyleComponent = DispatchStyleWeight["component"];

type RequestOptions = { signal?: AbortSignal };

export async function fetchDispatchStyle(options?: RequestOptions): Promise<DispatchStyle> {
  const data = await requestGraphQL({
    document: DispatchStyleDocument,
    operationName: "DispatchStyle",
    signal: options?.signal,
  });
  return data.dispatchStyle;
}

export async function resetDispatchStyle(): Promise<DispatchStyle> {
  const data = await requestGraphQL({
    document: ResetDispatchStyleDocument,
    operationName: "ResetDispatchStyle",
  });
  return data.resetDispatchStyle;
}
