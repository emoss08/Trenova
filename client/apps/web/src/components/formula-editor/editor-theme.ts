import { EditorView } from "@codemirror/view";

export const lightTheme = EditorView.theme(
  {
    "&": {
      "--expr-keyword": "var(--accent-violet-on-subtle)",
      "--expr-operator": "var(--foreground-muted)",
      "--expr-number": "var(--success-foreground)",
      "--expr-string": "var(--success-foreground)",
      "--expr-variable": "var(--brand-subtle-foreground)",
      "--expr-function": "var(--accent-teal-on-subtle)",
      "--expr-comment": "var(--foreground-subtle)",
      "--expr-punctuation": "var(--foreground-subtle)",
      "--expr-unknown": "var(--warning-foreground)",
    },
    "&.cm-editor": {
      backgroundColor: "var(--field)",
      color: "var(--foreground)",
      borderRadius: "var(--radius)",
      borderStyle: "solid",
    },
    ".cm-theme[aria-invalid='true'] &.cm-editor": {
      borderColor: "var(--destructive)",
      backgroundColor: "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-expr-hover": {
      padding: "4px 8px",
      fontFamily: "var(--font-mono, ui-monospace, monospace)",
      fontSize: "11px",
      maxWidth: "360px",
      whiteSpace: "pre-wrap",
    },
    ".cm-expr-completion-info": {
      display: "flex",
      flexDirection: "column",
      gap: "4px",
      padding: "8px 10px",
      maxWidth: "280px",
      fontSize: "12px",
      fontFamily: "inherit",
    },
    ".cm-expr-completion-info-title": {
      fontFamily: "'Geist Mono', monospace",
      fontWeight: "600",
    },
    ".cm-expr-completion-info-example": {
      fontFamily: "'Geist Mono', monospace",
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 12%, transparent)",
      borderRadius: "4px",
      padding: "3px 6px",
      display: "block",
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
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
    },
    ".cm-activeLine": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
      backgroundColor: "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-theme[aria-invalid='true'] & .cm-placeholder": {
      color: "var(--destructive)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      outline: "1px solid color-mix(in oklab, var(--muted-foreground) 60%, transparent)",
    },
    ".cm-placeholder": {
      color: "var(--muted-foreground)",
      fontStyle: "italic",
    },
    ".cm-tooltip": {
      backgroundColor: "var(--popover)",
      border: "1px solid var(--border)",
      borderRadius: "8px",
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
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail": {
      color: "var(--foreground)",
    },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-highlight)",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText": {
      color: "var(--color-highlight)",
    },
  },
  { dark: false },
);

export const darkTheme = EditorView.theme(
  {
    "&": {
      "--expr-keyword": "var(--accent-violet-on-subtle)",
      "--expr-operator": "var(--foreground-muted)",
      "--expr-number": "var(--success-foreground)",
      "--expr-string": "var(--success-foreground)",
      "--expr-variable": "var(--brand-subtle-foreground)",
      "--expr-function": "var(--accent-teal-on-subtle)",
      "--expr-comment": "var(--foreground-subtle)",
      "--expr-punctuation": "var(--foreground-subtle)",
      "--expr-unknown": "var(--warning-foreground)",
    },
    "&.cm-editor": {
      backgroundColor: "var(--field)",
      color: "var(--foreground)",
      borderRadius: "var(--radius)",
      borderStyle: "solid",
    },
    ".cm-theme[aria-invalid='true'] &.cm-editor": {
      borderColor: "var(--destructive)",
      backgroundColor: "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-expr-hover": {
      padding: "4px 8px",
      fontFamily: "var(--font-mono, ui-monospace, monospace)",
      fontSize: "11px",
      maxWidth: "360px",
      whiteSpace: "pre-wrap",
    },
    ".cm-expr-completion-info": {
      display: "flex",
      flexDirection: "column",
      gap: "4px",
      padding: "8px 10px",
      maxWidth: "280px",
      fontSize: "12px",
      fontFamily: "inherit",
    },
    ".cm-expr-completion-info-title": {
      fontFamily: "'Geist Mono', monospace",
      fontWeight: "600",
    },
    ".cm-expr-completion-info-example": {
      fontFamily: "'Geist Mono', monospace",
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 12%, transparent)",
      borderRadius: "4px",
      padding: "3px 6px",
      display: "block",
    },
    ".cm-content": {
      caretColor: "var(--brand)",
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
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
    },
    ".cm-activeLine": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
      backgroundColor: "color-mix(in oklab, var(--destructive) 20%, transparent)",
    },
    ".cm-theme[aria-invalid='true'] & .cm-placeholder": {
      color: "var(--destructive)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 30%, transparent)",
      outline: "1px solid color-mix(in oklab, var(--muted-foreground) 60%, transparent)",
    },
    ".cm-placeholder": {
      color: "var(--muted-foreground)",
      fontStyle: "italic",
    },
    ".cm-tooltip": {
      backgroundColor: "var(--popover)",
      border: "1px solid var(--border)",
      borderRadius: "8px",
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
      backgroundColor: "color-mix(in oklab, var(--muted-foreground) 10%, transparent)",
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
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionDetail": {
      color: "var(--foreground)",
    },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-highlight)",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected] .cm-completionMatchedText": {
      color: "var(--color-highlight)",
    },
  },
  { dark: true },
);

