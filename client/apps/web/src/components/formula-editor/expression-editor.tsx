import { FieldWrapper } from "@/components/fields/field-components";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type { VariableDefinitionInput } from "@trenova/shared/types/formula-template";
import { EditorView } from "@codemirror/view";
import CodeMirror, {
  type ReactCodeMirrorProps,
  type ReactCodeMirrorRef,
} from "@uiw/react-codemirror";
import { useMemo, useRef, type Ref } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { darkTheme, lightTheme } from "./editor-theme";
import { createExprLinter } from "./expr-lint";
import { buildKnownIdentifiers, exprLanguageSupport, type KnownIdentifiers } from "./expr-language";

type ExpressionEditorProps<T extends FieldValues> = FormControlProps<T> &
  ReactCodeMirrorProps & {
    customVariables?: VariableDefinitionInput[];
    knownIdentifiers?: KnownIdentifiers;
    label?: string;
    description?: string;
    variant?: "default" | "mini";
    lint?: boolean;
    editorRef?: Ref<ReactCodeMirrorRef>;
  };

export function ExpressionEditor<T extends FieldValues>({
  name,
  label,
  description,
  customVariables = [],
  knownIdentifiers,
  variant = "default",
  lint = true,
  editorRef,
  control,
  rules,
  ...props
}: ExpressionEditorProps<T>) {
  const { theme } = useTheme();

  const known = useMemo(
    () => knownIdentifiers ?? buildKnownIdentifiers(undefined, customVariables),
    [knownIdentifiers, customVariables],
  );

  const knownRef = useRef(known);
  knownRef.current = known;

  const extensions = useMemo(() => {
    const list = [exprLanguageSupport(known), EditorView.lineWrapping];
    if (lint) {
      list.push(createExprLinter(() => knownRef.current));
    }
    return list;
  }, [known, lint]);

  const isMini = variant === "mini";

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
              "border-input overflow-hidden rounded-md border transition-all duration-200",
              "focus-within:border-brand focus-within:ring-brand/30 focus-within:ring-4 focus-within:outline-hidden",
              fieldState.invalid &&
                "border-destructive focus-within:border-destructive focus-within:ring-destructive/20 focus-within:ring-4",
            )}
          >
            <CodeMirror
              ref={editorRef}
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              extensions={extensions}
              aria-invalid={fieldState.invalid ? "true" : "false"}
              minHeight={isMini ? "60px" : undefined}
              basicSetup={{
                lineNumbers: !isMini,
                foldGutter: false,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
                highlightActiveLine: !isMini,
                highlightActiveLineGutter: !isMini,
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
