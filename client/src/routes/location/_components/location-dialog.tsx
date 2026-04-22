import { GeocodedBadge } from "@/components/geocode-badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { SplitButton, type SplitButtonOption } from "@/components/ui/split-button";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useCreatePanelActionPreference,
  useEditPanelActionPreference,
  type CreatePanelSaveAction,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { useAuthStore } from "@/stores/auth-store";
import type { DataTablePanelProps } from "@/types/data-table";
import { type Location, type locationSchema } from "@/types/location";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import type { z } from "zod";
import { LocationForm } from "./location-form";
import { LocationGeofenceMap } from "./location-geofence-editor";

type LocationFormInput = z.input<typeof locationSchema>;
type LocationFormReturn = UseFormReturn<LocationFormInput, unknown, Location>;

const QUERY_KEY = "location-list";
const URL = "/locations/" as const;
const TITLE = "Location";
const FORM_ID = "location-dialog-form";

const CREATE_SAVE_OPTIONS: SplitButtonOption<CreatePanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
  { id: "save-add-another", label: "Save & Add Another" },
];

const EDIT_SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & Close" },
];

type LocationDialogProps = DataTablePanelProps<Location> & {
  form: LocationFormReturn;
};

export function LocationDialog({ open, onOpenChange, mode, row, form }: LocationDialogProps) {
  if (mode === "edit") {
    return <EditDialog open={open} onOpenChange={onOpenChange} row={row} form={form} />;
  }
  return <CreateDialog open={open} onOpenChange={onOpenChange} form={form} />;
}

type CreateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: LocationFormReturn;
};

function CreateDialog({ open, onOpenChange, form }: CreateDialogProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useCreatePanelActionPreference();
  const { isPopout, closePopout } = usePopoutWindow();

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open) {
      reset();
    }
  }, [open, reset]);

  const { mutateAsync } = useApiMutation<
    Location,
    { values: Location; action: CreatePanelSaveAction },
    unknown,
    Location
  >({
    mutationFn: async ({ values }) => api.post<Location>(URL, values),
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved.", {
        description: `${TITLE} created successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });

      if (isPopout) {
        closePopout();
        return;
      }

      if (variables.action === "save-close") {
        onOpenChange(false);
        reset();
      } else if (variables.action === "save-add-another") {
        reset();
      }
    },
    setFormError: setError,
    resourceName: TITLE,
  });

  const onSubmit = async (values: Location, action: CreatePanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: CreatePanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  return (
    <DialogShell
      open={open}
      onOpenChange={(next) => {
        if (next) reset();
        onOpenChange(next);
      }}
      titleNode={<DialogTitle>Create new place</DialogTitle>}
      descriptionNode={
        <DialogDescription>
          Build the operating boundary for this location with the map editor on the right.
        </DialogDescription>
      }
      footer={
        <>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              onOpenChange(false);
              reset();
            }}
          >
            Cancel
          </Button>
          <SplitButton
            options={CREATE_SAVE_OPTIONS}
            selectedOption={defaultAction}
            onOptionSelect={handleOptionSelect}
            isLoading={isSubmitting}
            loadingText="Saving..."
            formId={FORM_ID}
          />
        </>
      }
      form={form}
      onValidSubmit={(values) => onSubmit(values, defaultAction)}
    />
  );
}

type EditDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  row: Location | null;
  form: LocationFormReturn;
};

function EditDialog({ open, onOpenChange, row, form }: EditDialogProps) {
  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();
  const { isPopout, closePopout } = usePopoutWindow();
  const user = useAuthStore((s) => s.user);

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open && row) {
      reset(row);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  const { mutateAsync } = useApiMutation<
    Location,
    { values: Location; action: EditPanelSaveAction },
    unknown,
    Location
  >({
    mutationFn: async ({ values }) => api.put<Location>(`${URL}${row?.id as string}/`, values),
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({ queryKey: [QUERY_KEY] });
      const previousRecord = queryClient.getQueryData([QUERY_KEY]);
      queryClient.setQueryData([QUERY_KEY], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success("Changes have been saved", {
        description: `${TITLE} updated successfully`,
      });
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });

      if (isPopout) {
        closePopout();
        return;
      }

      if (variables.action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    setFormError: setError,
    resourceName: TITLE,
  });

  const onSubmit = async (values: Location, action: EditPanelSaveAction) => {
    await mutateAsync({ values, action });
  };

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  const lastUpdatedDescription = row?.updatedAt
    ? `Last updated on ${formatToUserTimezone(
        row.updatedAt as unknown as number,
        { timeFormat: user?.timeFormat || "24-hour" },
        user?.timezone,
      )}`
    : undefined;

  return (
    <DialogShell
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
      titleNode={
        <DialogTitle className="flex items-center justify-start gap-x-1">
          <span className="truncate">{row?.name ?? "Location"}</span>
          {row?.isGeocoded ? (
            <GeocodedBadge
              longitude={row.longitude as unknown as number}
              latitude={row.latitude as unknown as number}
              placeId={row.placeId ?? undefined}
            />
          ) : null}
        </DialogTitle>
      }
      descriptionNode={
        lastUpdatedDescription ? (
          <DialogDescription>{lastUpdatedDescription}</DialogDescription>
        ) : null
      }
      footer={
        <>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              onOpenChange(false);
              reset();
            }}
          >
            Cancel
          </Button>
          <SplitButton
            options={EDIT_SAVE_OPTIONS}
            selectedOption={defaultAction}
            onOptionSelect={handleOptionSelect}
            isLoading={isSubmitting}
            loadingText="Saving..."
            formId={FORM_ID}
          />
        </>
      }
      form={form}
      onValidSubmit={(values) => onSubmit(values, defaultAction)}
    />
  );
}

type DialogShellProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  titleNode: React.ReactNode;
  descriptionNode: React.ReactNode;
  footer: React.ReactNode;
  form: LocationFormReturn;
  onValidSubmit: (values: Location) => void | Promise<void>;
};

function DialogShell({
  open,
  onOpenChange,
  titleNode,
  descriptionNode,
  footer,
  form,
  onValidSubmit,
}: DialogShellProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="grid h-[80vh] max-h-[800px] w-[90vw] grid-cols-1 gap-0 overflow-hidden p-0 sm:max-w-[1100px] lg:grid-cols-[minmax(0,440px)_1fr]">
        <FormProvider {...form}>
          <Form
            id={FORM_ID}
            onSubmit={form.handleSubmit(onValidSubmit)}
            className="flex h-full min-h-0 min-w-0 flex-col border-r"
          >
            <div className="flex shrink-0 flex-col gap-1 border-b bg-background px-5 py-4">
              {titleNode}
              {descriptionNode}
            </div>
            <ScrollArea className="min-h-0 flex-1">
              <LocationForm />
            </ScrollArea>
            <div className="flex shrink-0 items-center justify-end gap-2 border-t border-border bg-muted/30 px-5 py-3">
              {footer}
            </div>
          </Form>
          <div className="relative hidden h-full min-h-0 lg:block">
            <LocationGeofenceMap />
          </div>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