/**
 * The language service's own chrome, the same in both themes: the lint
 * underlines and gutter marks, the hover card, signature help and the
 * inline values. Every colour is a token, so it follows the theme.
 */
export const languageServiceTheme = EditorView.theme({
  ".cm-lintRange-error": {
    backgroundImage: "none",
    textDecoration: "underline wavy var(--danger)",
    textUnderlineOffset: "3px",
    textDecorationSkipInk: "none",
  },
  ".cm-lintRange-warning": {
    backgroundImage: "none",
    textDecoration: "underline wavy var(--warning)",
    textUnderlineOffset: "3px",
    textDecorationSkipInk: "none",
  },
  ".cm-lintRange-info": {
    backgroundImage: "none",
    textDecoration: "underline dotted var(--foreground-subtle)",
    textUnderlineOffset: "3px",
  },
  ".cm-lintRange-active": {
    backgroundColor: "color-mix(in oklab, var(--warning) 14%, transparent)",
  },
  ".cm-lintPoint::after": {
    borderBottomColor: "var(--danger)",
  },
  ".cm-gutter-lint": {
    width: "14px",
  },
  ".cm-gutter-lint .cm-gutterElement": {
    padding: "0 0 0 4px",
  },
  ".cm-lint-marker": {
    width: "6px",
    height: "6px",
    borderRadius: "999px",
    content: "none",
    display: "inline-block",
    marginTop: "7px",
  },
  ".cm-lint-marker::before": { content: "none" },
  ".cm-lint-marker-error": { backgroundColor: "var(--danger)" },
  ".cm-lint-marker-warning": { backgroundColor: "var(--warning)" },
  ".cm-lint-marker-info": { backgroundColor: "var(--foreground-subtle)" },
  ".cm-tooltip.cm-tooltip-lint": {
    padding: "0",
    maxWidth: "380px",
  },
  ".cm-tooltip-lint .cm-diagnostic": {
    padding: "7px 10px 7px 22px",
    borderLeft: "none",
    position: "relative",
    fontSize: "12px",
    lineHeight: "1.45",
    fontFamily: "inherit",
    color: "var(--foreground)",
  },
  ".cm-tooltip-lint .cm-diagnostic + .cm-diagnostic": {
    borderTop: "1px solid var(--border)",
  },
  ".cm-tooltip-lint .cm-diagnostic::before": {
    content: "''",
    position: "absolute",
    left: "10px",
    top: "13px",
    width: "6px",
    height: "6px",
    borderRadius: "999px",
    backgroundColor: "var(--foreground-subtle)",
  },
  ".cm-tooltip-lint .cm-diagnostic-error::before": { backgroundColor: "var(--danger)" },
  ".cm-tooltip-lint .cm-diagnostic-warning::before": { backgroundColor: "var(--warning)" },
  ".cm-tooltip-lint .cm-diagnosticText": { display: "block" },
  ".cm-tooltip-lint .cm-diagnosticSource": { display: "none" },
  ".cm-tooltip-lint .cm-diagnosticAction": {
    display: "inline-flex",
    alignItems: "center",
    height: "22px",
    marginTop: "6px",
    marginRight: "6px",
    marginLeft: "0",
    padding: "0 8px",
    borderRadius: "var(--radius-control)",
    backgroundColor: "var(--ink)",
    color: "var(--ink-foreground)",
    fontSize: "11px",
    fontWeight: "500",
    fontFamily: "inherit",
    cursor: "pointer",
  },
  ".cm-tooltip .cm-expr-hover": {
    display: "flex",
    flexDirection: "column",
    gap: "4px",
    padding: "8px 10px",
    maxWidth: "340px",
    fontSize: "12px",
    fontFamily: "inherit",
    whiteSpace: "normal",
    color: "var(--foreground)",
  },
  ".cm-expr-hover-head": {
    display: "flex",
    flexWrap: "wrap",
    alignItems: "center",
    gap: "6px",
  },
  ".cm-expr-hover-name": {
    fontFamily: "'Geist Mono', monospace",
    fontWeight: "600",
    fontSize: "12px",
  },
  ".cm-expr-hover-tag": {
    display: "inline-flex",
    alignItems: "center",
    height: "16px",
    padding: "0 6px",
    borderRadius: "999px",
    backgroundColor: "color-mix(in oklab, var(--muted-foreground) 14%, transparent)",
    color: "var(--muted-foreground)",
    fontSize: "10.5px",
    lineHeight: "1",
  },
  ".cm-expr-hover-tag-warn": {
    backgroundColor: "color-mix(in oklab, var(--warning) 18%, transparent)",
    color: "var(--warning-foreground)",
  },
  ".cm-expr-hover-tag-bad": {
    backgroundColor: "color-mix(in oklab, var(--danger) 18%, transparent)",
    color: "var(--danger-foreground)",
  },
  ".cm-expr-hover-body": {
    color: "var(--muted-foreground)",
    lineHeight: "1.45",
  },
  ".cm-expr-hover-example": {
    display: "block",
    fontFamily: "'Geist Mono', monospace",
    fontSize: "11px",
    backgroundColor: "color-mix(in oklab, var(--muted-foreground) 12%, transparent)",
    borderRadius: "4px",
    padding: "3px 6px",
  },
  ".cm-expr-hover-value": {
    display: "flex",
    alignItems: "baseline",
    gap: "8px",
    marginTop: "2px",
    paddingTop: "6px",
    borderTop: "1px solid var(--border)",
  },
  ".cm-expr-hover-value-text": {
    fontFamily: "'Geist Mono', monospace",
    fontSize: "12px",
    fontVariantNumeric: "tabular-nums",
  },
  ".cm-expr-hover-source": {
    color: "var(--muted-foreground)",
    fontSize: "11px",
  },
  ".cm-tooltip .cm-expr-signature": {
    display: "flex",
    flexDirection: "column",
    gap: "3px",
    padding: "6px 10px",
    maxWidth: "360px",
    fontFamily: "inherit",
    fontSize: "12px",
    color: "var(--foreground)",
  },
  ".cm-expr-signature-code": {
    fontFamily: "'Geist Mono', monospace",
    fontSize: "12px",
    color: "var(--muted-foreground)",
  },
  ".cm-expr-signature-active": {
    color: "var(--foreground)",
    fontWeight: "600",
    backgroundColor: "color-mix(in oklab, var(--muted-foreground) 14%, transparent)",
    borderRadius: "3px",
    padding: "0 3px",
  },
  ".cm-expr-signature-body": {
    color: "var(--muted-foreground)",
    fontSize: "11.5px",
  },
  ".cm-expr-inline-value": {
    marginLeft: "14px",
    padding: "0 6px",
    borderRadius: "999px",
    fontFamily: "'Geist Mono', monospace",
    fontSize: "11px",
    fontVariantNumeric: "tabular-nums",
    verticalAlign: "baseline",
    userSelect: "none",
    pointerEvents: "none",
  },
  ".cm-expr-inline-value-ok": {
    color: "var(--success-foreground)",
    backgroundColor: "color-mix(in oklab, var(--success) 12%, transparent)",
  },
  ".cm-expr-inline-value-bad": {
    color: "var(--danger-foreground)",
    backgroundColor: "color-mix(in oklab, var(--danger) 12%, transparent)",
  },
  ".cm-expr-inline-value-muted": {
    color: "var(--muted-foreground)",
    backgroundColor: "color-mix(in oklab, var(--muted-foreground) 12%, transparent)",
  },
});
