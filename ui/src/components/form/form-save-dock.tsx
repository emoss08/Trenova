// src/components/form/form-save-dock.tsx
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import {
  faCircleExclamation,
  faClock,
} from "@fortawesome/pro-regular-svg-icons";
import { ReactNode } from "react";
import { PulsatingDots } from "../ui/pulsating-dots";
import { useFormSave } from "./form-save-context";

type DockPosition = "center" | "left" | "right";

interface FormSaveDockProps {
  /** Whether the form has unsaved changes */
  isDirty: boolean;

  /** Whether the form is currently submitting */
  isSubmitting: boolean;

  /** Function to call when the reset button is clicked */
  // onReset: () => void;

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
 */
export function FormSaveDock({
  isDirty,
  isSubmitting,
  // onReset,
  saveButtonContent = "Save",
  unsavedText = "Unsaved changes",
  position = "center",
  width = "350px",
  className,
}: FormSaveDockProps) {
  // Only render the dock if there are unsaved changes
  if (!isDirty) return null;

  // Position-specific classes
  const positionClasses = {
    center: "left-1/2 transform -translate-x-1/2",
    left: "left-20",
    right: "right-20",
  };

  return (
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
              // onClick={onReset}
              disabled={!isDirty || isSubmitting}
              className="bg-white/20 hover:bg-white/30 dark:bg-black/20 dark:hover:bg-black/30 hover:text-background text-background border-none"
            >
              Reset
            </Button>
            <Button
              type="submit"
              disabled={!isDirty || isSubmitting}
              className="bg-background text-foreground hover:bg-background/80 min-w-14"
            >
              {isSubmitting ? (
                <PulsatingDots size={1} color="foreground" />
              ) : (
                saveButtonContent
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Add space at the bottom to prevent content from being covered */}
      {/* <div className="h-16" /> */}
    </>
  );
}

/**
 * LastSavedIndicator - Shows when the form was last saved
 *
 * This component should be placed wherever you want to display the last saved timestamp.
 */
export function LastSavedIndicator() {
  const { lastSaved } = useFormSave();

  if (!lastSaved) return null;

  return (
    <div className="text-sm text-muted-foreground flex items-center">
      <Icon icon={faClock} className="mr-1" />
      Last saved: {lastSaved}
    </div>
  );
}
