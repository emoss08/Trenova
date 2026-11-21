import { SettingsReveal } from "@/components/settings/settings-reveal";
import type { DocumentControl } from "@/types/document-control";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * Said at the head of a tab whose settings only act on documents Trenova reads, while reading
 * is off: the settings stay where they are and can be set ahead, and the notice says why
 * they do nothing yet and opens the switch that changes that.
 */
export function ReadingOffNotice({ onOpenReading }: { onOpenReading: () => void }) {
  const t = useT();
  const { control } = useFormContext<DocumentControl>();
  const reading = useWatch({ control, name: "enableDocumentIntelligence" });

  return (
    <SettingsReveal show={!reading}>
      <div className="pb-5">
        <Alert size="sm">
          <AlertDescription>
            {t(
              "Uploaded documents are not read, so these settings do nothing until reading is turned on.",
            )}
          </AlertDescription>
          <AlertAction>
            <Button type="button" variant="secondary" size="sm" onClick={onOpenReading}>
              {t("Open Reading")}
            </Button>
          </AlertAction>
        </Alert>
      </div>
    </SettingsReveal>
  );
}
