import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { downloadTextFile } from "@/lib/utils";
import type { EDIX12Inspection } from "@/types/edi";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { CopyIcon, DownloadIcon } from "lucide-react";
import { useMemo, useState } from "react";
import type { useEditorTheme } from "../../components/designer-shared";
import type { InspectorContext } from "../inspector-context";
import { x12LineDecorations, x12StreamLanguage, x12ViewerTheme } from "../utils/x12-codemirror";

export default function RawViewTab({
  context,
  inspection,
  selectedSegmentIndex,
  editorTheme,
}: {
  context: InspectorContext;
  inspection: EDIX12Inspection;
  selectedSegmentIndex: number;
  editorTheme: ReturnType<typeof useEditorTheme>;
}) {
  const { copy } = useCopyToClipboard();
  const [wrap, setWrap] = useState(true);
  const extensions = useMemo(
    () => [
      x12StreamLanguage(inspection.separators),
      x12ViewerTheme,
      x12LineDecorations({
        inspection,
        selectedSegmentIndex,
        diagnostics: inspection.diagnostics,
      }),
      ...(wrap ? [EditorView.lineWrapping] : []),
    ],
    [inspection, selectedSegmentIndex, wrap],
  );

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => void copy(context.rawX12, { withToast: true })}
          >
            <CopyIcon className="size-4" />
            Copy raw
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => downloadTextFile(context.rawFilename, context.rawX12, "text/plain")}
          >
            <DownloadIcon className="size-4" />
            Download
          </Button>
        </div>
        <label className="flex items-center gap-2 text-xs text-muted-foreground">
          Wrap
          <Switch checked={wrap} onCheckedChange={setWrap} />
        </label>
      </div>
      <CodeMirror
        value={inspection.rawX12}
        editable={false}
        basicSetup={{ lineNumbers: true, foldGutter: false }}
        extensions={extensions}
        theme={editorTheme}
        className="min-h-0 overflow-auto rounded-md border text-xs"
      />
    </div>
  );
}
