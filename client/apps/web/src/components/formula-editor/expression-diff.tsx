import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";

/** Word-level diff of two expressions, themed with the rest of the editor. */
export function ExpressionDiff({ before, after }: { before: string; after: string }) {
  const resolvedTheme = useResolvedTheme();

  return (
    <div className="overflow-hidden rounded-md border font-mono text-xs">
      <ReactDiffViewer
        oldValue={before}
        newValue={after}
        splitView={false}
        compareMethod={DiffMethod.WORDS}
        useDarkTheme={resolvedTheme === "dark"}
        hideLineNumbers
      />
    </div>
  );
}
