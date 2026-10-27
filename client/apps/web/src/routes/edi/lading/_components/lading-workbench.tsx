import { useT } from "@trenova/shared/i18n/use-t";
import { LadingEditor } from "@/components/lading-editor/lading-editor";
import { SampleEditor } from "@/components/lading-editor/sample-editor";
import { checkProject, unlessRefused } from "@/components/lading-editor/lading-service";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  ladingSample,
  type LadingContext,
  type LadingFile,
  type LadingLocation,
  type LadingRange,
  type LadingRunValue,
  type LadingSample,
  type LadingTestReport,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import type {
  LadingProject,
  LadingVersion,
  LadingVersionDetail,
} from "@trenova/shared/types/lading";
import { useMemo, useState } from "react";
import type { NewEntry } from "./lading-add-entry-dialog";
import { LadingFileTree } from "./lading-file-tree";
import { LadingPanels, type PanelTab } from "./lading-panels";
import { LadingVersionToolbar } from "./lading-version-toolbar";
import {
  applyFileEdits,
  isDirty,
  isEditable,
  ladingName,
  PARTNER_FILE,
  projectProblems,
  renameEntry,
  starterSamplePath,
} from "./studio-model";

type Reveal = { path: string; range: LadingRange };

function firstPath(files: readonly LadingFile[], samples: readonly LadingSample[]): string {
  return (
    files.find((file) => file.path === PARTNER_FILE)?.path ??
    files[0]?.path ??
    samples[0]?.path ??
    ""
  );
}

/**
 * One version open in the studio. It holds the working copy of the version's files and samples,
 * checks the whole project in the browser on every change, and saves the copy back as one draft
 * write; every version but a draft opens read-only.
 */
