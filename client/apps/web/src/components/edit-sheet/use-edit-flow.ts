import { hotkeyOf } from "@/config/keybinds.config";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useHotkey } from "@tanstack/react-hotkeys";
import { editConflictOf } from "@trenova/shared/lib/edit-conflict";
import type { EditConflict } from "@trenova/shared/types/errors";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  type FieldPath,
  type FieldValues,
  type PathValue,
  type UseFormReturn,
  useFormState,
} from "react-hook-form";

/** How long "Saved" stays in the save bar after a save. */
const SAVED_NOTICE_MS = 1800;

export type EditFlowOptions<T extends FieldValues> = {
  form: UseFormReturn<T>;
  /**
   * Saves the values and returns them as saved, which the editor resets to: the server
   * moves the version on, and the next save must carry the new one.
   */
  onSave: (values: T) => Promise<T>;
  onClose: () => void;
  /** A new record: saving creates it and closes, and there is nothing to discard. */
  create?: boolean;
  /** Why the editor cannot be saved as it stands, shown in the save bar. */
  invalid?: string | null;
  resourceName?: string;
  /** Reads the record as the other person saved it, for "Load theirs". */
  loadLatest?: () => Promise<T>;
  /** The field holding the record's version, which "Keep mine" moves to theirs. */
  versionField?: FieldPath<T>;
  /** Whether the editor's keys are live: false while another editor sits above it. */
  enabled?: boolean;
  /** Bind Escape to closing. A sheet closes itself; a full-screen editor asks for this. */
  bindEscape?: boolean;
};

export type EditFlow = {
  /** The top-level fields that differ from what was loaded, in the form's order. */
  changed: string[];
  dirty: boolean;
  canSave: boolean;
  saving: boolean;
  saved: boolean;
  create: boolean;
  invalid: string | null;
  confirmingClose: boolean;
  reviewing: boolean;
  conflict: EditConflict | null;
  save: () => void;
  /** Closes, asking first when there are unsaved changes. */
  tryClose: () => void;
  close: () => void;
  keepEditing: () => void;
  toggleReview: () => void;
  setReviewing: (open: boolean) => void;
  discard: () => void;
  undo: (field: string) => void;
  loadTheirs: () => Promise<void>;
  keepMine: () => void;
  /** Backs out one step: the close question, then the review, then the editor itself. */
  back: () => void;
};

/**
 * Save and close behaviour shared by every editor, whether a side sheet or a full-screen
 * builder: what changed, saving with Mod+S, asking before discarding, reviewing and undoing
 * a change, and settling a save that lost a race with another person's.
 */
export function useEditFlow<T extends FieldValues>({
  form,
  onSave,
  onClose,
  create = false,
  invalid = null,
  resourceName,
  loadLatest,
  versionField = "version" as FieldPath<T>,
  enabled = true,
  bindEscape = false,
}: EditFlowOptions<T>): EditFlow {
  const { dirtyFields, isDirty } = useFormState({ control: form.control });
  const [confirmingClose, setConfirmingClose] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [conflict, setConflict] = useState<EditConflict | null>(null);
  const savedTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (savedTimer.current) {
        clearTimeout(savedTimer.current);
      }
    },
    [],
  );

  // dirtyFields is the form's own object, changed in place, so it is read on every render
  // rather than memoized on its identity.
  const changed = Object.keys(dirtyFields).filter((key) => key !== versionField);
  const dirty = isDirty && changed.length > 0;
  const canSave = (dirty || create) && !invalid && !saving;

  const close = useCallback(() => {
    setConfirmingClose(false);
    setReviewing(false);
    setConflict(null);
    onClose();
  }, [onClose]);

  const tryClose = useCallback(() => {
    if (dirty) {
      setConfirmingClose(true);
      return;
    }
    close();
  }, [close, dirty]);

  const submit = useCallback(
    (values: T) => {
      setSaving(true);
      setReviewing(false);
      return onSave(values)
        .then((result) => {
          form.reset(result);
          setConflict(null);
          setSaved(true);
          if (savedTimer.current) {
            clearTimeout(savedTimer.current);
          }
          savedTimer.current = setTimeout(() => setSaved(false), SAVED_NOTICE_MS);
          if (create) {
            close();
          }
        })
        .catch((error: unknown) => {
          const lost = editConflictOf(error);
          if (lost) {
            setConflict(lost);
            return;
          }
          handleMutationError({ error, form, resourceName });
        })
        .finally(() => setSaving(false));
    },
    [close, create, form, onSave, resourceName],
  );

  const save = useCallback(() => {
    if (!canSave) {
      return;
    }
    void form.handleSubmit(submit)();
  }, [canSave, form, submit]);

  const discard = useCallback(() => {
    form.reset();
    setReviewing(false);
  }, [form]);

  const undo = useCallback(
    (field: string) => {
      form.resetField(field as FieldPath<T>);
    },
    [form],
  );

  const loadTheirs = useCallback(async () => {
    if (!loadLatest) {
      return;
    }
    const latest = await loadLatest();
    form.reset(latest);
    setConflict(null);
  }, [form, loadLatest]);

  const keepMine = useCallback(() => {
    if (!conflict) {
      return;
    }
    form.setValue(versionField, conflict.version as PathValue<T, FieldPath<T>>);
    setConflict(null);
    void form.handleSubmit(submit)();
  }, [conflict, form, submit, versionField]);

  const back = useCallback(() => {
    if (confirmingClose) {
      setConfirmingClose(false);
      return;
    }
    if (reviewing) {
      setReviewing(false);
      return;
    }
    tryClose();
  }, [confirmingClose, reviewing, tryClose]);

  useHotkey(hotkeyOf("edit-sheet", "save"), save, {
    enabled,
    ignoreInputs: false,
    preventDefault: true,
  });
  useHotkey(hotkeyOf("edit-sheet", "close"), back, {
    enabled: enabled && bindEscape,
    ignoreInputs: false,
    preventDefault: false,
  });

  return {
    changed,
    dirty,
    canSave,
    saving,
    saved,
    create,
    invalid,
    confirmingClose,
    reviewing,
    conflict,
    save,
    tryClose,
    close,
    keepEditing: () => setConfirmingClose(false),
    toggleReview: () => setReviewing((open) => !open),
    setReviewing,
    discard,
    undo,
    loadTheirs,
    keepMine,
    back,
  };
}
