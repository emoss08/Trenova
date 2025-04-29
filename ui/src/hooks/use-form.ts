import { useCallback, useEffect, useState } from "react";
import { useFormState, type Control, type FieldValues } from "react-hook-form";

export function useUnsavedChanges<TFieldValues extends FieldValues>({
  control,
  onClose,
}: {
  control: Control<TFieldValues>;
  onClose: () => void;
}) {
  const { isDirty } = useFormState({ control });

  const [showWarning, setShowWarning] = useState(false);

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (isDirty) {
        // For modern browsers
        event.preventDefault();
        // For older browsers
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  const handleClose = useCallback(() => {
    if (isDirty) {
      setShowWarning(true);
    } else {
      onClose();
    }
  }, [isDirty, onClose]);

  const handleConfirmClose = useCallback(() => {
    setShowWarning(false);
    onClose();
  }, [onClose]);

  const handleCancelClose = useCallback(() => {
    setShowWarning(false);
  }, []);

  return {
    showWarning,
    handleClose,
    handleConfirmClose,
    handleCancelClose,
  };
}
