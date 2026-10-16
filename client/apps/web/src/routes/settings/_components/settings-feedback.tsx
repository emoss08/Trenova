import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

export const SETTINGS_TOASTER_ID = "user-settings";
export const SETTINGS_PREVIEW_TOASTER_ID = "user-settings-preview";

const SAVED_TOAST_ID = "user-settings-saved";
const SAVED_TOAST_MS = 4200;
const MESSAGE_TOAST_MS = 2400;
const SAVED_PULSE_MS = 900;

export type UndoAction = () => void | Promise<void>;

type SettingsFeedback = {
  /** A change was saved; offers Undo when the change can be reversed. */
  saved: (undo?: UndoAction) => void;
  /** A short note in the same place as Saved, for things that are not a setting. */
  message: (text: string) => void;
  /** True for a moment after each save, so the header can flash "Saved". */
  pulsing: boolean;
};

const SettingsFeedbackContext = createContext<SettingsFeedback | null>(null);

export function SettingsFeedbackProvider({ children }: { children: ReactNode }) {
  const [pulsing, setPulsing] = useState(false);
  const pulseTimer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(pulseTimer.current), []);

  const saved = useCallback((undo?: UndoAction) => {
    setPulsing(true);
    window.clearTimeout(pulseTimer.current);
    pulseTimer.current = window.setTimeout(() => setPulsing(false), SAVED_PULSE_MS);
    toast.success(translate("Saved"), {
      id: SAVED_TOAST_ID,
      toasterId: SETTINGS_TOASTER_ID,
      duration: SAVED_TOAST_MS,
      action: undo ? { label: translate("Undo"), onClick: () => void undo() } : undefined,
    });
  }, []);

  const message = useCallback((text: string) => {
    toast(text, {
      id: SAVED_TOAST_ID,
      toasterId: SETTINGS_TOASTER_ID,
      duration: MESSAGE_TOAST_MS,
      action: undefined,
    });
  }, []);

  const value = useMemo(() => ({ saved, message, pulsing }), [saved, message, pulsing]);

  return (
    <SettingsFeedbackContext.Provider value={value}>{children}</SettingsFeedbackContext.Provider>
  );
}

export function useSettingsFeedback(): SettingsFeedback {
  const value = useContext(SettingsFeedbackContext);
  if (!value) {
    throw new Error("useSettingsFeedback must be used inside SettingsFeedbackProvider");
  }
  return value;
}
