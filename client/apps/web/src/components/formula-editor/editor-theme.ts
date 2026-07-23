import { EditorView } from "@codemirror/view";

export const lightTheme = EditorView.theme(
  {
    "&": {
      "--expr-keyword": "oklch(0.75 0.15 320)",
      "--expr-operator": "oklch(0.7 0 0)",
      "--expr-number": "oklch(0.75 0.12 150)",
      "--expr-string": "oklch(0.75 0.12 150)",
      "--expr-variable": "oklch(0.7 0.18 260)",
      "--expr-function": "oklch(0.7 0.12 200)",
      "--expr-comment": "oklch(0.5 0 0)",
      "--expr-punctuation": "oklch(0.6 0 0)",
    },
    "&.cm-editor": {
      backgroundColor: "var(--muted)",
      color: "var(--foreground)",
      borderRadius: "var(--radius)",
      borderStyle: "solid",
    },
    ".cm-theme[aria-invalid='true'] &.cm-editor": {
      borderColor: "var(--destructive)",
      backgroundColor:
        "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-content": {
      caretColor: "var(--primary)",
      fontFamily: "'Geist Mono', 'SF Mono', Monaco, 'Cascadia Code', monospace",
      fontSize: "13px",
      lineHeight: "1.6",
      padding: "12px 0",
    },
    ".cm-line": {
      padding: "0 12px",
    },
    ".cm-cursor, .cm-dropCursor": {
      borderLeftColor: "var(--foreground)",
      borderLeftWidth: "2px",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
      {
        backgroundColor:
          "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      },
    ".cm-activeLine": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--muted)",
      color: "var(--muted-foreground)",
      border: "none",
      borderRight: "1px solid var(--border)",
      fontFamily: "'Geist Mono', monospace",
      fontSize: "12px",
    },
    ".cm-theme[aria-invalid='true'] & .cm-gutters": {
      borderColor: "var(--destructive)",
      color: "var(--destructive)",
      backgroundColor:
        "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-theme[aria-invalid='true'] & .cm-placeholder": {
      color: "var(--destructive)",
    },
    ".cm-activeLineGutter": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
      color: "var(--foreground)",
    },
    ".cm-lineNumbers .cm-gutterElement": {
      padding: "0 8px 0 12px",
      minWidth: "32px",
    },
    "&.cm-focused": {
      outline: "none",
    },
    ".cm-matchingBracket": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      outline:
        "1px solid color-mix(in oklab, var(--muted-foreground) 60%, transparent)",
    },
    ".cm-placeholder": {
      color: "var(--muted-foreground)",
      fontStyle: "italic",
    },
    ".cm-tooltip": {
      backgroundColor: "var(--popover)",
      border: "1px solid var(--border)",
      borderRadius: "8px",
      boxShadow: "0 4px 12px var(--shadow-md)",
      overflow: "hidden",
    },
    ".cm-tooltip.cm-tooltip-autocomplete": {
      padding: "4px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul": {
      fontFamily: "'Geist Mono', monospace",
      fontSize: "13px",
      maxHeight: "300px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
      padding: "6px 10px",
      borderRadius: "4px",
      display: "flex",
      alignItems: "center",
      gap: "8px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
      color: "var(--foreground)",
    },
    ".cm-completionIcon": {
      fontSize: "14px",
      opacity: "0.8",
    },
    ".cm-completionLabel": {
      flex: "1",
    },
    ".cm-completionDetail": {
      color: "var(--muted-foreground)",
      fontSize: "11px",
      fontStyle: "normal",
      marginLeft: "auto",
      paddingLeft: "12px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail":
      {
        color: "var(--foreground)",
      },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-highlight)",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText":
      {
        color: "var(--color-highlight)",
      },
  },
  { dark: false },
);

export const darkTheme = EditorView.theme(
  {
    "&": {
      "--expr-keyword": "oklch(0.75 0.15 320)",
      "--expr-operator": "oklch(0.7 0 0)",
      "--expr-number": "oklch(0.75 0.12 150)",
      "--expr-string": "oklch(0.75 0.12 150)",
      "--expr-variable": "oklch(0.7 0.18 260)",
      "--expr-function": "oklch(0.7 0.12 200)",
      "--expr-comment": "oklch(0.5 0 0)",
      "--expr-punctuation": "oklch(0.6 0 0)",
    },
    "&.cm-editor": {
      backgroundColor: "var(--muted)",
      color: "var(--foreground)",
      borderRadius: "var(--radius)",
      borderStyle: "solid",
    },
    ".cm-theme[aria-invalid='true'] &.cm-editor": {
      borderColor: "var(--destructive)",
      backgroundColor:
        "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-content": {
      caretColor: "oklch(0.7 0.18 260)",
      fontFamily: "'Geist Mono', 'SF Mono', Monaco, 'Cascadia Code', monospace",
      fontSize: "13px",
      lineHeight: "1.6",
      padding: "12px 0",
    },
    ".cm-line": {
      padding: "0 12px",
    },
    ".cm-cursor, .cm-dropCursor": {
      borderLeftColor: "var(--foreground)",
      borderLeftWidth: "2px",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
      {
        backgroundColor:
          "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      },
    ".cm-activeLine": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--muted)",
      color: "var(--muted-foreground)",
      border: "none",
      borderRight: "1px solid var(--border)",
      fontFamily: "'Geist Mono', monospace",
      fontSize: "12px",
    },
    ".cm-theme[aria-invalid='true'] & .cm-gutters": {
      borderColor: "var(--destructive)",
      color: "var(--destructive)",
      backgroundColor:
        "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-theme[aria-invalid='true'] & .cm-placeholder": {
      color: "var(--destructive)",
    },
    ".cm-activeLineGutter": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
      color: "var(--foreground)",
    },
    ".cm-lineNumbers .cm-gutterElement": {
      padding: "0 8px 0 12px",
      minWidth: "32px",
    },
    "&.cm-focused": {
      outline: "none",
    },
    ".cm-matchingBracket": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      outline:
        "1px solid color-mix(in oklab, var(--muted-foreground) 60%, transparent)",
    },
    ".cm-placeholder": {
      color: "var(--muted-foreground)",
      fontStyle: "italic",
    },
    ".cm-tooltip": {
      backgroundColor: "var(--popover)",
      border: "1px solid var(--border)",
      borderRadius: "8px",
      boxShadow: "0 4px 12px var(--shadow-md)",
      overflow: "hidden",
    },
    ".cm-tooltip.cm-tooltip-autocomplete": {
      padding: "4px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul": {
      fontFamily: "'Geist Mono', monospace",
      fontSize: "13px",
      maxHeight: "300px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
      padding: "6px 10px",
      borderRadius: "4px",
      display: "flex",
      alignItems: "center",
      gap: "8px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
      backgroundColor:
        "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
      color: "var(--foreground)",
    },
    ".cm-completionIcon": {
      fontSize: "14px",
      opacity: "0.8",
    },
    ".cm-completionLabel": {
      flex: "1",
    },
    ".cm-completionDetail": {
      color: "var(--muted-foreground)",
      fontSize: "11px",
      fontStyle: "normal",
      marginLeft: "auto",
      paddingLeft: "12px",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail":
      {
        color: "var(--foreground)",
      },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-highlight)",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText":
      {
        color: "var(--color-highlight)",
      },
  },
  { dark: true },
);
