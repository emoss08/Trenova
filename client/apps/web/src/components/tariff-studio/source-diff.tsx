import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";

// The viewer sets its own `monospace` on the code; the override puts the code face and the
// person's ligature choice back.
const DIFF_STYLES = {
  contentText: {
    fontFamily: "var(--font-code)",
    fontVariantLigatures: "var(--code-ligatures)",
  },
};

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
    <div className="overflow-hidden rounded-md border font-code text-xs">
      <ReactDiffViewer
        oldValue={before}
        newValue={after}
        splitView={false}
        compareMethod={DiffMethod.WORDS_WITH_SPACE}
        useDarkTheme={resolvedTheme === "dark"}
        leftTitle={beforeTitle}
        rightTitle={afterTitle}
        styles={DIFF_STYLES}
      />
    </div>
  );
}
