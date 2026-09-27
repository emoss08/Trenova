import { FormSaveDock } from "@/components/form-save-dock";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { documentControlSchema, type DocumentControl } from "@/types/document-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";

/**
 * The organization's capture settings. They live on the document settings, as
 * the rest of how documents are handled does, so saving here saves that record.
 */
export default function CaptureSettingsForm() {
  const t = useT();
  const { data } = useSuspenseQuery({ ...queries.documentControl.get() });

  const form = useForm<DocumentControl>({
    resolver: zodResolver(documentControlSchema),
    defaultValues: data,
  });
  const { control, handleSubmit, reset } = form;
  const enabled = useWatch({ control, name: "enableCapture" });

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentControl.get._def,
    mutationFn: async (values: DocumentControl) => apiService.documentControlService.update(values),
    resourceName: "Capture settings",
    resetForm: reset,
    form,
    invalidateQueries: [queries.documentControl.get._def, queries.capture.access._def],
  });

  const onSubmit = useCallback(
    async (values: DocumentControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <Card>
            <CardHeader>
              <CardTitle>{t("Availability")}</CardTitle>
              <CardDescription>
                {t(
                  "Whether people can scan and print into Trenova with Trenova Capture. Turning it off stops every paired computer until it is turned on again; nothing is unpaired.",
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className="max-w-prose">
              <FormGroup cols={1}>
                <FormControl className="min-h-[3em]">
                  <SwitchField
                    control={control}
                    name="enableCapture"
                    label={t("Turn on scanning and printing into Trenova")}
                    position="left"
                  />
                </FormControl>
              </FormGroup>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("Filing")}</CardTitle>
              <CardDescription>
                {t("What happens to a stack once it is read, and how long unfiled pages are kept.")}
              </CardDescription>
            </CardHeader>
            <CardContent className="max-w-prose">
              <FormGroup cols={1}>
                <FormControl className="min-h-[3em]">
                  <SwitchField
                    control={control}
                    name="captureAutoFileCoverSheets"
                    label={t("File documents behind a cover sheet on their own")}
                    description={t(
                      "A document that follows a Trenova cover sheet is filed onto the sheet's record without waiting in Intake. Anything else still waits for a person.",
                    )}
                    position="left"
                    disabled={!enabled}
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name="captureRetentionDays"
                    label={t("Keep unfiled pages for (days)")}
                    description={t(
                      "Between 1 and 365. Pages not filed by then are deleted, and their owner is reminded a week before.",
                    )}
                  />
                </FormControl>
              </FormGroup>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("Trenova Capture versions")}</CardTitle>
              <CardDescription>
                {t("Which versions of the companion may connect, and whether it updates itself.")}
              </CardDescription>
            </CardHeader>
            <CardContent className="max-w-prose">
              <FormGroup cols={1}>
                <FormControl>
                  <InputField
                    control={control}
                    name="captureMinAgentVersion"
                    label={t("Oldest version allowed")}
                    placeholder="1.0.0"
                    description={t(
                      "A computer running anything older is asked to update before it can scan. Leave blank to allow any version.",
                    )}
                  />
                </FormControl>
                <FormControl className="min-h-[3em]">
                  <SwitchField
                    control={control}
                    name="captureAllowAutoUpdate"
                    label={t("Let Trenova Capture update itself")}
                    description={t(
                      "Installs new versions when they are released. Turn off where software is rolled out by IT.",
                    )}
                    position="left"
                  />
                </FormControl>
              </FormGroup>
            </CardContent>
          </Card>

          <FormSaveDock saveButtonContent={t("Save changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}