export function LadingWorkbench({
  project,
  versions,
  detail,
  context,
  partnerName,
  onPickVersion,
}: {
  project: LadingProject;
  versions: readonly LadingVersion[];
  detail: LadingVersionDetail;
  context: LadingContext;
  partnerName: string;
  onPickVersion: (id: string) => void;
}) {
  const t = useT();
  const edi = useEdi();
  const version = detail.version;
  const editable = isEditable(version);

  const [files, setFiles] = useState<LadingFile[]>(() => version.files);
  const [samples, setSamples] = useState<LadingSample[]>(() => detail.samples);
  const [notes, setNotes] = useState(() => version.notes ?? "");
  const [openPath, setOpenPath] = useState(() => firstPath(version.files, detail.samples));
  const [reveal, setReveal] = useState<Reveal | null>(null);
  const [values, setValues] = useState<readonly LadingRunValue[]>([]);
  const [coverage, setCoverage] = useState<LadingTestReport["coverage"]>([]);
  const [panelTab, setPanelTab] = useState<PanelTab>("problems");

  const working = useMemo(() => ({ files, samples }), [files, samples]);
  const check = checkProject(edi, working, context);
  const problems = projectProblems(check);
  const dirty =
    isDirty({ files: version.files, samples: detail.samples }, working) ||
    notes !== (version.notes ?? "");

  const openFile = openPath.endsWith(".lading")
    ? files.find((file) => file.path === openPath)
    : undefined;
  const openSample = openFile ? undefined : samples.find((sample) => sample.path === openPath);

  const open = (location: LadingLocation) => {
    setOpenPath(location.path);
    setReveal({ path: location.path, range: location.range });
  };

  const discard = () => {
    setFiles(version.files);
    setSamples(detail.samples);
    setNotes(version.notes ?? "");
    if (!version.files.some((file) => file.path === openPath)) {
      setOpenPath(firstPath(version.files, detail.samples));
    }
  };

  const addEntries = (entries: NewEntry[]) => {
    const nextFiles: LadingFile[] = [
      ...files,
      ...entries
        .filter((entry) => entry.kind === "file")
        .map((entry) => ({ path: entry.path, source: entry.content })),
    ];
    const nextSamples: LadingSample[] = [
      ...samples,
      ...entries
        .filter((entry) => entry.kind === "sample")
        .map((entry) => ({ path: entry.path, content: entry.content })),
    ];
    for (const entry of entries) {
      if (entry.kind !== "file" || !entry.withSample) continue;
      const starter = unlessRefused(
        () =>
          ladingSample(edi, {
            project: { files: nextFiles, samples: nextSamples },
            context,
            map: ladingName(entry.path),
          }),
        null,
      );
      if (!starter) continue;
      const taken = [...nextFiles, ...nextSamples].map((item) => item.path);
      nextSamples.push({
        path: starterSamplePath(entry.path, starter.format, taken),
        content: starter.content,
      });
    }
    setFiles(nextFiles);
    setSamples(nextSamples);
    if (entries[0]) setOpenPath(entries[0].path);
  };

  const moveEntry = (from: string, to: string) => {
    const moved = renameEntry({ files, samples }, from, to);
    setFiles(moved.files);
    setSamples(moved.samples);
    if (openPath === from) setOpenPath(to);
    setReveal((current) => (current?.path === from ? { ...current, path: to } : current));
  };

  const removeEntry = (path: string) => {
    setFiles((current) => current.filter((file) => file.path !== path));
    setSamples((current) => current.filter((sample) => sample.path !== path));
    if (path === openPath) {
      const remaining = files.filter((file) => file.path !== path);
      setOpenPath(
        firstPath(
          remaining,
          samples.filter((sample) => sample.path !== path),
        ),
      );
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col gap-3">
      <LadingVersionToolbar
        project={project}
        versions={versions}
        version={version}
        check={check}
        working={{ files, samples, notes }}
        dirty={dirty}
        onDiscard={discard}
        onPickVersion={onPickVersion}
      />
      <ResizablePanelGroup
        orientation="horizontal"
        persistId="lading-studio:files"
        className="min-h-0 flex-1 rounded-md border"
      >
        <ResizablePanel id="files" defaultSize="20%" minSize="12rem" maxSize="28rem">
          <LadingFileTree
            files={files}
            samples={samples}
            check={check}
            openPath={openPath}
            editable={editable}
            partnerLabel={check.partner?.label ?? partnerName}
            zone={context.org.zone}
            onOpen={(path) => setOpenPath(path)}
            onAdd={addEntries}
            onRename={moveEntry}
            onRemove={removeEntry}
          />
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel id="work" defaultSize="80%" minSize="40%">
          <ResizablePanelGroup orientation="vertical" persistId="lading-studio:panels">
            <ResizablePanel id="editor" defaultSize="62%" minSize="30%">
              {openFile ? (
                <LadingEditor
                  key={openFile.path}
                  path={openFile.path}
                  value={openFile.source}
                  onChange={(source) =>
                    setFiles((current) =>
                      current.map((file) =>
                        file.path === openFile.path ? { ...file, source } : file,
                      ),
                    )
                  }
                  project={working}
                  context={context}
                  values={values}
                  onOpenFile={open}
                  onEditFiles={(edits) => setFiles((current) => applyFileEdits(current, edits))}
                  onFormat={(reason) => {
                    if (reason)
                      toast.error(t("The file was not formatted"), { description: reason });
                  }}
                  onShowProblems={() => setPanelTab("problems")}
                  reveal={reveal?.path === openFile.path ? reveal.range : null}
                  uncovered={coverage.find((entry) => entry.path === openFile.path)?.uncovered}
                  readOnly={!editable}
                />
              ) : openSample ? (
                <SampleEditor
                  key={openSample.path}
                  path={openSample.path}
                  value={openSample.content}
                  readOnly={!editable}
                  onChange={(content) =>
                    setSamples((current) =>
                      current.map((sample) =>
                        sample.path === openSample.path ? { ...sample, content } : sample,
                      ),
                    )
                  }
                />
              ) : (
                <p className="text-muted-foreground p-4 text-sm">
                  {t("Pick a file or a sample to open it")}
                </p>
              )}
            </ResizablePanel>
            <ResizableHandle />
            <ResizablePanel id="panels" defaultSize="38%" minSize="8rem" maxSize="70%">
              <LadingPanels
                project={project}
                version={version}
                versions={versions}
                check={check}
                problems={problems}
                working={working}
                context={context}
                notes={notes}
                editable={editable}
                dirty={dirty}
                onNotesChange={setNotes}
                onOpen={open}
                onValues={setValues}
                onCoverage={setCoverage}
                tab={panelTab}
                onTabChange={setPanelTab}
              />
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
      <UnsavedChangesGuard when={dirty} />
    </div>
  );
}
