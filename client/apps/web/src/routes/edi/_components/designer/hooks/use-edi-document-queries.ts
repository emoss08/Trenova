import { edi } from "@/lib/queries/edi";
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
  const profilesQuery = useQuery(edi.documentProfiles(profilesQueryString));
  const templatesQuery = useQuery(edi.templates(templateFilters));
  const messagesQuery = useQuery(edi.messages(messagesQueryString));

  return {
    profilesQuery,
    templatesQuery,
    messagesQuery,
  };
}

export function useEDIMessageDetailQuery(messageId: string) {
  return useQuery({
    ...edi.message(messageId),
    enabled: !!messageId,
  });
}

export function useEDIMessageInspectionQuery(messageId: string) {
  return useQuery({
    ...edi.messageInspection(messageId),
    enabled: !!messageId,
  });
}
