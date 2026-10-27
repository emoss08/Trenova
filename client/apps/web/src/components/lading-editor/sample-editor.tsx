import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { RangeSetBuilder, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";
import { x12Spans } from "@trenova/shared/lib/highlight/x12";
import { cn } from "@trenova/shared/lib/utils";
import CodeMirror from "@uiw/react-codemirror";
import { darkTheme, languageServiceTheme, lightTheme } from "../formula-editor/editor-theme";
import { syntaxHighlightTheme } from "../formula-editor/tariff-highlight";

const marks = new Map<string, Decoration>();

function markFor(role: string): Decoration {
  let mark = marks.get(role);
  if (!mark) {
    mark = Decoration.mark({ class: `cm-syntax-${role}` });
    marks.set(role, mark);
  }
  return mark;
}

/** A sample's X12 colours as decorations; none for a sample that is not X12. */
export function x12Decorations(text: string): DecorationSet {
  const builder = new RangeSetBuilder<Decoration>();
  for (const span of x12Spans(text)) {
    if (span.role && span.to > span.from) builder.add(span.from, span.to, markFor(span.role));
  }
  return builder.finish();
}

const x12Highlight = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = x12Decorations(view.state.doc.toString());
    }

    update(update: ViewUpdate) {
      if (update.docChanged) this.decorations = x12Decorations(update.state.doc.toString());
    }
  },
  { decorations: (value) => value.decorations },
);

const EXTENSIONS: Extension[] = [
  x12Highlight,
  syntaxHighlightTheme,
  EditorView.lineWrapping,
  languageServiceTheme,
];

/**
 * One sample of a Lading project: X12 coloured with the interchange's own separators (segment
 * IDs, the envelope apart, separators and numbers); CSV, JSON and other text shown plain.
 */
export function SampleEditor({
  path,
  value,
  onChange,
  readOnly = false,
  className,
}: {
  path: string;
  value: string;
  onChange: (content: string) => void;
  readOnly?: boolean;
  className?: string;
}) {
  const resolvedTheme = useResolvedTheme();
  return (
    <div
      data-sample-path={path}
      className={cn("isolate h-full min-h-0 overflow-hidden", className)}
    >
      <CodeMirror
        value={value}
        onChange={onChange}
        readOnly={readOnly}
        editable={!readOnly}
        aria-label={path}
        height="100%"
        className="h-full"
        extensions={EXTENSIONS}
        basicSetup={{
          lineNumbers: true,
          foldGutter: false,
          highlightActiveLine: true,
          highlightActiveLineGutter: true,
          highlightSelectionMatches: true,
          searchKeymap: true,
          autocompletion: false,
          syntaxHighlighting: false,
        }}
        theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
      />
    </div>
  );
}
