import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  CreateTcaSubscriptionDocument,
  TcaAllowlistedTablesDocument,
  TcaSubscriptionFieldsFragmentDoc,
  UpdateTcaSubscriptionDocument,
  type SaveTcaSubscriptionInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  tcaAllowlistedTableSchema,
  tcaSubscriptionSchema,
  type TCASubscription,
  type TCASubscriptionFormValues,
} from "@/types/table-change-alert";
import { z } from "zod";

export function toTCASubscriptionInput(
  values: TCASubscriptionFormValues,
): SaveTcaSubscriptionInput {
  return {
    name: values.name,
    tableName: values.tableName,
    recordId: values.recordId || null,
    eventTypes: values.eventTypes,
    conditions: values.conditions ?? [],
    conditionMatch: values.conditionMatch ?? "all",
    watchedColumns: values.watchedColumns ?? [],
    customTitle: values.customTitle ?? "",
    customMessage: values.customMessage ?? "",
    topic: values.topic ?? "",
    priority: values.priority ?? "medium",
    status: values.status ?? "Active",
  };
}

export class TableChangeAlertService {
  public async listAllowlistedTables(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: TcaAllowlistedTablesDocument,
      operationName: "TCAAllowlistedTables",
      signal: options?.signal,
    });

    return z.array(tcaAllowlistedTableSchema).parse(response.tcaAllowlistedTables);
  }

  public async createSubscription(data: TCASubscriptionFormValues) {
    const response = await requestGraphQL({
      document: CreateTcaSubscriptionDocument,
      operationName: "CreateTCASubscription",
      variables: { input: toTCASubscriptionInput(data) },
    });

    return safeParse(
      tcaSubscriptionSchema,
      getFragmentData(TcaSubscriptionFieldsFragmentDoc, response.createTcaSubscription),
      "TCA Subscription",
    );
  }

  public async updateSubscription(
    id: TCASubscription["id"],
    version: number,
    data: TCASubscriptionFormValues,
  ) {
    const response = await requestGraphQL({
      document: UpdateTcaSubscriptionDocument,
      operationName: "UpdateTCASubscription",
      variables: { id, version, input: toTCASubscriptionInput(data) },
    });

    return safeParse(
      tcaSubscriptionSchema,
      getFragmentData(TcaSubscriptionFieldsFragmentDoc, response.updateTcaSubscription),
      "TCA Subscription",
    );
  }
}
