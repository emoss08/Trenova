import type { SeatedForm } from "@/hooks/use-seated-form";
import type { FieldValues } from "react-hook-form";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Suspense, createContext, lazy, useContext, useState, type ReactNode } from "react";
import type { ComparedContent } from "./newer-version-dialog";
import { useOnceOpened } from "./studio-rail-tabs";

/*
 * What an edit page shows while it holds a newer saved version back from a form with edits:
 * the notice in the Studio's source section, and the comparison the notice and a save open.
 * The comparison's diff is heavy, so its module is read the first time it opens.
 */
const NewerVersionDialog = lazy(() =>
  import("./newer-version-dialog").then((module) => ({ default: module.NewerVersionDialog })),
);

type ComparableValues = {
  name?: string | null;
  description?: string | null;
  source?: string | null;
};

function compared(values: ComparableValues): ComparedContent {
  return {
    name: values.name ?? "",
    description: values.description ?? "",
    source: values.source ?? "",
  };
}

type NewerVersionNoticeValue = {
  /** Opens the comparison of the person's edits with the newer saved version. */
  onCompare: () => void;
};

const NewerVersionNoticeContext = createContext<NewerVersionNoticeValue | null>(null);

/** Says the record was saved again after the edits began, and offers the comparison. */
export function NewerVersionNotice() {
  const t = useT();
  const notice = useContext(NewerVersionNoticeContext);
  if (!notice) return null;
  return (
    <Alert variant="warning" size="sm" role="status">
      <AlertCircleIcon />
      <AlertDescription>
        {t(
          "This was saved again after you started editing. Compare it with your edits before you save.",
        )}
      </AlertDescription>
      <AlertAction>
        <Button type="button" variant="ghost" size="xs" onClick={notice.onCompare}>
          {t("Compare")}
        </Button>
      </AlertAction>
    </Alert>
  );
}

export type NewerVersionCompare = {
  /** Opens the comparison with the form's values as they stand. */
  open: (values: ComparableValues) => void;
  dialog: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    mine: ComparedContent | null;
  };
};

/** The comparison's state, held by the edit page so a save can open it too. */
export function useNewerVersionCompare(): NewerVersionCompare {
  const [open, setOpen] = useState(false);
  const [mine, setMine] = useState<ComparedContent | null>(null);
  return {
    open: (values) => {
      setMine(compared(values));
      setOpen(true);
    },
    dialog: { open, onOpenChange: setOpen, mine },
  };
}

type NewerVersionProviderProps<T extends FieldValues & ComparableValues> = {
  seat: SeatedForm<T>;
  compare: NewerVersionCompare;
  /** Reads the form's values when the notice opens the comparison. */
  values: () => ComparableValues;
  /** The newer saved version's number, when the record has one. */
  savedVersionNumber: number | null;
  children: ReactNode;
};

/**
 * Gives the Studio below it the notice while `seat` holds a newer version, and draws the
 * comparison with its two ways out: load the saved version, or keep the edits over it.
 */
export function NewerVersionProvider<T extends FieldValues & ComparableValues>({
  seat,
  compare,
  values,
  savedVersionNumber,
  children,
}: NewerVersionProviderProps<T>) {
  const { open, onOpenChange, mine } = compare.dialog;
  const opened = useOnceOpened(open);
  const held = seat.held;
  return (
    <NewerVersionNoticeContext.Provider
      value={held ? { onCompare: () => compare.open(values()) } : null}
    >
      {children}
      {opened && held && mine && (
        <Suspense fallback={null}>
          <NewerVersionDialog
            open={open}
            onOpenChange={onOpenChange}
            saved={compared(held.values)}
            mine={mine}
            savedVersionNumber={savedVersionNumber}
            onLoadSaved={() => {
              onOpenChange(false);
              seat.seat(held);
            }}
            onKeepEdits={() => {
              onOpenChange(false);
              seat.keepEdits();
            }}
          />
        </Suspense>
      )}
    </NewerVersionNoticeContext.Provider>
  );
}
