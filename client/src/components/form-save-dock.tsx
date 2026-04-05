import { Button } from "@/components/ui/button";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { cn } from "@/lib/utils";
import { CircleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { Spinner } from "./ui/spinner";

type DockPosition = "center" | "left" | "right";

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
  const positionClasses = {
    center: "left-1/2 transform -translate-x-1/2",
    left: "left-20",
    right: "right-35",
  };

  const showUnsavedIndicator = isDirty || !alwaysVisible;

  return (
    <div
      className={cn("fixed bottom-6 z-50", positionClasses[position || "center"], className)}
      style={{ width }}
    >
      <div className="flex w-fit min-w-[450px] items-center gap-x-10 rounded-lg bg-foreground p-2 shadow-lg">
        {showUnsavedIndicator && (
          <div className="flex items-center gap-x-3">
            <CircleAlert className="rounded-full bg-amber-400/10 text-amber-400 dark:text-amber-600" />
            <div className="flex flex-col">
              <span className="text-sm font-medium text-background">{unsavedText}</span>
              <span className="text-2xs text-background/80">You have unsaved changes.</span>
            </div>
          </div>
        )}
        <div className="ml-auto flex items-center space-x-2">
          {showReset && (
            <Button
              type="reset"
              variant="outline"
              onClick={onReset}
              disabled={isSubmitting}
              className="border-none bg-white/20 text-background hover:bg-white/30 hover:text-background dark:bg-black/20 dark:hover:bg-black/30"
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
        </div>
      </div>
    </div>
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
        keepValues: false,
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
      <div className="h-16" />
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
