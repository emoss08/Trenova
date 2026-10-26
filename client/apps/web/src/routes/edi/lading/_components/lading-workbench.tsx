import { useT } from "@trenova/shared/i18n/use-t";
import { LadingEditor } from "@/components/lading-editor/lading-editor";
import { checkProject } from "@/components/lading-editor/lading-service";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { toast } from "@trenova/shared/components/ui/toast";
import type {
  LadingContext,
  LadingFile,
  LadingLocation,
  LadingRange,
  LadingRunValue,
  LadingSample,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import type {
  LadingProject,
  LadingVersion,
  LadingVersionDetail,
} from "@trenova/shared/types/lading";
import { useMemo, useState } from "react";
import { LadingFileTree } from "./lading-file-tree";
import { LadingPanels } from "./lading-panels";
import { LadingVersionToolbar } from "./lading-version-toolbar";
import { isDirty, isEditable, problemCounts, projectProblems } from "./studio-model";

type Reveal = { path: string; range: LadingRange };

function firstPath(files: readonly LadingFile[], samples: readonly LadingSample[]): string {
  return (
    files.find((file) => file.path === "partner.lading")?.path ??
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

  const working = useMemo(() => ({ files, samples }), [files, samples]);
  const check = checkProject(edi, working, context);
  const problems = projectProblems(check);
  const counts = problemCounts(check);
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

  const addEntry = (kind: "file" | "sample", path: string, content: string) => {
    if (kind === "file") {
      setFiles((current) => [...current, { path, source: content }]);
    } else {
      setSamples((current) => [...current, { path, content }]);
    }
    setOpenPath(path);
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
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1 rounded-md border">
        <ResizablePanel defaultSize="20%" minSize="14%">
          <LadingFileTree
            files={files}
            samples={samples}
            counts={counts}
            openPath={openPath}
            editable={editable}
            partnerName={check.partner?.name ?? partnerName}
            onOpen={(path) => setOpenPath(path)}
            onAdd={addEntry}
            onRemove={removeEntry}
          />
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel defaultSize="80%" minSize="40%">
          <ResizablePanelGroup orientation="vertical">
            <ResizablePanel defaultSize="62%" minSize="25%">
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
                  onFormat={(reason) => {
                    if (reason)
                      toast.error(t("The file was not formatted"), { description: reason });
                  }}
                  reveal={reveal?.path === openFile.path ? reveal.range : null}
                  readOnly={!editable}
                />
              ) : openSample ? (
                <div className="h-full overflow-auto p-3">
                  <Textarea
                    key={openSample.path}
                    bare
                    aria-label={openSample.path}
                    className="min-h-full w-full resize-none font-mono text-xs"
                    value={openSample.content}
                    readOnly={!editable}
                    spellCheck={false}
                    onChange={(event) => {
                      const content = event.target.value;
                      setSamples((current) =>
                        current.map((sample) =>
                          sample.path === openSample.path ? { ...sample, content } : sample,
                        ),
                      );
                    }}
                  />
                </div>
              ) : (
                <p className="text-muted-foreground p-4 text-sm">
                  {t("Pick a file or a sample to open it")}
                </p>
              )}
            </ResizablePanel>
            <ResizableHandle />
            <ResizablePanel defaultSize="38%" minSize="15%">
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
              />
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
      <UnsavedChangesGuard when={dirty} />
    </div>
  );
}
