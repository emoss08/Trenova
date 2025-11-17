import { useLocalStorage } from "@/hooks/use-local-storage";
import { cn } from "@/lib/utils";
import type { TextareaFieldProps } from "@/types/fields";
import { json } from "@codemirror/lang-json";
import { PostgreSQL, sql } from "@codemirror/lang-sql";
import { EditorView, keymap } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import { vim, Vim } from "@replit/codemirror-vim";
import { createTheme } from "@uiw/codemirror-themes";
import CodeMirror, { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useCallback, useEffect, useRef } from "react";
import { Controller, FieldValues } from "react-hook-form";
import { useTheme } from "../theme-provider";
import { FieldWrapper } from "./field-components";

interface SQLEditorFieldProps<T extends FieldValues>
  extends Omit<TextareaFieldProps<T>, "rows" | "cols"> {
  height?: string;
}

const lightTheme = createTheme({
  theme: "light",
  settings: {
    background: "#f3f3f3",
    backgroundImage: "",
    foreground: "#000000",
    caret: "#000000",
    selection: "#d6d6d6",
    selectionMatch: "#dcdbdb",
    gutterBackground: "#f3f3f3",
    gutterForeground: "#000000",
    gutterBorder: "#f3f3f3",
    gutterActiveForeground: "",
    lineHighlight: "#d6d6d6",
  },
  styles: [
    { tag: t.comment, color: "#787b80" },
    { tag: t.name, color: "#f14445" },
    { tag: t.definition(t.typeName), color: "#111826" },
    { tag: t.typeName, color: "#f96f70" },
    { tag: t.tagName, color: "#008a02" },
    { tag: t.variableName, color: "#f14445" },
  ],
});

const darkTheme = createTheme({
  theme: "dark",
  settings: {
    background: "#0d0d0d",
    backgroundImage: "",
    foreground: "#e3e3e3",
    caret: "#ffffff",
    selection: "#171717",
    selectionMatch: "#0e0e0e",
    gutterBackground: "#0e0e0e",
    gutterForeground: "#666666",
    gutterBorder: "#7c7979",
    gutterActiveForeground: "",
    lineHighlight: "#171717",
  },
  styles: [
    { tag: t.comment, color: "#787b80" },
    { tag: t.name, color: "#f14445" },
    { tag: t.definition(t.typeName), color: "#111826" },
    { tag: t.typeName, color: "#f96f70" },
    { tag: t.tagName, color: "#008a02" },
    { tag: t.variableName, color: "#f14445" },
  ],
});

export function SQLEditorField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  placeholder,
  height = "150px",
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
}: SQLEditorFieldProps<T>) {
  const inputId = `sql-editor-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const { theme } = useTheme();

  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const [vimEnabled, setVimEnabled] = useLocalStorage(
    "sql-editor-vim-mode",
    false,
  );

  const toggleVimMode = useCallback(() => {
    setVimEnabled(!vimEnabled);
  }, [vimEnabled, setVimEnabled]);

  useEffect(() => {
    if (vimEnabled) {
      Vim.map("jj", "<Esc>", "insert");
      Vim.map("jk", "<Esc>", "insert");

      Vim.map("H", "^", "normal"); // H goes to first non-blank character
      Vim.map("L", "$", "normal"); // L goes to end of line

      return () => {
        Vim.unmap("jj", "insert");
        Vim.unmap("jk", "insert");
        Vim.unmap("H", "normal");
        Vim.unmap("L", "normal");
      };
    }
  }, [vimEnabled]);

  const toggleVimKeymap = keymap.of([
    {
      key: "Mod-Alt-v",
      run: () => {
        toggleVimMode();
        return true;
      },
    },
  ]);

  const extensions = vimEnabled
    ? [vim({ status: true }), json(), EditorView.lineWrapping, toggleVimKeymap]
    : [sql({ dialect: PostgreSQL }), EditorView.lineWrapping, toggleVimKeymap];

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
        >
          <div
            className={cn(
              "relative overflow-hidden rounded-md border border-muted-foreground/20",
              fieldState.invalid &&
                "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 [&_.cm-editor.cm-focused]:border-red-600",
              disabled && "opacity-50 cursor-not-allowed",
              className,
            )}
            id={inputId}
            aria-label={ariaLabel || label}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
          >
            {vimEnabled && (
              <div className="pointer-events-none absolute top-2 right-2 z-10">
                <div className="rounded bg-primary/10 px-2 py-0.5 text-xs text-primary">
                  Vim Mode
                </div>
              </div>
            )}
            <CodeMirror
              ref={editorRef}
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              onBlur={() => field.onBlur()}
              height={height}
              extensions={extensions}
              editable={!disabled}
              placeholder={placeholder}
              basicSetup={{
                lineNumbers: true,
                foldGutter: false,
                dropCursor: false,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
                rectangularSelection: false,
                highlightSelectionMatches: false,
                searchKeymap: false,
              }}
              theme={theme === "dark" ? darkTheme : lightTheme}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}

export function JSONEditorField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  placeholder,
  height = "150px",
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
}: SQLEditorFieldProps<T>) {
  const inputId = `json-editor-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const { theme } = useTheme();

  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const [vimEnabled, setVimEnabled] = useLocalStorage(
    "json-editor-vim-mode",
    false,
  );

  const toggleVimMode = useCallback(() => {
    setVimEnabled(!vimEnabled);
  }, [vimEnabled, setVimEnabled]);

  useEffect(() => {
    if (vimEnabled) {
      Vim.map("jj", "<Esc>", "insert");
      Vim.map("jk", "<Esc>", "insert");

      Vim.map("H", "^", "normal"); // H goes to first non-blank character
      Vim.map("L", "$", "normal"); // L goes to end of line

      return () => {
        Vim.unmap("jj", "insert");
        Vim.unmap("jk", "insert");
        Vim.unmap("H", "normal");
        Vim.unmap("L", "normal");
      };
    }
  }, [vimEnabled]);

  const toggleVimKeymap = keymap.of([
    {
      key: "Mod-Alt-v",
      run: () => {
        toggleVimMode();
        return true;
      },
    },
  ]);

  const extensions = vimEnabled
    ? [vim({ status: true }), json(), EditorView.lineWrapping, toggleVimKeymap]
    : [json(), EditorView.lineWrapping, toggleVimKeymap];

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
        >
          <div
            className={cn(
              "relative overflow-hidden rounded-md border border-muted-foreground/20",
              fieldState.invalid &&
                "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 [&_.cm-editor.cm-focused]:border-red-600",
              disabled && "opacity-50 cursor-not-allowed",
              className,
            )}
            id={inputId}
            aria-label={ariaLabel || label}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy,
            )}
          >
            {vimEnabled && (
              <div className="pointer-events-none absolute top-2 right-2 z-10">
                <div className="rounded bg-primary/10 px-2 py-0.5 text-xs text-primary">
                  Vim Mode
                </div>
              </div>
            )}
            <CodeMirror
              ref={editorRef}
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              onBlur={() => field.onBlur()}
              height={height}
              extensions={extensions}
              editable={!disabled}
              placeholder={placeholder}
              basicSetup={{
                lineNumbers: true,
                foldGutter: false,
                dropCursor: false,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
                rectangularSelection: false,
                highlightSelectionMatches: false,
                searchKeymap: false,
              }}
              theme={theme === "dark" ? darkTheme : lightTheme}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
