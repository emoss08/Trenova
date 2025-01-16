import { useCallback, useEffect, useState } from "react";

export function useUnsavedChanges({
  isDirty,
  onClose,
}: {
  isDirty: boolean;
  onClose: () => void;
}) {
  const [showWarning, setShowWarning] = useState(false);

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (isDirty) {
        event.preventDefault();
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
