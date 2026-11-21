import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { DocumentControl } from "@/types/document-control";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type DocumentIntelligenceTab = "reading" | "filing" | "drafts";

export type DocumentIntelligenceField = keyof DocumentControl & string;

export type DocumentIntelligenceTabSpec = SettingsTabSpec<DocumentIntelligenceField> & {
  value: DocumentIntelligenceTab;
};

const TAB_LABELS = defineLabels({
  reading: "Reading",
  filing: "Filing",
  drafts: "Shipment drafts",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DocumentIntelligenceTab, readonly DocumentIntelligenceField[]> = {
  reading: ["enableDocumentIntelligence", "enableOcr", "enableFullTextIndexing"],
  filing: [
    "enableAutoClassification",
    "enableAiAssistedClassification",
    "enableAutoDocumentTypeAssociate",
    "enableAutoCreateDocumentTypes",
  ],
  drafts: [
    "enableShipmentDraftExtraction",
    "shipmentDraftAllowedResources",
    "enableAiAssistedExtraction",
  ],
};

const TAB_ORDER: readonly DocumentIntelligenceTab[] = ["reading", "filing", "drafts"];

/**
 * The tabs of the page, in order: whether documents are read at all, how a read document is
 * filed, and which documents become shipment drafts. Every organization that can open the
 * page has Document Intelligence, which is all the AI-assisted settings need on the server,
 * so no tab depends on the plan. The form and its loading state both read this.
 */
export function documentIntelligenceTabs(): DocumentIntelligenceTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs of the page, for the form and its loading state alike. */
export function useDocumentIntelligenceTabs(): DocumentIntelligenceTabSpec[] {
  return documentIntelligenceTabs();
}
