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
  const documentTypesQuery = useQuery(queries.edi.documentTypes());
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
  const sourceFieldsQuery = useQuery(
    queries.edi.sourceContextFields(
      "?limit=100&status=Active&transactionSet=204&direction=Outbound",
    ),
  );
  const partnerFieldsQuery = useQuery(
    queries.edi.partnerSettingFields(
      "?limit=100&status=Active&transactionSet=204&direction=Outbound",
    ),
  );

  return {
    templatesQuery,
    documentTypesQuery,
    templateQuery,
    versionsQuery,
    versionQuery,
    sourceFieldsQuery,
    partnerFieldsQuery,
  };
}
