import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";

type UseEDITemplateQueriesParams = {
  templatesQueryString: string;
  selectedTemplateId: string;
  selectedVersionId: string;
};

export function useEDITemplateQueries({
  templatesQueryString,
  selectedTemplateId,
  selectedVersionId,
}: UseEDITemplateQueriesParams) {
  const templatesQuery = useQuery(queries.edi.templates(templatesQueryString));
  const templateQuery = useQuery({
    ...queries.edi.template(selectedTemplateId),
    enabled: !!selectedTemplateId,
  });
  const versionsQuery = useQuery({
    ...queries.edi.templateVersions(selectedTemplateId),
    enabled: !!selectedTemplateId,
  });
  const versionQuery = useQuery({
    ...queries.edi.templateVersion(selectedTemplateId, selectedVersionId),
    enabled: !!selectedTemplateId && !!selectedVersionId,
  });

  return {
    templatesQuery,
    templateQuery,
    versionsQuery,
    versionQuery,
  };
}
