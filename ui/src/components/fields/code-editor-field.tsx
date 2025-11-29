import { useLocalStorage } from "@/hooks/use-local-storage";
import { cn } from "@/lib/utils";
import { css } from "@codemirror/lang-css";
import { html } from "@codemirror/lang-html";
import { EditorView, keymap } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import { vim, Vim } from "@replit/codemirror-vim";
import { createTheme } from "@uiw/codemirror-themes";
import CodeMirror, { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useCallback, useEffect, useRef } from "react";
import type { Control, RegisterOptions } from "react-hook-form";
import { Controller, FieldValues, Path } from "react-hook-form";
import { useTheme } from "../theme-provider";
import { FieldWrapper } from "./field-components";

type CodeLanguage = "html" | "css";

interface CodeEditorFieldProps<T extends FieldValues> {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
  label?: string;
  description?: string;
  language?: CodeLanguage;
  height?: string;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

const lightTheme = createTheme({
  theme: "light",
  settings: {
    background: "#fafafa",
    backgroundImage: "",
    foreground: "#1f2937",
    caret: "#1f2937",
    selection: "#e5e7eb",
    selectionMatch: "#d1d5db",
    gutterBackground: "#f3f4f6",
    gutterForeground: "#6b7280",
    gutterBorder: "transparent",
    gutterActiveForeground: "#1f2937",
    lineHighlight: "#f3f4f6",
  },
  styles: [
    { tag: t.comment, color: "#6b7280" },
    { tag: t.string, color: "#059669" },
    { tag: t.keyword, color: "#7c3aed" },
    { tag: t.tagName, color: "#2563eb" },
    { tag: t.attributeName, color: "#d97706" },
    { tag: t.attributeValue, color: "#059669" },
    { tag: t.propertyName, color: "#2563eb" },
    { tag: t.number, color: "#dc2626" },
    { tag: t.operator, color: "#6b7280" },
    { tag: t.bracket, color: "#6b7280" },
    { tag: t.className, color: "#d97706" },
    { tag: t.definition(t.variableName), color: "#7c3aed" },
  ],
});

const darkTheme = createTheme({
  theme: "dark",
  settings: {
    background: "#0a0a0a",
    backgroundImage: "",
    foreground: "#e5e7eb",
    caret: "#ffffff",
    selection: "#1f2937",
    selectionMatch: "#374151",
    gutterBackground: "#0a0a0a",
    gutterForeground: "#6b7280",
    gutterBorder: "transparent",
    gutterActiveForeground: "#e5e7eb",
    lineHighlight: "#111827",
  },
  styles: [
    { tag: t.comment, color: "#6b7280" },
    { tag: t.string, color: "#34d399" },
    { tag: t.keyword, color: "#a78bfa" },
    { tag: t.tagName, color: "#60a5fa" },
    { tag: t.attributeName, color: "#fbbf24" },
    { tag: t.attributeValue, color: "#34d399" },
    { tag: t.propertyName, color: "#60a5fa" },
    { tag: t.number, color: "#f87171" },
    { tag: t.operator, color: "#9ca3af" },
    { tag: t.bracket, color: "#9ca3af" },
    { tag: t.className, color: "#fbbf24" },
    { tag: t.definition(t.variableName), color: "#a78bfa" },
  ],
});

export function CodeEditorField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  language = "html",
  height = "300px",
  placeholder,
  disabled,
  className,
}: CodeEditorFieldProps<T>) {
  const inputId = `code-editor-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const { theme } = useTheme();

  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const [vimEnabled, setVimEnabled] = useLocalStorage(
    "code-editor-vim-mode",
    false,
  );

  const toggleVimMode = useCallback(() => {
    setVimEnabled(!vimEnabled);
  }, [vimEnabled, setVimEnabled]);

  useEffect(() => {
    if (vimEnabled) {
      Vim.map("jj", "<Esc>", "insert");
      Vim.map("jk", "<Esc>", "insert");
      Vim.map("H", "^", "normal");
      Vim.map("L", "$", "normal");

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

  const languageExtension = language === "css" ? css() : html();

  const extensions = vimEnabled
    ? [
        vim({ status: true }),
        languageExtension,
        EditorView.lineWrapping,
        toggleVimKeymap,
      ]
    : [languageExtension, EditorView.lineWrapping, toggleVimKeymap];

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
          className={className}
        >
          <div
            className={cn(
              "relative overflow-hidden rounded-md border border-muted-foreground/20 bg-background",
              fieldState.invalid &&
                "border-red-500 bg-red-500/10 ring-0 ring-red-500",
              disabled && "cursor-not-allowed opacity-50",
            )}
            style={{ height }}
            id={inputId}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
            )}
          >
            {vimEnabled && (
              <div className="pointer-events-none absolute top-2 right-2 z-10">
                <div className="rounded bg-primary/10 px-1.5 py-0.5 text-2xs font-medium text-primary">
                  VIM
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
                foldGutter: true,
                dropCursor: false,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
                rectangularSelection: false,
                highlightSelectionMatches: true,
                searchKeymap: true,
              }}
              theme={theme === "dark" ? darkTheme : lightTheme}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
