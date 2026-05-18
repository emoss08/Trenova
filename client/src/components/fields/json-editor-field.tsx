import { FieldWrapper } from "@/components/fields/field-components";
import { useTheme } from "@/components/theme-provider";
import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import { json, jsonParseLinter } from "@codemirror/lang-json";
import { linter } from "@codemirror/lint";
import { EditorView } from "@codemirror/view";
import CodeMirror, { type ReactCodeMirrorProps } from "@uiw/react-codemirror";
import { useMemo } from "react";
import { Controller, type FieldValues } from "react-hook-form";

type JsonEditorFieldProps<T extends FieldValues> = FormControlProps<T> &
  Omit<ReactCodeMirrorProps, "value" | "onChange" | "theme" | "extensions"> & {
    label: string;
    description?: string;
    disabled?: boolean;
    className?: string;
    editorClassName?: string;
  };

export function JsonEditorField<T extends FieldValues>({
  name,
  label,
  description,
  control,
  rules,
  disabled,
  className,
  editorClassName,
  ...props
}: JsonEditorFieldProps<T>) {
  const { theme } = useTheme();
  const extensions = useMemo(
    () => [json(), linter(jsonParseLinter()), EditorView.lineWrapping],
    [],
  );

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          error={fieldState.error?.message}
          required={!!rules?.required}
          className={className}
        >
          <div
            className={cn(
              "overflow-hidden rounded-md border border-input transition-all duration-200",
              "focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/30 focus-within:outline-hidden",
              fieldState.invalid &&
                "border-destructive focus-within:border-destructive focus-within:ring-destructive/20",
              disabled && "opacity-70",
              editorClassName,
            )}
          >
            <CodeMirror
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              extensions={extensions}
              aria-invalid={fieldState.invalid ? "true" : "false"}
              editable={!disabled}
              readOnly={disabled}
              basicSetup={{
                lineNumbers: true,
                foldGutter: false,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: false,
                highlightActiveLine: true,
                highlightActiveLineGutter: true,
                highlightSelectionMatches: true,
                searchKeymap: false,
              }}
              theme={theme === "dark" ? darkTheme : lightTheme}
              {...props}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
