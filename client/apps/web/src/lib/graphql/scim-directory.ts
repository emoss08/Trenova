import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  ScimDirectoryFieldsFragmentDoc,
  ScimDirectoriesListDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { SCIMDirectory } from "@trenova/shared/types/iam";

type RequestOptions = { signal?: AbortSignal };

export type SCIMDirectoryPage = {
  directories: SCIMDirectory[];
  hasNextPage: boolean;
  endCursor: string | null;
};

/**
 * One page of the organization's SCIM directories, newest first, read through the
 * `scimDirectories` connection: pass the previous page's `endCursor` as `after` for the next.
 */
export async function listSCIMDirectoriesGraphQL(
  { first, after }: { first: number; after?: string | null },
  options?: RequestOptions,
): Promise<SCIMDirectoryPage> {
  const data = await requestGraphQL({
    document: ScimDirectoriesListDocument,
    operationName: "SCIMDirectoriesList",
    variables: { input: { first, ...(after ? { after } : {}) } },
    signal: options?.signal,
  });
  const connection = data.scimDirectories;

  return {
    directories: connection.edges.map((edge) =>
      getFragmentData(ScimDirectoryFieldsFragmentDoc, edge.node),
    ),
    hasNextPage: connection.pageInfo.hasNextPage,
    endCursor: connection.pageInfo.endCursor ?? null,
  };
}
