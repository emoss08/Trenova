import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
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
import { Activity, useId, type ReactNode } from "react";
import { LadingProblemsPanel } from "./lading-problems-panel";
import { LadingReleasePanel } from "./lading-release-panel";
import { LadingRunPanel } from "./lading-run-panel";
import { LadingSettingsPanel } from "./lading-settings-panel";
import { LadingTestsPanel } from "./lading-tests-panel";
import type { Problem } from "./studio-model";

const NOTES_LIMIT = 2_000;
const NOTES_NEAR_LIMIT = 0.9;

/** The tabs under the editor; the workbench holds which is open, so the editor can open Problems. */
export type PanelTab = "problems" | "tests" | "run" | "release" | "settings" | "notes";

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
  tab,
  onTabChange,
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
  tab: PanelTab;
  onTabChange: (tab: PanelTab) => void;
}) {
  const t = useT();
  return (
    <Tabs
      value={tab}
      onValueChange={(next) => onTabChange(next as PanelTab)}
      className="flex h-full min-h-0 flex-col gap-0"
    >
      <TabsList variant="underline" className="w-full shrink-0 justify-start border-b px-2">
        <TabsTab value="problems" className="grow-0">
          {t("Problems")}
          <ProblemsBadge problems={problems} />
        </TabsTab>
        <TabsTab value="tests" className="grow-0">
          {t("Tests")}
        </TabsTab>
        <TabsTab value="run" className="grow-0">
          {t("Run")}
        </TabsTab>
        <TabsTab value="release" className="grow-0">
          {t("Release")}
        </TabsTab>
        <TabsTab value="settings" className="grow-0">
          {t("Settings")}
        </TabsTab>
        <TabsTab value="notes" className="grow-0">
          {t("Notes")}
        </TabsTab>
      </TabsList>
      <ScrollingPanel value="problems" active={tab === "problems"}>
        <LadingProblemsPanel problems={problems} working={working} onOpen={onOpen} />
      </ScrollingPanel>
      <ScrollingPanel value="tests" active={tab === "tests"}>
        <LadingTestsPanel
          working={working}
          context={context}
          onOpen={onOpen}
          onCoverage={onCoverage}
        />
      </ScrollingPanel>
      <ScrollingPanel value="run" active={tab === "run"}>
        <LadingRunPanel
          project={project}
          version={version}
          check={check}
          working={working}
          context={context}
          dirty={dirty}
          onValues={onValues}
        />
      </ScrollingPanel>
      <ScrollingPanel value="release" active={tab === "release"}>
        <LadingReleasePanel
          project={project}
          version={version}
          versions={versions}
          check={check}
          dirty={dirty}
        />
      </ScrollingPanel>
      <ScrollingPanel value="settings" active={tab === "settings"}>
        <LadingSettingsPanel project={project} version={version} check={check} />
      </ScrollingPanel>
      <ScrollingPanel value="notes" active={tab === "notes"}>
        <VersionNotes notes={notes} editable={editable} onChange={onNotesChange} />
      </ScrollingPanel>
    </Tabs>
  );
}

/**
 * A tab's panel that fills the space under the tabs and scrolls its content with the fade. It
 * stays mounted while another tab is open, inside an `Activity` that hides it and pauses its
 * effects, so a test run, a map run or a half-written setting is still there on the way back.
 */
function ScrollingPanel({
  value,
  active,
  children,
}: {
  value: PanelTab;
  active: boolean;
  children: ReactNode;
}) {
  return (
    <TabsPanel value={value} keepMounted className="min-h-0">
      <Activity mode={active ? "visible" : "hidden"}>
        <ScrollArea className="h-full">{children}</ScrollArea>
      </Activity>
    </TabsPanel>
  );
}

/** The problem count on the tab, in the tone of the worst problem. */
function ProblemsBadge({ problems }: { problems: readonly Problem[] }) {
  if (problems.length === 0) return null;
  const tone = problems.some((problem) => problem.severity === "error")
    ? "danger"
    : problems.some((problem) => problem.severity === "warning")
      ? "warning"
      : "neutral";
  return (
    <Badge variant={tone} className="tabular-nums">
      {problems.length}
    </Badge>
  );
}

/** The version's notes: what changed and why, saved with the draft, with what is left to write. */
function VersionNotes({
  notes,
  editable,
  onChange,
}: {
  notes: string;
  editable: boolean;
  onChange: (notes: string) => void;
}) {
  const t = useT();
  const id = useId();
  const near = notes.length >= NOTES_LIMIT * NOTES_NEAR_LIMIT;
  return (
    <div className="flex max-w-3xl flex-col gap-1.5 p-3">
      <div className="flex items-baseline gap-3">
        <label htmlFor={id} className="flex-1 text-sm font-medium">
          {t("Version notes")}
        </label>
        <span
          className={cn(
            "text-xs tabular-nums",
            near ? "text-warning-foreground" : "text-muted-foreground",
          )}
        >
          {t("{0} / {1}", notes.length, NOTES_LIMIT)}
        </span>
      </div>
      <p className="text-muted-foreground text-xs">
        {editable
          ? t("Saved with the draft. Say what changed and why, for whoever certifies it.")
          : t("Notes are written while the version is a draft.")}
      </p>
      <Textarea
        id={id}
        placeholder={
          editable ? t("What changed in this version and why") : t("No notes for this version")
        }
        value={notes}
        readOnly={!editable}
        maxLength={NOTES_LIMIT}
        minRows={6}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}
