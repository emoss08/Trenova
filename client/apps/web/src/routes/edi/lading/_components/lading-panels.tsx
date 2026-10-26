import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import type {
  LadingCheck,
  LadingContext,
  LadingLocation,
  LadingProjectInput,
  LadingRunValue,
  LadingTestReport,
} from "@trenova/shared/edi/lading";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { cn } from "@trenova/shared/lib/utils";
import { LadingReleasePanel } from "./lading-release-panel";
import { LadingRunPanel } from "./lading-run-panel";
import { LadingSettingsPanel } from "./lading-settings-panel";
import { LadingTestsPanel } from "./lading-tests-panel";
import { lineOf, type Problem } from "./studio-model";

const NOTES_LIMIT = 2_000;

/** The panels under the editor: problems, tests, runs, releases, settings and notes. */
export function LadingPanels({
  project,
  version,
  versions,
  check,
  problems,
  working,
  context,
  notes,
  editable,
  dirty,
  onNotesChange,
  onOpen,
  onValues,
  onCoverage,
}: {
  project: LadingProject;
  version: LadingVersion;
  versions: readonly LadingVersion[];
  check: LadingCheck;
  problems: readonly Problem[];
  working: LadingProjectInput;
  context: LadingContext;
  notes: string;
  editable: boolean;
  dirty: boolean;
  onNotesChange: (notes: string) => void;
  onOpen: (location: LadingLocation) => void;
  onValues: (values: readonly LadingRunValue[]) => void;
  onCoverage: (coverage: LadingTestReport["coverage"]) => void;
}) {
  const t = useT();
  return (
    <Tabs defaultValue="problems" className="flex h-full min-h-0 flex-col">
      <TabsList variant="underline" className="shrink-0 border-b px-2">
        <TabsTab value="problems">
          {t("{0, plural, =0 {Problems} other {Problems (#)}}", problems.length)}
        </TabsTab>
        <TabsTab value="tests">{t("Tests")}</TabsTab>
        <TabsTab value="run">{t("Run")}</TabsTab>
        <TabsTab value="release">{t("Release")}</TabsTab>
        <TabsTab value="settings">{t("Settings")}</TabsTab>
        <TabsTab value="notes">{t("Notes")}</TabsTab>
      </TabsList>
      <TabsPanel value="problems" className="min-h-0 flex-1 overflow-y-auto">
        <ProblemsList problems={problems} working={working} onOpen={onOpen} />
      </TabsPanel>
      <TabsPanel value="tests" className="min-h-0 flex-1 overflow-y-auto">
        <LadingTestsPanel
          working={working}
          context={context}
          onOpen={onOpen}
          onCoverage={onCoverage}
        />
      </TabsPanel>
      <TabsPanel value="run" className="min-h-0 flex-1 overflow-y-auto">
        <LadingRunPanel
          project={project}
          version={version}
          check={check}
          working={working}
          context={context}
          dirty={dirty}
          onValues={onValues}
        />
      </TabsPanel>
      <TabsPanel value="release" className="min-h-0 flex-1 overflow-y-auto">
        <LadingReleasePanel
          project={project}
          version={version}
          versions={versions}
          check={check}
          dirty={dirty}
        />
      </TabsPanel>
      <TabsPanel value="settings" className="min-h-0 flex-1 overflow-y-auto">
        <LadingSettingsPanel project={project} version={version} check={check} />
      </TabsPanel>
      <TabsPanel value="notes" className="min-h-0 flex-1 overflow-y-auto p-3">
        <Textarea
          aria-label={t("Version notes")}
          placeholder={
            editable ? t("What changed in this version and why") : t("No notes for this version")
          }
          value={notes}
          readOnly={!editable}
          maxLength={NOTES_LIMIT}
          minRows={4}
          onChange={(event) => onNotesChange(event.target.value)}
        />
        <p className="text-muted-foreground mt-1 text-right text-xs tabular-nums">
          {t("{0} of {1}", notes.length, NOTES_LIMIT)}
        </p>
      </TabsPanel>
    </Tabs>
  );
}

function ProblemsList({
  problems,
  working,
  onOpen,
}: {
  problems: readonly Problem[];
  working: LadingProjectInput;
  onOpen: (location: LadingLocation) => void;
}) {
  const t = useT();
  if (problems.length === 0) {
    return <p className="text-muted-foreground p-3 text-sm">{t("The project checks clean.")}</p>;
  }
  const severityLabel = {
    error: t("Error"),
    warning: t("Warning"),
    info: t("Note"),
  } as const;
  return (
    <ul className="flex flex-col">
      {problems.map((problem, index) => {
        const source = working.files.find((file) => file.path === problem.path)?.source ?? "";
        return (
          <li
            // oxlint-disable-next-line react/no-array-index-key -- two findings can share a file, code and range
            key={`${problem.path}:${problem.range.from}:${problem.code}:${index}`}
          >
            <button
              type="button"
              className="ui-focus-ring hover:bg-surface-hover flex w-full items-baseline gap-3 px-3 py-1.5 text-left text-sm"
              onClick={() => onOpen({ path: problem.path, range: problem.range })}
            >
              <span
                className={cn(
                  "w-14 shrink-0 text-xs",
                  problem.severity === "error"
                    ? "text-danger"
                    : problem.severity === "warning"
                      ? "text-warning"
                      : "text-muted-foreground",
                )}
              >
                {severityLabel[problem.severity]}
              </span>
              <span className="min-w-0 flex-1">{translateMessage(problem.message, t)}</span>
              <span className="text-muted-foreground shrink-0 font-mono text-xs">
                {t("{0}:{1}", problem.path, lineOf(source, problem.range.from))}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
