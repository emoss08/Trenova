import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";

type UseEDIDocumentArchiveQueriesParams = {
  messagesQueryString: string;
};

export function useEDIDocumentArchiveQueries({
  messagesQueryString,
}: UseEDIDocumentArchiveQueriesParams) {
  const partnersQuery = useQuery(queries.edi.partnerOptions());
  const profilesQuery = useQuery(
    queries.edi.documentProfiles("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const templatesQuery = useQuery(
    queries.edi.templates("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const messagesQuery = useQuery(queries.edi.messages(messagesQueryString));

  return {
    partnersQuery,
    profilesQuery,
    templatesQuery,
    messagesQuery,
  };
}

export function useEDIMessageDetailQuery(messageId: string) {
  return useQuery({
    ...queries.edi.message(messageId),
    enabled: !!messageId,
  });
}
