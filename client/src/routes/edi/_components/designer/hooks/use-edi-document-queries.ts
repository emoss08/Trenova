import { queries } from "@/lib/queries";
import type { EDITemplateListFilters } from "@/lib/queries/edi";
import { useQuery } from "@tanstack/react-query";

type UseEDIDocumentArchiveQueriesParams = {
  messagesQueryString: string;
  profilesQueryString?: string;
  templateFilters?: EDITemplateListFilters;
};

export function useEDIDocumentArchiveQueries({
  messagesQueryString,
  profilesQueryString = "?limit=100",
  templateFilters,
}: UseEDIDocumentArchiveQueriesParams) {
  const profilesQuery = useQuery(queries.edi.documentProfiles(profilesQueryString));
  const templatesQuery = useQuery(queries.edi.templates(templateFilters));
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

export function useEDIMessageInspectionQuery(messageId: string) {
  return useQuery({
    ...queries.edi.messageInspection(messageId),
    enabled: !!messageId,
  });
}
