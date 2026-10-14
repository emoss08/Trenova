import { Button } from "@trenova/shared/components/ui/button";
import { CheckIcon, ReverseLeftIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
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
import { toast } from "sonner";

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

function InkToast({ children }: { children: ReactNode }) {
  return (
    <div
      role="status"
      className="bg-ink text-ink-foreground ui-lift flex h-10 items-center gap-3 rounded-lg pr-1.5 pl-3.5 text-sm"
    >
      {children}
    </div>
  );
}

function SavedToast({ toastId, onUndo }: { toastId: string | number; onUndo?: UndoAction }) {
  const t = useT();

  return (
    <InkToast>
      <span className="flex items-center gap-1.5">
        <CheckIcon className="size-3.5" />
        {t("Saved")}
      </span>
      {onUndo ? (
        <Button
          type="button"
          variant="bare"
          size="xs"
          className="hover:bg-ink-hover text-ink-foreground rounded-md px-2.5 opacity-80 hover:opacity-100"
          onClick={() => {
            toast.dismiss(toastId);
            void onUndo();
          }}
        >
          <ReverseLeftIcon className="size-3" />
          {t("Undo")}
        </Button>
      ) : null}
    </InkToast>
  );
}

export function SettingsFeedbackProvider({ children }: { children: ReactNode }) {
  const [pulsing, setPulsing] = useState(false);
  const pulseTimer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(pulseTimer.current), []);

  const saved = useCallback((undo?: UndoAction) => {
    setPulsing(true);
    window.clearTimeout(pulseTimer.current);
    pulseTimer.current = window.setTimeout(() => setPulsing(false), SAVED_PULSE_MS);
    toast.custom((id) => <SavedToast toastId={id} onUndo={undo} />, {
      id: SAVED_TOAST_ID,
      toasterId: SETTINGS_TOASTER_ID,
      duration: SAVED_TOAST_MS,
    });
  }, []);

  const message = useCallback((text: string) => {
    toast.custom(() => <InkToast>{text}</InkToast>, {
      id: SAVED_TOAST_ID,
      toasterId: SETTINGS_TOASTER_ID,
      duration: MESSAGE_TOAST_MS,
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
