import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { downloadTextFile } from "@/lib/utils";
import type { EDIMessage } from "@/types/edi";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { CopyIcon, DownloadIcon } from "lucide-react";
import { useMemo, useState } from "react";
import type { useEditorTheme } from "../../components/designer-shared";
import { buildX12Filename } from "../../utils/edi-message-utils";
import { x12LineDecorations, x12StreamLanguage, x12ViewerTheme } from "../utils/x12-codemirror";
import { x12DisplayText, type ParsedX12Document } from "../utils/x12-parser";

export default function RawViewTab({
  message,
  document,
  selectedSegmentIndex,
  editorTheme,
}: {
  message: EDIMessage;
  document: ParsedX12Document;
  selectedSegmentIndex: number;
  editorTheme: ReturnType<typeof useEditorTheme>;
}) {
  const { copy } = useCopyToClipboard();
  const [wrap, setWrap] = useState(true);
  const displayRawX12 = useMemo(() => x12DisplayText(document), [document]);
  const extensions = useMemo(
    () => [
      x12StreamLanguage(document.delimiters),
      x12ViewerTheme,
      x12LineDecorations({
        document,
        selectedSegmentIndex,
        diagnostics: message.validationErrors,
      }),
      ...(wrap ? [EditorView.lineWrapping] : []),
    ],
    [document, message.validationErrors, selectedSegmentIndex, wrap],
  );

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => void copy(message.rawX12, { withToast: true })}
          >
            <CopyIcon className="size-4" />
            Copy raw
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              downloadTextFile(buildX12Filename(message), message.rawX12, "text/plain")
            }
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
        value={displayRawX12}
        editable={false}
        basicSetup={{ lineNumbers: true, foldGutter: false }}
        extensions={extensions}
        theme={editorTheme}
        className="min-h-0 overflow-auto rounded-md border text-xs"
      />
    </div>
  );
}
