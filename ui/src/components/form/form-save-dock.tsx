import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import { faCircleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { ReactNode, useCallback } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { PulsatingDots } from "../ui/pulsating-dots";

type DockPosition = "center" | "left" | "right";

interface FormSaveDockProps {
  /** Custom save button content */
  saveButtonContent?: ReactNode;

  /** Custom text to display in the dock */
  unsavedText?: string;

  /** Position of the dock (center, left, or right) */
  position?: DockPosition;

  /** Custom width for the dock */
  width?: string;

  /** Additional className for the dock container */
  className?: string;
}

/**
 * FormSaveDock - A floating dock that appears when a form has unsaved changes
 *
 * This component should be placed inside a Form component and will automatically
 * appear when the form has unsaved changes. It provides save and reset buttons
 * and displays a notification about unsaved changes.
 *
 * Note: Make sure this is wrapped in a FormProvider
 *
 * @example
 * <FormProvider {...form}>
 *   <FormSaveDock />
 * </FormProvider>
 */
export function FormSaveDock({
  saveButtonContent = "Save",
  unsavedText = "Unsaved changes",
  position = "center",
  width = "350px",
  className,
}: FormSaveDockProps) {
  const { control, reset } = useFormContext();
  const { isDirty, dirtyFields, isSubmitting } = useFormState({
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

  console.info("FormSaveDock debug information", {
    isDirty,
    isSubmitting,
    dirtyFields,
  });

  // Position-specific classes
  const positionClasses = {
    center: "left-1/2 transform -translate-x-1/2",
    left: "left-20",
    right: "right-20",
  };

  return isDirty ? (
    <>
      <div
        className={cn(
          "fixed bottom-6 z-50",
          positionClasses[position],
          className,
        )}
        style={{ width }}
      >
        <div className="bg-foreground rounded-lg p-2 shadow-lg flex items-center gap-x-10 w-[380px]">
          <div className="flex items-center gap-x-3">
            <Icon
              icon={faCircleExclamation}
              className="text-amber-400 bg-amber-400/10 dark:text-amber-600 rounded-full"
            />
            <div className="flex flex-col">
              <span className="text-sm font-medium text-background">
                {unsavedText}
              </span>
              <span className="text-2xs text-background/80">
                You have unsaved changes.
              </span>
            </div>
          </div>
          <div className="ml-auto flex items-center space-x-2">
            <Button
              type="reset"
              variant="outline"
              onClick={handleReset}
              disabled={!isDirty || isSubmitting}
              className="bg-white/20 hover:bg-white/30 dark:bg-black/20 dark:hover:bg-black/30 hover:text-background text-background border-none"
            >
              Reset
            </Button>
            <Button type="submit" disabled={!isDirty || isSubmitting}>
              {isSubmitting ? (
                <PulsatingDots size={1} color="white" />
              ) : (
                saveButtonContent
              )}
            </Button>
          </div>
        </div>
      </div>
    </>
  ) : null;
}
