import {
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryState,
  useQueryStates,
} from "nuqs";

const designerTabs = ["templates", "documents"] as const;
const inspectorTabs = [
  "overview",
  "controls",
  "raw",
  "formatted",
  "segments",
  "diagnostics",
  "payload",
  "provenance",
] as const;

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
  templateTransactionSet: parseAsString.withDefault(""),
  templateDirection: parseAsString.withDefault(""),
};

export type TemplateDesignerUrlState = {
  templateId: string;
  versionId: string;
  segmentId: string;
  elementPosition: number;
  templateSearch: string;
  templateStatus: string;
  templateTransactionSet: string;
  templateDirection: string;
};

export type TemplateDesignerUrlStatePatch = Partial<TemplateDesignerUrlState>;

type TemplateDesignerSelectionState = Pick<
  TemplateDesignerUrlState,
  "templateId" | "versionId" | "segmentId" | "elementPosition"
>;

type TemplateDesignerSelectionPatch = Partial<TemplateDesignerSelectionState>;

type Identified = {
  id: string;
};

type SegmentSelection = Identified & {
  elements: Array<{ position: number }>;
};

export function getChangedTemplateUrlStatePatch(
  state: TemplateDesignerUrlState,
  patch: TemplateDesignerUrlStatePatch,
): TemplateDesignerUrlStatePatch | null {
  const changedEntries = Object.entries(patch).filter(
    ([key, value]) =>
      value !== undefined && state[key as keyof TemplateDesignerUrlState] !== value,
  );
  return changedEntries.length > 0
    ? (Object.fromEntries(changedEntries) as TemplateDesignerUrlStatePatch)
    : null;
}

export function getTemplateDesignerSelectionPatch({
  templateId,
  versionId,
  segmentId,
  elementPosition,
  templates,
  versions,
  segments,
  segmentsReady,
}: TemplateDesignerSelectionState & {
  templates: Identified[];
  versions: Identified[];
  segments: SegmentSelection[];
  segmentsReady: boolean;
}): TemplateDesignerSelectionPatch | null {
  const patch: TemplateDesignerSelectionPatch = {};
  const firstTemplate = templates[0];
  if (
    firstTemplate &&
    (!templateId || !templates.some((template) => template.id === templateId))
  ) {
    patch.templateId = firstTemplate.id;
    patch.versionId = "";
    patch.segmentId = "";
    patch.elementPosition = 0;
    return getChangedSelectionPatch(
      { templateId, versionId, segmentId, elementPosition },
      patch,
    );
  }

  const firstVersion = versions[0];
  if (
    firstVersion &&
    (!versionId || !versions.some((version) => version.id === versionId))
  ) {
    patch.versionId = firstVersion.id;
    patch.segmentId = "";
    patch.elementPosition = 0;
    return getChangedSelectionPatch(
      { templateId, versionId, segmentId, elementPosition },
      patch,
    );
  }

  if (!segmentsReady) {
    return null;
  }

  if (segments.length === 0) {
    patch.segmentId = "";
    patch.elementPosition = 0;
    return getChangedSelectionPatch(
      { templateId, versionId, segmentId, elementPosition },
      patch,
    );
  }

  const selectedSegment = segments.find((segment) => segment.id === segmentId) ?? segments[0];
  patch.segmentId = selectedSegment.id;

  const firstElementPosition = selectedSegment.elements[0]?.position ?? 0;
  const selectedElement = selectedSegment.elements.find(
    (element) => element.position === elementPosition,
  );
  patch.elementPosition = selectedElement ? elementPosition : firstElementPosition;

  return getChangedSelectionPatch(
    { templateId, versionId, segmentId, elementPosition },
    patch,
  );
}

function getChangedSelectionPatch(
  state: TemplateDesignerSelectionState,
  patch: TemplateDesignerSelectionPatch,
): TemplateDesignerSelectionPatch | null {
  const changedEntries = Object.entries(patch).filter(
    ([key, value]) =>
      value !== undefined && state[key as keyof TemplateDesignerSelectionState] !== value,
  );
  return changedEntries.length > 0
    ? (Object.fromEntries(changedEntries) as TemplateDesignerSelectionPatch)
    : null;
}

export const documentArchiveUrlStateParser = {
  archivePartnerId: parseAsString.withDefault(""),
  archiveTransactionSet: parseAsString.withDefault(""),
  archiveDirection: parseAsString.withDefault(""),
  archiveStatus: parseAsString.withDefault(""),
  archiveGeneratedFrom: parseAsString.withDefault(""),
  archiveGeneratedTo: parseAsString.withDefault(""),
  archiveQuery: parseAsString.withDefault(""),
  inspectorTab: parseAsStringLiteral(inspectorTabs).withDefault("overview"),
  inspectorSegment: parseAsInteger.withDefault(1),
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
