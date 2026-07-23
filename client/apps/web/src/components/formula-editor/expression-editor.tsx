import { FieldWrapper } from "@/components/fields/field-components";
import { useTheme } from "@/components/theme-provider";
import { cn } from "@/lib/utils";
import type { FormControlProps } from "@/types/fields";
import type { VariableDefinitionInput } from "@/types/formula-template";
import { EditorView } from "@codemirror/view";
import CodeMirror, { type ReactCodeMirrorProps } from "@uiw/react-codemirror";
import { Controller, type FieldValues } from "react-hook-form";
import { darkTheme, lightTheme } from "./editor-theme";
import { exprLanguageSupport } from "./expr-language";

type ExpressionEditorProps<T extends FieldValues> = FormControlProps<T> &
  ReactCodeMirrorProps & {
    customVariables?: VariableDefinitionInput[];
    label?: string;
    description?: string;
  };

export function ExpressionEditor<T extends FieldValues>({
  name,
  label,
  description,
  customVariables = [],
  control,
  rules,
  ...props
}: ExpressionEditorProps<T>) {
  const { theme } = useTheme();

  const extensions = [
    exprLanguageSupport(customVariables),
    EditorView.lineWrapping,
  ];

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
        >
          <div
            className={cn(
              "overflow-hidden rounded-md border border-input transition-all duration-200",
              "focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/30 focus-within:outline-hidden",
              fieldState.invalid &&
                "border-destructive focus-within:border-destructive focus-within:ring-4 focus-within:ring-destructive/20",
            )}
          >
            <CodeMirror
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              extensions={extensions}
              aria-invalid={fieldState.invalid ? "true" : "false"}
              basicSetup={{
                lineNumbers: true,
                foldGutter: false,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
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
