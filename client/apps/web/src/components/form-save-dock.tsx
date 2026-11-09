import {
  ACTION_DOCK_SECONDARY_BUTTON,
  ActionDock,
  type ActionDockPosition,
} from "@/components/action-dock";
import { useChangedFieldNames } from "@/hooks/use-dirty-fields";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormFieldRegistry } from "@trenova/shared/lib/form-field-registry";
import { AnimatePresence } from "motion/react";
import type { ReactNode } from "react";
import { useCallback, useRef, type Ref } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { UnsavedChanges, type ChangeFields } from "./form-changes";

const NO_CHANGE_FIELDS: ChangeFields = {};

type DockPosition = ActionDockPosition;

interface SplitButtonConfig<T extends string = string> {
  options: SplitButtonOption<T>[];
  selectedOption: T;
  onOptionSelect: (optionId: T) => void;
  loadingText?: string;
}

/** The panel asking whether to throw away unsaved changes before it closes. */
export type DockClosePrompt = {
  count: number;
  onKeepEditing: () => void;
  onDiscard: () => void;
};

interface FormSaveDockProps<T extends string = string> {
  /**
   * Set while the panel is asking before it closes; the dock asks in place of
   * offering the save, since the panel's own footer sits under it.
   */
  closePrompt?: DockClosePrompt | null;
  /** Labels for the change review the dock opens; see FormCreatePanel. */
  changeFields?: ChangeFields;
  saveButtonContent?: ReactNode;
  position?: DockPosition;
  width?: string;
  className?: string;
  splitButton?: SplitButtonConfig<T>;
  formId?: string;
  alwaysVisible?: boolean;
  showReset?: boolean;
  requireInteraction?: boolean;
  showHeightGap?: boolean;
}

function SaveDockContent<T extends string = string>({
  saveButtonContent,
  position,
  width,
  className,
  isSubmitting,
  isDirty,
  onReset,
  splitButton,
  formId,
  showReset = true,
  changes,
  dockRef,
}: FormSaveDockProps<T> & {
  isSubmitting: boolean;
  isDirty: boolean;
  onReset: () => void;
  /** The live count of unsaved changes, in place of the fixed message. */
  changes?: ReactNode;
  /** The dock's pill, which the change review centres over. */
  dockRef?: Ref<HTMLDivElement>;
}) {
  const t = useT();

  return (
    <ActionDock
      position={position}
      width={width}
      className={className}
      pillRef={dockRef}
      indicator={isDirty ? changes : undefined}
    >
      {showReset && (
        <Button
          type="button"
          variant="outline"
          onClick={onReset}
          disabled={isSubmitting}
          className={ACTION_DOCK_SECONDARY_BUTTON}
        >
          {t("Reset")}
        </Button>
      )}
      {splitButton ? (
        <SplitButton
          options={splitButton.options}
          selectedOption={splitButton.selectedOption}
          onOptionSelect={splitButton.onOptionSelect}
          isLoading={isSubmitting}
          loadingText={splitButton.loadingText}
          formId={formId}
        />
      ) : (
        <Button
          type="submit"
          variant="ghostInvert"
          className="pr-2"
          disabled={isSubmitting}
          form={formId}
        >
          {isSubmitting ? <Spinner /> : (saveButtonContent ?? t("Save changes"))}
        </Button>
      )}
    </ActionDock>
  );
}

/** The dock asking whether to discard, in place of the save. */
function ClosePromptDock({
  prompt,
  position,
  width,
  className,
}: {
  prompt: DockClosePrompt;
  position?: DockPosition;
  width?: string;
  className?: string;
}) {
  const t = useT();

  return (
    <ActionDock
      position={position}
      width={width}
      className={className}
      pillClassName="w-112.5 gap-x-4"
      indicator={
        <div role="alert" className="flex min-w-0 items-center gap-x-3">
          <AlertCircleIcon className="bg-warning-subtle text-warning-foreground shrink-0 rounded-full" />
          <span
            className="text-background min-w-0 truncate text-sm font-medium whitespace-nowrap"
            title={
              prompt.count === 1
                ? t("Discard 1 unsaved change?")
                : t("Discard {0} unsaved changes?", prompt.count)
            }
          >
            {prompt.count === 1
              ? t("Discard 1 unsaved change?")
              : t("Discard {0} unsaved changes?", prompt.count)}
          </span>
        </div>
      }
    >
      <Button
        type="button"
        size="sm"
        variant="outline"
        autoFocus
        onClick={prompt.onKeepEditing}
        className={ACTION_DOCK_SECONDARY_BUTTON}
      >
        {t("Keep editing")}
      </Button>
      <Button type="button" size="sm" variant="destructive" onClick={prompt.onDiscard}>
        {t("Discard and close")}
      </Button>
    </ActionDock>
  );
}

export function FormSaveDock<T extends string = string>({
  closePrompt,
  changeFields,
  saveButtonContent,
  position = "center",
  width = "350px",
  className,
  splitButton,
  formId,
  alwaysVisible = false,
  showReset = true,
  requireInteraction = false,
  showHeightGap = true,
}: FormSaveDockProps<T>) {
  const form = useFormContext();
  const dockRef = useRef<HTMLDivElement>(null);
  const { control, reset } = form;
  const registry = useFormFieldRegistry();

  const { isDirty, touchedFields, isSubmitting } = useFormState({
    control,
  });
  const changed = useChangedFieldNames(control);

  const handleReset = useCallback(() => reset(), [reset]);

  const hasDirtyFields = isDirty && changed.length > 0;
  const liveChanges = hasDirtyFields ? (
    <UnsavedChanges
      form={form}
      changed={changed}
      fields={changeFields ?? NO_CHANGE_FIELDS}
      registry={registry}
      tone="dock"
      anchor={dockRef}
    />
  ) : undefined;
  const hasTouchedFields = Object.keys(touchedFields).length > 0;

  const visible = alwaysVisible
    ? true
    : requireInteraction
      ? hasDirtyFields && hasTouchedFields
      : hasDirtyFields;

  return (
    <>
      {showHeightGap && (closePrompt || visible) ? <div className="h-16" /> : null}
      <AnimatePresence mode="wait">
        {closePrompt ? (
          <ClosePromptDock
            key="close-prompt"
            prompt={closePrompt}
            position={position}
            width={width}
            className={className}
          />
        ) : visible ? (
          <SaveDockContent
            key="save"
            saveButtonContent={saveButtonContent}
            position={position}
            width={width}
            className={className}
            isSubmitting={isSubmitting}
            isDirty={isDirty}
            onReset={handleReset}
            splitButton={splitButton}
            formId={formId}
            showReset={showReset}
            changes={liveChanges}
            dockRef={dockRef}
          />
        ) : null}
      </AnimatePresence>
    </>
  );
}
