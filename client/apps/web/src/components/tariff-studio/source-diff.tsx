import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";

/**
 * Line diff of two versions of a Tariff source, themed with the rest of the editor. A tariff is
 * a file, so changes read by line, with the words that changed inside a line marked.
 */
export function SourceDiff({
  before,
  after,
  beforeTitle,
  afterTitle,
}: {
  before: string;
  after: string;
  beforeTitle?: string;
  afterTitle?: string;
}) {
  const resolvedTheme = useResolvedTheme();

  return (
    <div className="overflow-hidden rounded-md border font-mono text-xs">
      <ReactDiffViewer
        oldValue={before}
        newValue={after}
        splitView={false}
        compareMethod={DiffMethod.WORDS_WITH_SPACE}
        useDarkTheme={resolvedTheme === "dark"}
        leftTitle={beforeTitle}
        rightTitle={afterTitle}
      />
    </div>
  );
}
