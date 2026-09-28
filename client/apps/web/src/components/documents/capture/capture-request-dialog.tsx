import {
  CaptureDeviceAutocompleteField,
  CaptureProfileAutocompleteField,
  DocumentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import { SelectField } from "@/components/fields/select-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { CaptureRecordKind } from "@/lib/capture";
import {
  captureRequestFormSchema,
  emptyToNull,
  type CaptureRequestFormValues,
} from "@/lib/capture-forms";
import {
  createCaptureRequest,
  type CaptureDevice,
  type CaptureRequestMode,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

/** The scanner picker's value for "whichever scanner the computer defaults to". */
const DEFAULT_SCANNER = "__default__";

const DEFAULT_VALUES: CaptureRequestFormValues = {
  deviceId: "",
  sourceName: DEFAULT_SCANNER,
  profileId: "",
  documentTypeId: "",
};

/** The computer to offer first: one that is connected, else the first. */
function preferredDevice(devices: CaptureDevice[]): CaptureDevice | undefined {
  return devices.find((device) => device.isOnline) ?? devices[0];
}

/**
 * Asks one of the person's own computers to scan into this record, or to catch
 * the next thing they print. A request only ever reaches the person's own
 * computers: starting a scanner somebody else is standing at is not something
 * this can do.
 */
export function CaptureRequestDialog({
  open,
  onOpenChange,
  mode,
  kind,
  recordId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: CaptureRequestMode;
  kind: CaptureRecordKind;
  recordId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const devicesQuery = useQuery({ ...queries.capture.myDevices("Active"), enabled: open });
  const devices = devicesQuery.data ?? [];

  const form = useForm<CaptureRequestFormValues>({
    resolver: zodResolver(captureRequestFormSchema) as Resolver<CaptureRequestFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset, setValue, getValues } = form;
  const deviceId = useWatch({ control, name: "deviceId" });
  const chosen = devices.find((device) => device.id === deviceId);

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  // Start on a connected computer once the person's computers are known.
  useEffect(() => {
    const preferred = preferredDevice(devicesQuery.data ?? []);
    if (open && preferred !== undefined && getValues("deviceId") === "") {
      setValue("deviceId", preferred.id);
    }
  }, [open, devicesQuery.data, getValues, setValue]);

  // A scanner belongs to one computer; choosing another starts from its default.
  useEffect(() => {
    setValue("sourceName", DEFAULT_SCANNER);
  }, [deviceId, setValue]);

  const request = useApiMutation<
    Awaited<ReturnType<typeof createCaptureRequest>>,
    CaptureRequestFormValues,
    unknown,
    CaptureRequestFormValues
  >({
    form,
    resourceName: "Capture request",
    mutationFn: (values) =>
      createCaptureRequest({
        deviceId: values.deviceId,
        mode,
        targetType: kind,
        targetId: recordId,
        documentTypeId: emptyToNull(values.documentTypeId),
        profileId: mode === "Scan" ? emptyToNull(values.profileId) : null,
        sourceName:
          mode === "Scan" && values.sourceName !== DEFAULT_SCANNER ? values.sourceName : null,
      }),
    onSuccess: async () => {
      const name = chosen?.name ?? "";
      toast.success(
        mode === "Scan"
          ? t("Sent to {0}. Put the pages in the scanner.", name)
          : t("Print to Trenova from any program on {0} in the next ten minutes.", name),
      );
      onOpenChange(false);
      await queryClient.invalidateQueries({
        queryKey: queries.capture.requests(kind, recordId).queryKey,
      });
    },
  });

  const title = mode === "Scan" ? t("Scan into this record") : t("Print into this record");
  const description =
    mode === "Scan"
      ? t(
          "The scan starts on the computer you choose, and its pages are filed onto this record as they arrive. Anything Trenova cannot place waits in Intake.",
        )
      : t(
          "The next document you print to the Trenova printer on the computer you choose is filed here.",
        );

  const scannerOptions = [
    { value: DEFAULT_SCANNER, label: t("The computer's default scanner") },
    ...(chosen?.sources ?? []).map((source) => ({ value: source.name, label: source.name })),
  ];

  const body = () => {
    if (devicesQuery.isPending) {
      return (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </div>
      );
    }
    if (devicesQuery.isError) {
      return (
        <Alert variant="destructive" size="sm">
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            <span>{t("Your computers could not be loaded.")}</span>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={() => void devicesQuery.refetch()}
              isLoading={devicesQuery.isRefetching}
            >
              {t("Try again")}
            </Button>
          </AlertDescription>
        </Alert>
      );
    }
    if (devices.length === 0) {
      return (
        <Alert variant="info" size="sm">
          <AlertDescription>
            {t(
              "No computer is set up to scan for you yet. Install Trenova Capture, sign in from its tray icon, and approve the code it shows.",
            )}{" "}
            <Link to="/capture/devices" className="ui-focus-ring text-brand hover:underline">
              {t("My scanners")}
            </Link>
          </AlertDescription>
        </Alert>
      );
    }
    return (
      <FormGroup cols={1}>
        <FormControl>
          <CaptureDeviceAutocompleteField<CaptureRequestFormValues>
            control={control}
            name="deviceId"
            label={t("Computer")}
            placeholder={t("Choose a computer")}
            rules={{ required: true }}
            clearable={false}
          />
        </FormControl>
        {chosen !== undefined && !chosen.isOnline && (
          <Alert variant="warning" size="sm">
            <AlertDescription>
              {t(
                "{0} is not connected right now. The request waits a few minutes for it to come online.",
                chosen.name,
              )}
            </AlertDescription>
          </Alert>
        )}
        {mode === "Scan" && (
          <>
            <FormControl>
              <SelectField<CaptureRequestFormValues>
                control={control}
                name="sourceName"
                label={t("Scanner")}
                options={scannerOptions}
                description={
                  chosen !== undefined && (chosen.sources ?? []).length === 0
                    ? t("{0} has not reported its scanners yet.", chosen.name)
                    : undefined
                }
              />
            </FormControl>
            <FormControl>
              <CaptureProfileAutocompleteField<CaptureRequestFormValues>
                control={control}
                name="profileId"
                label={t("Scan settings")}
                placeholder={t("Organization default")}
              />
            </FormControl>
          </>
        )}
        <FormControl>
          <DocumentTypeAutocompleteField<CaptureRequestFormValues>
            control={control}
            name="documentTypeId"
            label={t("Document type")}
            placeholder={t("Optional")}
          />
        </FormControl>
      </FormGroup>
    );
  };

  const ready = devicesQuery.isSuccess && devices.length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => request.mutateAsync(values))(submitEvent);
            }}
          >
            {body()}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                disabled={!ready}
                isLoading={request.isPending}
                loadingText={t("Sending")}
              >
                {mode === "Scan" ? t("Start scan") : t("Wait for my print")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
