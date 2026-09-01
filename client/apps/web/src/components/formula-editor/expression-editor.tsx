import { FieldWrapper } from "@/components/fields/field-components";
import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type { VariableDefinitionInput } from "@trenova/shared/types/formula-template";
import type { EditorView } from "@codemirror/view";
import CodeMirror, {
  type ReactCodeMirrorProps,
  type ReactCodeMirrorRef,
} from "@uiw/react-codemirror";
import { useCallback, useEffect, useMemo, useRef, type Ref } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { useActiveEditorRegistration } from "./active-editor";
import { darkTheme, lightTheme } from "./editor-theme";
import { buildKnownIdentifiers, type KnownIdentifiers } from "./expr-language";
import { useExprExtensions } from "./use-expr-extensions";

type ExpressionEditorProps<T extends FieldValues> = FormControlProps<T> &
  ReactCodeMirrorProps & {
    customVariables?: VariableDefinitionInput[];
    knownIdentifiers?: KnownIdentifiers;
    label?: string;
    description?: string;
    variant?: "default" | "mini";
    lint?: boolean;
    editorRef?: Ref<ReactCodeMirrorRef>;
    /** The editor reference-pane inserts fall back to when nothing else was focused. */
    primary?: boolean;
  };

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (!ref) return;
  if (typeof ref === "function") {
    ref(value);
  } else {
    (ref as { current: T | null }).current = value;
  }
}

export function ExpressionEditor<T extends FieldValues>({
  name,
  label,
  description,
  customVariables = [],
  knownIdentifiers,
  variant = "default",
  lint = true,
  editorRef,
  primary = false,
  control,
  rules,
  ...props
}: ExpressionEditorProps<T>) {
  const resolvedTheme = useResolvedTheme();
  const { setActive, setPrimary } = useActiveEditorRegistration();

  const known = useMemo(
    () => knownIdentifiers ?? buildKnownIdentifiers(undefined, customVariables),
    [knownIdentifiers, customVariables],
  );

  const viewRef = useRef<EditorView | null>(null);
  const extensions = useExprExtensions(known, lint, viewRef);

  const handleRef = useCallback(
    (instance: ReactCodeMirrorRef | null) => {
      viewRef.current = instance?.view ?? null;
      assignRef(editorRef, instance);
      if (primary) {
        setPrimary?.(instance?.view ?? null);
      }
    },
    [editorRef, primary, setPrimary],
  );

  // The view is created after the ref callback fires, so the primary
  // registration is refreshed once it exists.
  useEffect(() => {
    if (primary && viewRef.current) {
      setPrimary?.(viewRef.current);
    }
  });

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
            onFocusCapture={() => setActive?.(viewRef.current)}
            className={cn(
              "border-input overflow-hidden rounded-md border transition-all duration-200",
              "focus-within:border-brand focus-within:ring-brand/30 focus-within:ring-4 focus-within:outline-hidden",
              fieldState.invalid &&
                "border-destructive focus-within:border-destructive focus-within:ring-destructive/20 focus-within:ring-4",
            )}
          >
            <CodeMirror
              ref={handleRef}
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
              theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
              {...props}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
