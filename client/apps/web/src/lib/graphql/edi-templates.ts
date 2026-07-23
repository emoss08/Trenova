import {
  EdiTemplateListDocument,
  type EdiDocumentDirection,
  type EdiTemplateStatus,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import { ediTemplateSchema, type EDITemplate } from "@/types/edi";
import { z } from "zod";

const ediTemplateListPageSchema = z.object({
  results: z.array(ediTemplateSchema),
  totalCount: z.number().nullish(),
  hasNextPage: z.boolean(),
  endCursor: z.string().nullish(),
});

export type EdiTemplateListPage = z.infer<typeof ediTemplateListPageSchema>;

export type ListEdiTemplatesParams = {
  first: number;
  after?: string | null;
  query?: string;
  status?: string;
  transactionSet?: string;
  direction?: string;
};

type EdiTemplateListResponse = {
  ediTemplates: {
    edges: Array<{ node: unknown }>;
    totalCount?: number | null;
    pageInfo: { hasNextPage: boolean; endCursor?: string | null };
  };
};

export async function listEdiTemplatesGraphQL(
  params: ListEdiTemplatesParams,
): Promise<EdiTemplateListPage> {
  const data = await requestGraphQL<EdiTemplateListResponse>({
    document: EdiTemplateListDocument,
    operationName: "EdiTemplateList",
    variables: {
      input: {
        first: params.first,
        after: params.after ?? null,
        query: params.query?.trim() || null,
      },
      status: (params.status || null) as EdiTemplateStatus | null,
      transactionSet: params.transactionSet || null,
      direction: (params.direction || null) as EdiDocumentDirection | null,
    },
  });

  const connection = data.ediTemplates;
  return safeParse(
    ediTemplateListPageSchema,
    {
      results: connection.edges.map((edge) => edge.node),
      totalCount: connection.totalCount,
      hasNextPage: connection.pageInfo.hasNextPage,
      endCursor: connection.pageInfo.endCursor,
    },
    "EdiTemplateListPage",
  );
}

export function flattenEdiTemplatePages(
  pages: EdiTemplateListPage[] | undefined,
): EDITemplate[] {
  return pages?.flatMap((page) => page.results) ?? [];
}
