import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import type { EDITemplateScriptLibrary } from "@trenova/shared/types/edi";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { PlusIcon, SaveIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import {
  getEDIScriptPresetsByCategory,
  insertScriptPresetCode,
  type EDIScriptPreset,
} from "../../edi-script-presets";
import { InputBlock, ScriptPresetPicker } from "../components/designer-shared";
import { useSaveEDITemplateScriptsMutation } from "../hooks/use-edi-template-mutations";
import {
  useCurrentTemplateInvalidation,
  useSelectedTemplateDesignerData,
  useSelectedTemplateDesignerIds,
} from "@/hooks/use-template-designer-state";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import { isTemplateVersionEditable } from "../utils/edi-designer-utils";

export function ScriptLibraryEditor() {
  const { theme } = useTheme();
  const editorTheme = theme === "dark" ? darkTheme : lightTheme;
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const libraries = useTemplateDesignerStore((state) => state.scriptDraft);
  const replaceScripts = useTemplateDesignerStore((state) => state.replaceScripts);
  const clearScriptsDirty = useTemplateDesignerStore((state) => state.clearScriptsDirty);
  const invalidateTemplateQueries = useCurrentTemplateInvalidation();
  const isEditable = isTemplateVersionEditable(selectedVersion);
  const [selectedId, setSelectedId] = useState("");
  const selected = libraries.find((library) => library.id === selectedId) ?? libraries[0];

  const saveScriptsMutation = useSaveEDITemplateScriptsMutation({
    onSuccess: async () => {
      toast.success("Script libraries saved");
      clearScriptsDirty();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save script libraries"),
  });

  useEffect(() => {
    if (!selectedId && libraries[0]) setSelectedId(libraries[0].id);
  }, [libraries, selectedId]);

  const updateSelected = (patch: Partial<EDITemplateScriptLibrary>) => {
    if (!selected) return;
    replaceScripts(
      libraries.map((library) => (library.id === selected.id ? { ...library, ...patch } : library)),
    );
  };
  const applyPreset = (preset: EDIScriptPreset) => {
    if (!selected) return;
    updateSelected({ script: insertScriptPresetCode(selected.script, preset) });
  };

  return (
    <div className="grid h-full min-h-0 grid-cols-[260px_minmax(0,1fr)] overflow-hidden">
      <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden border-r">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-xs font-semibold">Libraries</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!isEditable}
            onClick={() => {
              const id = `draft-${Date.now()}`;
              replaceScripts([
                ...libraries,
                {
                  id,
                  templateVersionId: "",
                  name: "new_library",
                  description: "",
                  language: "Starlark",
                  script: "def normalize(value):\n    return value\n",
                  status: "Draft",
                  version: 0,
                  functionNames: ["normalize"],
                },
              ]);
              setSelectedId(id);
            }}
          >
            <PlusIcon className="size-4" />
          </Button>
        </div>
        <ScrollArea className="min-h-0" viewportClassName="min-h-0">
          {libraries.map((library) => (
            <button
              key={library.id}
              type="button"
              onClick={() => setSelectedId(library.id)}
              className={cn(
                "block w-full border-b px-3 py-2 text-left hover:bg-muted",
                selected?.id === library.id && "bg-muted",
              )}
            >
              <div className="truncate text-sm font-medium">{library.name}</div>
              <div className="truncate text-xs text-muted-foreground">
                {library.functionNames.length > 0
                  ? library.functionNames.join(", ")
                  : "No functions discovered"}
              </div>
            </button>
          ))}
        </ScrollArea>
      </div>
      <div className="grid min-h-0 grid-rows-[auto_auto_minmax(0,1fr)] overflow-hidden">
        <div className="flex items-end justify-between gap-3 border-b p-3">
          <div className="grid flex-1 grid-cols-2 gap-2">
            <InputBlock
              label="Name"
              value={selected?.name ?? ""}
              disabled={!isEditable || !selected}
              onChange={(name) => updateSelected({ name })}
            />
            <InputBlock
              label="Description"
              value={selected?.description ?? ""}
              disabled={!isEditable || !selected}
              onChange={(description) => updateSelected({ description })}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={!isEditable || !selected}
              onClick={() =>
                selected &&
                replaceScripts(libraries.filter((library) => library.id !== selected.id))
              }
            >
              <Trash2Icon className="size-4" />
              Remove
            </Button>
            <Button
              type="button"
              disabled={!isEditable}
              isLoading={saveScriptsMutation.isPending}
              onClick={() =>
                saveScriptsMutation.mutate({
                  templateId: selectedTemplateId,
                  versionId: selectedVersionId,
                  request: {
                    scriptLibraries: libraries,
                    version: selectedVersion?.version,
                  },
                })
              }
            >
              <SaveIcon className="size-4" />
              Save Scripts
            </Button>
          </div>
        </div>
        <div className="border-b p-3">
          <ScriptPresetPicker
            title="Script Presets"
            presets={getEDIScriptPresetsByCategory("scriptLibrary")}
            disabled={!isEditable || !selected}
            onApply={applyPreset}
          />
        </div>
        {selected ? (
          <div className="min-h-0 overflow-hidden">
            <CodeMirror
              value={selected.script}
              editable={isEditable}
              height="100%"
              extensions={[EditorView.lineWrapping, json()]}
              theme={editorTheme}
              basicSetup={{ lineNumbers: true, foldGutter: true, autocompletion: true }}
              onChange={(script) => updateSelected({ script })}
            />
          </div>
        ) : (
          <div className="p-4 text-sm text-muted-foreground">No script libraries.</div>
        )}
      </div>
    </div>
  );
}

export default ScriptLibraryEditor;
