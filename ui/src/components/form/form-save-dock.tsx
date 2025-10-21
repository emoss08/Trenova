import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import { faCircleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { Kbd } from "../ui/kbd";
import { PulsatingDots } from "../ui/pulsating-dots";

type DockPosition = "center" | "left" | "right";

interface FormSaveDockProps {
  saveButtonContent?: ReactNode;
  unsavedText?: string;
  position?: DockPosition;
  width?: string;
  className?: string;
}

function SaveDockContent({
  saveButtonContent,
  unsavedText,
  position,
  width,
  className,
  isSubmitting,
  onReset,
}: FormSaveDockProps & {
  isSubmitting: boolean;
  onReset: () => void;
}) {
  const positionClasses = {
    center: "left-1/2 transform -translate-x-1/2",
    left: "left-20",
    right: "right-20",
  };

  return (
    <div
      className={cn(
        "fixed bottom-6 z-50",
        positionClasses[position || "center"],
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
            onClick={onReset}
            disabled={isSubmitting}
            className="bg-white/20 hover:bg-white/30 dark:bg-black/20 dark:hover:bg-black/30 hover:text-background text-background border-none"
          >
            Reset
          </Button>
          <Button
            type="submit"
            variant="background"
            className="pr-2"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <PulsatingDots size={1} color="white" />
            ) : (
              <>
                {saveButtonContent && (
                  <>
                    {saveButtonContent}
                    <Kbd className="shrink-0">⏎</Kbd>
                  </>
                )}
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

export function FormSaveDock({
  saveButtonContent = "Save",
  unsavedText = "Unsaved changes",
  position = "center",
  width = "350px",
  className,
}: FormSaveDockProps) {
  const { control, reset } = useFormContext();
  const [isVisible, setIsVisible] = useState(false);
  const visibilityTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );

  const { isDirty, dirtyFields, isSubmitting } = useFormState({
    control,
  });

  useEffect(() => {
    if (visibilityTimerRef.current) {
      clearTimeout(visibilityTimerRef.current);
    }

    visibilityTimerRef.current = setTimeout(() => {
      if (isDirty && !isVisible) {
        setIsVisible(true);
      } else if (!isDirty && isVisible) {
        setIsVisible(false);
      }
    }, 100);

    return () => {
      if (visibilityTimerRef.current) {
        clearTimeout(visibilityTimerRef.current);
      }
    };
  }, [isDirty, isVisible]);

  const handleReset = useCallback(() => {
    reset(
      {},
      {
        keepDirty: false,
        keepValues: true,
      },
    );
  }, [reset]);

  if (!isVisible || Object.keys(dirtyFields).length === 0) {
    return null;
  }

  return (
    <SaveDockContent
      saveButtonContent={saveButtonContent}
      unsavedText={unsavedText}
      position={position}
      width={width}
      className={className}
      isSubmitting={isSubmitting}
      onReset={handleReset}
    />
  );
}
