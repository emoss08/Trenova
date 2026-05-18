import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { EDITemplateScriptLibrary } from "@/types/edi";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { PlusIcon, SaveIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import {
  getEDIScriptPresetsByCategory,
  insertScriptPresetCode,
  type EDIScriptPreset,
} from "../../edi-script-presets";
import { InputBlock, ScriptPresetPicker } from "../components/designer-shared";

export function ScriptLibraryEditor({
  libraries,
  isEditable,
  onChange,
  onSave,
  isSaving,
}: {
  libraries: EDITemplateScriptLibrary[];
  isEditable: boolean;
  onChange: (libraries: EDITemplateScriptLibrary[]) => void;
  onSave: () => void;
  isSaving: boolean;
}) {
  const { theme } = useTheme();
  const editorTheme = theme === "dark" ? darkTheme : lightTheme;
  const [selectedId, setSelectedId] = useState("");
  const selected = libraries.find((library) => library.id === selectedId) ?? libraries[0];

  useEffect(() => {
    if (!selectedId && libraries[0]) setSelectedId(libraries[0].id);
  }, [libraries, selectedId]);

  const updateSelected = (patch: Partial<EDITemplateScriptLibrary>) => {
    if (!selected) return;
    onChange(
      libraries.map((library) => (library.id === selected.id ? { ...library, ...patch } : library)),
    );
  };
  const applyPreset = (preset: EDIScriptPreset) => {
    if (!selected) return;
    updateSelected({ script: insertScriptPresetCode(selected.script, preset) });
  };

  return (
    <div className="grid h-full grid-cols-[260px_minmax(0,1fr)]">
      <div className="min-h-0 overflow-auto border-r">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-xs font-semibold">Libraries</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!isEditable}
            onClick={() => {
              const id = `draft-${Date.now()}`;
              onChange([
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
      </div>
      <div className="grid min-h-0 grid-rows-[auto_auto_minmax(0,1fr)]">
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
                selected && onChange(libraries.filter((library) => library.id !== selected.id))
              }
            >
              <Trash2Icon className="size-4" />
              Remove
            </Button>
            <Button type="button" disabled={!isEditable} isLoading={isSaving} onClick={onSave}>
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
