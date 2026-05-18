import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";

type UseEDIDocumentArchiveQueriesParams = {
  messagesQueryString: string;
  profilesQueryString?: string;
  templatesQueryString?: string;
};

export function useEDIDocumentArchiveQueries({
  messagesQueryString,
  profilesQueryString = "?limit=100",
  templatesQueryString = "?limit=100",
}: UseEDIDocumentArchiveQueriesParams) {
  const profilesQuery = useQuery(queries.edi.documentProfiles(profilesQueryString));
  const templatesQuery = useQuery(queries.edi.templates(templatesQueryString));
  const messagesQuery = useQuery(queries.edi.messages(messagesQueryString));

  return {
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
