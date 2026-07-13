import {
  CreateTableConfigurationDocument,
  DefaultTableConfigurationDocument,
  DeleteTableConfigurationDocument,
  SetDefaultTableConfigurationDocument,
  TableConfigurationDetailDocument,
  TableConfigurationTableDocument,
  UpdateTableConfigurationDocument,
  type CreateTableConfigurationMutation,
  type DefaultTableConfigurationQuery,
  type SetDefaultTableConfigurationMutation,
  type TableConfigurationDetailQuery,
  type TableConfigurationInput,
  type TableConfigurationTableQuery,
  type UpdateTableConfigurationMutation,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import {
  tableConfigurationResponseSchema,
  tableConfigurationSchema,
  type ConfigurationVisibility,
  type TableConfiguration,
  type TableConfigurationFormValues,
} from "@/types/table-configuration";

export type ListTableConfigurationsParams = {
  resource: string;
  visibility?: ConfigurationVisibility;
  limit?: number;
  offset?: number;
  query?: string;
};

function toTableConfigurationInput(
  data: TableConfigurationFormValues,
): TableConfigurationInput {
  return {
    name: data.name,
    description: data.description ?? "",
    resource: data.resource,
    tableConfig: data.tableConfig,
    visibility: data.visibility ?? "Private",
    isDefault: data.isDefault ?? false,
  };
}

export class TableConfigurationService {
  public async list(params: ListTableConfigurationsParams) {
    const response = (await requestGraphQL({
      document: TableConfigurationTableDocument,
      operationName: "TableConfigurationTable",
      variables: {
        input: {
          first: params.limit ?? 20,
          query: params.query || undefined,
        },
        resource: params.resource,
        visibility: params.visibility ?? null,
      },
    })) as TableConfigurationTableQuery;

    const results = response.tableConfigurations.edges.map((edge) => edge.node);
    const totalCount = response.tableConfigurations.totalCount;

    return safeParse(
      tableConfigurationResponseSchema,
      {
        results,
        count: totalCount ?? results.length,
        next: null,
        prev: null,
      },
      "Table Configuration List",
    );
  }

  public async get(id: TableConfiguration["id"]) {
    const response = (await requestGraphQL({
      document: TableConfigurationDetailDocument,
      operationName: "TableConfigurationDetail",
      variables: { id },
    })) as TableConfigurationDetailQuery;

    return safeParse(
      tableConfigurationSchema,
      response.tableConfiguration,
      "Table Configuration",
    );
  }

  public async getDefault(resource: TableConfiguration["resource"]) {
    const response = (await requestGraphQL({
      document: DefaultTableConfigurationDocument,
      operationName: "DefaultTableConfiguration",
      variables: { resource },
    })) as DefaultTableConfigurationQuery;

    if (!response.defaultTableConfiguration) return null;

    return safeParse(
      tableConfigurationSchema,
      response.defaultTableConfiguration,
      "Table Configuration",
    );
  }

  public async create(data: TableConfigurationFormValues) {
    const response = (await requestGraphQL({
      document: CreateTableConfigurationDocument,
      operationName: "CreateTableConfiguration",
      variables: { input: toTableConfigurationInput(data) },
    })) as CreateTableConfigurationMutation;

    return safeParse(
      tableConfigurationSchema,
      response.createTableConfiguration,
      "Table Configuration",
    );
  }

  public async update(
    id: TableConfiguration["id"],
    data: TableConfigurationFormValues,
  ) {
    const response = (await requestGraphQL({
      document: UpdateTableConfigurationDocument,
      operationName: "UpdateTableConfiguration",
      variables: { id, input: toTableConfigurationInput(data) },
    })) as UpdateTableConfigurationMutation;

    return safeParse(
      tableConfigurationSchema,
      response.updateTableConfiguration,
      "Table Configuration",
    );
  }

  public async delete(id: TableConfiguration["id"]) {
    await requestGraphQL({
      document: DeleteTableConfigurationDocument,
      operationName: "DeleteTableConfiguration",
      variables: { id },
    });
  }

  public async setDefault(id: TableConfiguration["id"]) {
    const response = (await requestGraphQL({
      document: SetDefaultTableConfigurationDocument,
      operationName: "SetDefaultTableConfiguration",
      variables: { id },
    })) as SetDefaultTableConfigurationMutation;

    return safeParse(
      tableConfigurationSchema,
      response.setDefaultTableConfiguration,
      "Table Configuration",
    );
  }
}
