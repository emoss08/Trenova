import {
  ACTION_DOCK_SECONDARY_BUTTON,
  ActionDock,
  ActionDockIndicator,
  type ActionDockPosition,
} from "@/components/action-dock";
import { Button } from "@trenova/shared/components/ui/button";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import type { ReactNode } from "react";
import { useCallback } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { Spinner } from "@trenova/shared/components/ui/spinner";

type DockPosition = ActionDockPosition;

interface SplitButtonConfig<T extends string = string> {
  options: SplitButtonOption<T>[];
  selectedOption: T;
  onOptionSelect: (optionId: T) => void;
  loadingText?: string;
}

interface FormSaveDockProps<T extends string = string> {
  saveButtonContent?: ReactNode;
  unsavedText?: string;
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
  unsavedText,
  position,
  width,
  className,
  isSubmitting,
  isDirty,
  onReset,
  splitButton,
  formId,
  showReset = true,
  alwaysVisible,
}: FormSaveDockProps<T> & {
  isSubmitting: boolean;
  isDirty: boolean;
  onReset: () => void;
}) {
  const showUnsavedIndicator = isDirty || !alwaysVisible;

  return (
    <ActionDock
      position={position}
      width={width}
      className={className}
      indicator={
        showUnsavedIndicator ? (
          <ActionDockIndicator
            title={unsavedText ?? "Unsaved changes"}
            description="You have unsaved changes."
          />
        ) : undefined
      }
    >
      {showReset && (
        <Button
          type="reset"
          variant="outline"
          onClick={onReset}
          disabled={isSubmitting}
          className={ACTION_DOCK_SECONDARY_BUTTON}
        >
          Reset
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
          variant="default"
          className="pr-2"
          disabled={isSubmitting}
          form={formId}
        >
          {isSubmitting ? <Spinner /> : (saveButtonContent ?? "Save")}
        </Button>
      )}
    </ActionDock>
  );
}

export function FormSaveDock<T extends string = string>({
  saveButtonContent = "Save",
  unsavedText = "Unsaved changes",
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
  const { control, reset } = useFormContext();

  const { isDirty, dirtyFields, touchedFields, isSubmitting } = useFormState({
    control,
  });

  const handleReset = useCallback(() => {
    reset(
      {},
      {
        keepDirty: false,
        keepValues: true,
      },
    );
  }, [reset]);

  const hasDirtyFields = isDirty && Object.keys(dirtyFields).length > 0;
  const hasTouchedFields = Object.keys(touchedFields).length > 0;

  if (alwaysVisible) {
    // always show
  } else if (requireInteraction) {
    if (!hasDirtyFields || !hasTouchedFields) {
      return null;
    }
  } else if (!hasDirtyFields) {
    return null;
  }

  return (
    <>
      {showHeightGap && <div className="h-16" />}
      <SaveDockContent
        saveButtonContent={saveButtonContent}
        unsavedText={unsavedText}
        position={position}
        width={width}
        className={className}
        isSubmitting={isSubmitting}
        isDirty={isDirty}
        onReset={handleReset}
        splitButton={splitButton}
        formId={formId}
        alwaysVisible={alwaysVisible}
        showReset={showReset}
      />
    </>
  );
}
