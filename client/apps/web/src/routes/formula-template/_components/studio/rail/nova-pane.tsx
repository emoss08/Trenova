import { TariffNovaPane } from "@/components/tariff-studio/nova-pane";
import { apiService } from "@/services/api";
import { useT } from "@trenova/shared/i18n/use-t";
import { useStudio } from "../studio-context";

/** Nova in the template Studio's rail, about this template's tariff. */
export function NovaPane() {
  const t = useT();
  const { template, novaRequest, clearNovaRequest, setNovaBusy, askNova, unsavedNovaKey, check } =
    useStudio();
  const empty = (check.outline?.items.length ?? 0) <= 1 && check.inputs.length === 0;

  return (
    <TariffNovaPane
      kind="tariff"
      recordId={template?.id ?? null}
      unsavedKey={unsavedNovaKey}
      openThread={(id) => apiService.formulaTemplateService.openAssistantThread(id)}
      request={novaRequest}
      onRequestSent={clearNovaRequest}
      onWorkingChange={setNovaBusy}
      ask={askNova}
      presets={
        empty
          ? [
              t("Per mile with an 18% fuel surcharge and a $250 minimum"),
              t("LTL weight breaks from a rate per CWT, with deficit weight rating"),
              t("$75 per stop after the first two"),
            ]
          : [
              t("Explain this tariff, statement by statement"),
              t("Add a weekend pickup fee"),
              t("Suggest tests for the cases this tariff could get wrong"),
            ]
      }
      emptyTitle={empty ? t("How should this price a load?") : t("What should we change?")}
      emptyBody={t(
        "Nova reads the tariff as it is and makes the changes you ask for: its inputs, its charges, its total rules and its tests. Every change is checked and tested before you see it, can be undone, and nothing is saved until you save.",
      )}
    />
  );
}
