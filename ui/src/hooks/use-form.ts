import { useCallback, useEffect, useRef, useState } from "react";

export function useUnsavedChanges({
  isDirty,
  onClose,
}: {
  isDirty: boolean;
  onClose: () => void;
}) {
  const [showWarning, setShowWarning] = useState(false);
  const isDirtyRef = useRef(isDirty);

  // Keep ref updated with latest isDirty value for use in event handlers
  useEffect(() => {
    isDirtyRef.current = isDirty;
  }, [isDirty]);

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (isDirtyRef.current) {
        // For modern browsers
        event.preventDefault();
        event.returnValue = "";
        // For older browsers
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, []);

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
