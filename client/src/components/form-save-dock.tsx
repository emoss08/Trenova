import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CircleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { Spinner } from "./ui/spinner";

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
      className={cn("fixed bottom-6 z-50", positionClasses[position || "center"], className)}
      style={{ width }}
    >
      <div className="flex w-[420px] items-center gap-x-10 rounded-lg bg-foreground p-2 shadow-lg">
        <div className="flex items-center gap-x-3">
          <CircleAlert className="rounded-full bg-amber-400/10 text-amber-400 dark:text-amber-600" />
          <div className="flex flex-col">
            <span className="text-sm font-medium text-background">{unsavedText}</span>
            <span className="text-2xs text-background/80">You have unsaved changes.</span>
          </div>
        </div>
        <div className="ml-auto flex items-center space-x-2">
          <Button
            type="reset"
            variant="outline"
            onClick={onReset}
            disabled={isSubmitting}
            className="border-none bg-white/20 text-background hover:bg-white/30 hover:text-background dark:bg-black/20 dark:hover:bg-black/30"
          >
            Reset
          </Button>
          <Button type="submit" variant="default" className="pr-2" disabled={isSubmitting}>
            {isSubmitting ? <Spinner /> : (saveButtonContent ?? "Save")}
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

  if (!isDirty || Object.keys(dirtyFields).length === 0) {
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
