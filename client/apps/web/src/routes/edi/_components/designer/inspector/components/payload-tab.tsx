import { Button } from "@trenova/shared/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { downloadJsonFile } from "@trenova/shared/lib/utils";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { CopyIcon, DownloadIcon } from "lucide-react";
import { useMemo } from "react";
import type { useEditorTheme } from "../../components/designer-shared";
import type { InspectorContext } from "../inspector-context";

export default function PayloadTab({
  context,
  editorTheme,
}: {
  context: InspectorContext;
  editorTheme: ReturnType<typeof useEditorTheme>;
}) {
  const { copy } = useCopyToClipboard();
  const payloadJson = useMemo(
    () => JSON.stringify(context.payload?.value ?? {}, null, 2),
    [context.payload],
  );

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="mb-2 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(payloadJson, { withToast: true })}
        >
          <CopyIcon className="size-4" />
          Copy
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            downloadJsonFile(
              context.payload?.filename ?? "edi-payload.json",
              context.payload?.value,
            )
          }
        >
          <DownloadIcon className="size-4" />
          Download
        </Button>
      </div>
      <CodeMirror
        value={payloadJson}
        editable={false}
        basicSetup={{ lineNumbers: true, foldGutter: true }}
        extensions={[json(), EditorView.lineWrapping]}
        theme={editorTheme}
        className="min-h-0 overflow-auto rounded-md border text-xs"
      />
    </div>
  );
}
