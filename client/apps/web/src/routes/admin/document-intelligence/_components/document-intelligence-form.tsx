import { SettingsForm } from "@/components/settings/settings-form";
import { documentControl } from "@/lib/queries/document-control";
import { documentControlService } from "@/services/document-control";
import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  useDocumentIntelligenceTabs,
  type DocumentIntelligenceTab,
} from "../document-intelligence-tabs";
import { DraftsTab } from "./drafts-tab";
import { FilingTab } from "./filing-tab";
import { ReadingTab } from "./reading-tab";

const save = (values: DocumentControl) => documentControlService.update(values);

/**
 * Document extraction: what Trenova does with the documents an organization uploads, in one
 * form saved once and split into a tab for each job: reading them, filing them and drafting
 * shipments from rate confirmations. The capture settings on the same record are edited on
 * the Capture page and saved here as they were read.
 */
export default function DocumentIntelligenceForm() {
  const { data } = useSuspenseQuery({ ...documentControl.get() });
  const tabs = useDocumentIntelligenceTabs();

  return (
    <SettingsForm<DocumentControl, DocumentIntelligenceTab>
      values={data}
      schema={documentControlSchema}
      queryKey={documentControl.get._def}
      save={save}
      resourceName="Document extraction"
      tabs={tabs}
      content={(tab, { selectTab }) => {
        const openReading = () => selectTab("reading");
        switch (tab) {
          case "reading":
            return <ReadingTab />;
          case "filing":
            return <FilingTab onOpenReading={openReading} />;
          case "drafts":
            return <DraftsTab onOpenReading={openReading} />;
        }
      }}
    />
  );
}
