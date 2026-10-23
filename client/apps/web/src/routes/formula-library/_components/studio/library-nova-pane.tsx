import { TariffNovaPane } from "@/components/tariff-studio/nova-pane";
import { apiService } from "@/services/api";
import { useT } from "@trenova/shared/i18n/use-t";
import { useLibraryStudio } from "./library-studio-context";

/** Nova in the library studio's rail, about this library's source. */
export function LibraryNovaPane() {
  const t = useT();
  const { library, novaRequest, clearNovaRequest, setNovaBusy, askNova, unsavedNovaKey, check } =
    useLibraryStudio();
  const empty = (check.outline?.items.length ?? 0) <= 1;

  return (
    <TariffNovaPane
      kind="library"
      recordId={library?.id ?? null}
      unsavedKey={unsavedNovaKey}
      openThread={(id) => apiService.formulaLibraryService.openAssistantThread(id)}
      request={novaRequest}
      onRequestSent={clearNovaRequest}
      onWorkingChange={setNovaBusy}
      ask={askNova}
      presets={
        empty
          ? [
              t("A detention function: free time, an hourly rate and 15 minute increments"),
              t("Hazmat and residential delivery fees our tariffs can share"),
              t("A fuel surcharge table by price band"),
            ]
          : [
              t("Explain this library, function by function"),
              t("Suggest tests for the edge cases"),
              t("Which of these functions could refuse to rate, and why?"),
            ]
      }
      emptyTitle={empty ? t("What should templates share?") : t("What should we change?")}
      emptyBody={t(
        "Nova reads the library as it is and makes the changes you ask for: its functions, constants, types and tests. Every change is checked and tested before you see it, can be undone, and nothing is saved until you save.",
      )}
    />
  );
}
