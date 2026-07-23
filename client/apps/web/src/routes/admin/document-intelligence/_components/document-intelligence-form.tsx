import { FormSaveDock } from "@/components/form-save-dock";
import { SwitchField } from "@/components/fields/switch-field";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  documentControlResourceSchema,
  documentControlSchema,
  type DocumentControl,
  type DocumentControlResource,
} from "@/types/document-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";

const allowedResourceOptions: Array<{
  value: DocumentControlResource;
  label: string;
  description: string;
}> = [
  {
    value: documentControlResourceSchema.enum.shipment,
    label: "Shipment",
    description:
      "Allow shipment drafts to be generated for shipment documents.",
  },
  {
    value: documentControlResourceSchema.enum.trailer,
    label: "Trailer",
    description:
      "Allow shipment-draft style extraction on trailer-linked documents.",
  },
  {
    value: documentControlResourceSchema.enum.tractor,
    label: "Tractor",
    description:
      "Allow shipment-draft style extraction on tractor-linked documents.",
  },
  {
    value: documentControlResourceSchema.enum.worker,
    label: "Worker",
    description:
      "Allow shipment-draft style extraction on worker-linked documents.",
  },
];

export default function DocumentIntelligenceForm() {
  const { data } = useSuspenseQuery({
    ...queries.documentControl.get(),
  });

  const form = useForm<DocumentControl>({
    resolver: zodResolver(documentControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, reset, setError } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentControl.get._def,
    mutationFn: async (values: DocumentControl) =>
      apiService.documentControlService.update(values),
    resourceName: "Document Intelligence",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.documentControl.get._def],
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
          <PlatformAvailabilityCard />
          <ClassificationAndExtractionCard />
          <ShipmentDraftCard />
          <SearchCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function PlatformAvailabilityCard() {
  const { control } = useFormContext<DocumentControl>();
  const enabled = useWatch({
    control,
    name: "enableDocumentIntelligence",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Platform Availability</CardTitle>
        <CardDescription>
          Control whether document intelligence is active for this tenant. When
          this is disabled, extraction and shipment-draft workflows remain off
          even if the OpenAI integration is configured.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableDocumentIntelligence"
              label="Enable Document Intelligence"
              description="Master switch for OCR, document classification, extraction, and indexing workflows."
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em] pl-10">
            <SwitchField
              control={control}
              name="enableOcr"
              label="Enable OCR"
              description="Run OCR when native text extraction is unavailable or insufficient."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ClassificationAndExtractionCard() {
  const { control } = useFormContext<DocumentControl>();
  const enabled = useWatch({
    control,
    name: "enableDocumentIntelligence",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Classification And Extraction</CardTitle>
        <CardDescription>
          Manage automatic routing, document type assignment, and optional
          AI-assisted extraction. AI toggles here depend on a configured and
          enabled OpenAI integration.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAutoClassification"
              label="Enable Automatic Classification"
              description="Classify uploaded documents into supported kinds as they are processed."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
          <FormControl className="min-h-[3em] pl-10">
            <SwitchField
              control={control}
              name="enableAiAssistedClassification"
              label="Enable AI-Assisted Classification"
              description="Use the OpenAI integration to improve document-kind routing when deterministic classification is insufficient."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAutoDocumentTypeAssociate"
              label="Enable Automatic Document Type Association"
              description="Map detected document kinds to existing document types automatically."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
          <FormControl className="min-h-[3em] pl-10">
            <SwitchField
              control={control}
              name="enableAutoCreateDocumentTypes"
              label="Enable Automatic Document Type Creation"
              description="Create missing document types during auto-association when a mapping does not exist yet."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableAiAssistedExtraction"
              label="Enable AI-Assisted Extraction"
              description="Use the OpenAI integration for structured extraction on supported document kinds."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ShipmentDraftCard() {
  const { control, setValue } = useFormContext<DocumentControl>();
  const enabled = useWatch({
    control,
    name: "enableDocumentIntelligence",
  });
  const shipmentDraftEnabled = useWatch({
    control,
    name: "enableShipmentDraftExtraction",
  });
  const selectedResources = useWatch({
    control,
    name: "shipmentDraftAllowedResources",
  });

  const toggleResource = useCallback(
    (resource: DocumentControlResource, checked: boolean) => {
      const current = selectedResources ?? [];
      const next = checked
        ? Array.from(new Set([...current, resource]))
        : current.filter((value) => value !== resource);

      setValue("shipmentDraftAllowedResources", next, {
        shouldDirty: true,
        shouldValidate: true,
      });
    },
    [selectedResources, setValue],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Shipment Draft Extraction</CardTitle>
        <CardDescription>
          Limit structured shipment-draft generation to the resources where
          operators should be able to review a draft and create a shipment from
          it.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableShipmentDraftExtraction"
              label="Enable Shipment Draft Extraction"
              description="Generate reviewable shipment drafts for supported document kinds such as rate confirmations."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
          <div className="grid gap-2 pl-10">
            {allowedResourceOptions.map((option) => {
              const checked = (selectedResources ?? []).includes(option.value);

              return (
                <div
                  key={option.value}
                  className="flex items-start gap-3 rounded-md border border-border/80 p-3"
                >
                  <Checkbox
                    checked={checked}
                    disabled={!enabled || !shipmentDraftEnabled}
                    onCheckedChange={(value) =>
                      toggleResource(option.value, value === true)
                    }
                    className="mt-0.5"
                  />
                  <div className="grid gap-1">
                    <Label>{option.label}</Label>
                    <p className="text-2xs text-muted-foreground">
                      {option.description}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function SearchCard() {
  const { control } = useFormContext<DocumentControl>();
  const enabled = useWatch({
    control,
    name: "enableDocumentIntelligence",
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Search And Retrieval</CardTitle>
        <CardDescription>
          Control whether extracted text is indexed for document search and
          retrieval experiences.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="enableFullTextIndexing"
              label="Enable Full-Text Indexing"
              description="Store extracted text in the search index so operators can find documents by content."
              position="left"
              disabled={!enabled}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
