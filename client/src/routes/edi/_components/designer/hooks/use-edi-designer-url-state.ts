import {
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryState,
  useQueryStates,
} from "nuqs";

const designerTabs = ["templates", "documents"] as const;

export const ediDesignerUrlStateParser = {
  designerTab: parseAsStringLiteral(designerTabs).withDefault("templates"),
};

export const templateDesignerUrlStateParser = {
  templateId: parseAsString.withDefault(""),
  versionId: parseAsString.withDefault(""),
  segmentId: parseAsString.withDefault(""),
  elementPosition: parseAsInteger.withDefault(0),
  templateSearch: parseAsString.withDefault(""),
  templateStatus: parseAsString.withDefault(""),
};

export const documentArchiveUrlStateParser = {
  archivePartnerId: parseAsString.withDefault(""),
  archiveTransactionSet: parseAsString.withDefault("204"),
  archiveDirection: parseAsString.withDefault("Outbound"),
  archiveStatus: parseAsString.withDefault(""),
  archiveGeneratedFrom: parseAsString.withDefault(""),
  archiveGeneratedTo: parseAsString.withDefault(""),
  archiveQuery: parseAsString.withDefault(""),
};

export function useEDIDesignerUrlState() {
  return useQueryStates(ediDesignerUrlStateParser, {
    clearOnDefault: true,
    history: "push",
  });
}

export function useTemplateDesignerUrlState() {
  return useQueryStates(templateDesignerUrlStateParser, {
    clearOnDefault: true,
    history: "replace",
  });
}

export function useDocumentArchiveUrlState() {
  const archiveState = useQueryStates(documentArchiveUrlStateParser, {
    clearOnDefault: true,
    history: "replace",
  });
  const messageState = useQueryState(
    "messageId",
    parseAsString.withDefault("").withOptions({
      clearOnDefault: true,
      history: "push",
    }),
  );

  return [archiveState, messageState] as const;
}
