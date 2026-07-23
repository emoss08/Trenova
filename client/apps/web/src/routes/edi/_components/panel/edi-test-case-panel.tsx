import { EDIDocumentProfileAutocompleteField } from "@/components/autocomplete-fields";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { EDITestCaseVerdictBadge } from "@trenova/shared/components/status-badge";
import { InputField } from "@/components/fields/input-field";
import { JsonEditorField } from "@/components/fields/json-editor-field";
import { NumberField } from "@/components/fields/number-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { EDIDocumentPreview, EDITestCaseRow } from "@trenova/shared/types/edi";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlayIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { useEditorTheme } from "../designer/components/designer-shared";
import type { InspectorTab } from "../designer/inspector/components/inspector-tabs";
import PreviewInspectorSheet from "../designer/inspector/preview-inspector-sheet";
import {
  ediTestCaseFormSchema,
  getTestCaseFormDefaults,
  toTestCaseRequest,
  type EDITestCaseFormValues,
} from "../edi-schemas";
import { invalidateEDITestCases } from "./edi-panel-invalidation";

export function TestCasePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDITestCaseRow>) {
  if (mode === "create") {
    return <CreateTestCasePanel open={open} onOpenChange={onOpenChange} />;
  }

  return <TestCaseEditPanel open={open} onOpenChange={onOpenChange} testCaseId={row?.id ?? null} />;
}

function CreateTestCasePanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const form = useForm<EDITestCaseFormValues>({
    resolver: zodResolver(ediTestCaseFormSchema),
    defaultValues: getTestCaseFormDefaults(),
    mode: "onChange",
  });

  const mutation = useApiMutation({
    mutationFn: (values: EDITestCaseFormValues) =>
      apiService.ediService.createTestCase(toTestCaseRequest(values)),
    setFormError: form.setError,
    resourceName: "EDI Test Case",
    onSuccess: async () => {
      toast.success("EDI test case created");
      form.reset(getTestCaseFormDefaults());
      onOpenChange(false);
      await invalidateEDITestCases(queryClient);
    },
  });

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      form.reset(getTestCaseFormDefaults());
    }
    onOpenChange(nextOpen);
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="New EDI Test Case"
      description="Bind a document profile to a payload and expected validation outcome for partner certification."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="edi-create-test-case-form" isLoading={mutation.isPending}>
            Create Test Case
          </Button>
        </>
      }
    >
      <TestCaseForm
        id="edi-create-test-case-form"
        form={form}
        disabled={false}
        onSubmit={(values) => mutation.mutate(values)}
      />
    </DataTablePanelContainer>
  );
}

function TestCaseEditPanel({
  open,
  onOpenChange,
  testCaseId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  testCaseId: string | null;
}) {
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const { data: testCase } = useQuery({
    ...queries.edi.testCase(testCaseId ?? ""),
    enabled: open && Boolean(testCaseId),
  });
  const form = useForm<EDITestCaseFormValues>({
    resolver: zodResolver(ediTestCaseFormSchema),
    defaultValues: getTestCaseFormDefaults(),
    mode: "onChange",
  });
  const [preview, setPreview] = useState<EDIDocumentPreview | undefined>(undefined);
  const [inspectorOpen, setInspectorOpen] = useState(false);
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>("overview");
  const [selectedSegmentIndex, setSelectedSegmentIndex] = useState(1);
  const editorTheme = useEditorTheme();

  useEffect(() => {
    if (open && testCase) {
      form.reset(getTestCaseFormDefaults(testCase));
    }
  }, [form, open, testCase]);

  useEffect(() => {
    if (!open) {
      setPreview(undefined);
      setInspectorOpen(false);
      setInspectorTab("overview");
      setSelectedSegmentIndex(1);
    }
  }, [open]);

  const mutation = useApiMutation({
    mutationFn: (values: EDITestCaseFormValues) => {
      if (!testCaseId) {
        throw new Error("Test case is required");
      }
      return apiService.ediService.updateTestCase(testCaseId, toTestCaseRequest(values));
    },
    setFormError: form.setError,
    resourceName: "EDI Test Case",
    onSuccess: async () => {
      toast.success("EDI test case updated");
      await invalidateEDITestCases(queryClient, testCaseId ?? undefined);
    },
  });

  const previewMutation = useMutation({
    mutationFn: () => {
      if (!testCaseId) {
        throw new Error("Test case is required");
      }
      return apiService.ediService.previewTestCase(testCaseId);
    },
    onSuccess: (result) => {
      setPreview(result);
      setInspectorTab("overview");
      setInspectorOpen(true);
    },
    onError: () => {
      toast.error("Failed to run the test case preview");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!testCaseId) {
        throw new Error("Test case is required");
      }
      return apiService.ediService.deleteTestCase(testCaseId);
    },
    onSuccess: async () => {
      toast.success("EDI test case deleted");
      onOpenChange(false);
      await invalidateEDITestCases(queryClient, testCaseId ?? undefined);
    },
    onError: () => {
      toast.error("Failed to delete the test case");
    },
  });

  const canDelete = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Delete),
  );
  const isDirty = form.formState.isDirty;

  return (
    <>
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={testCase?.name ?? "EDI Test Case"}
        description="Run the stored payload through the partner's template and inspect the rendered X12."
        size="xl"
        footer={
          <>
            {testCase && canDelete && (
              <Button
                type="button"
                variant="destructive"
                onClick={() => deleteMutation.mutate()}
                isLoading={deleteMutation.isPending}
              >
                Delete
              </Button>
            )}
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            {testCase && (
              <Button
                type="button"
                variant="outline"
                onClick={() => previewMutation.mutate()}
                isLoading={previewMutation.isPending}
                disabled={isDirty}
                title={isDirty ? "Save your changes before running the preview" : undefined}
              >
                <PlayIcon className="size-4" />
                Run Preview
              </Button>
            )}
            {testCase && canUpdate && (
              <Button type="submit" form="edi-edit-test-case-form" isLoading={mutation.isPending}>
                Save Test Case
              </Button>
            )}
          </>
        }
      >
        {testCase && preview && (
          <TestCaseVerdict
            preview={preview}
            testCase={testCase}
            onOpenInspector={() => setInspectorOpen(true)}
          />
        )}
        {testCase && (
          <TestCaseForm
            id="edi-edit-test-case-form"
            form={form}
            disabled={!canUpdate}
            onSubmit={(values) => mutation.mutate(values)}
          />
        )}
      </DataTablePanelContainer>
      <PreviewInspectorSheet
        preview={preview}
        open={inspectorOpen}
        selectedTab={inspectorTab}
        selectedSegmentIndex={selectedSegmentIndex}
        editorTheme={editorTheme}
        onOpenChange={setInspectorOpen}
        onTabChange={setInspectorTab}
        onSelectSegment={(segmentIndex) => {
          setSelectedSegmentIndex(segmentIndex);
          setInspectorTab("segments");
        }}
      />
    </>
  );
}

function CodeDiffLine({ label, codes }: { label: string; codes: string[] }) {
  if (codes.length === 0) return null;
  return (
    <p className="text-xs">
      <span className="text-muted-foreground">{label}: </span>
      <span className="font-mono">{codes.join(", ")}</span>
    </p>
  );
}

function verdictCountLine(label: string, actual: number, expected: number) {
  if (actual === expected) {
    return `${label}: ${actual} (matches expected)`;
  }
  return `${label}: ${actual}, expected ${expected}`;
}

function diffCodes(expected: string[], actual: string[]) {
  if (expected.length === 0) return { missing: [] as string[], unexpected: [] as string[] };
  const expectedSet = new Set(expected);
  const actualSet = new Set(actual);
  return {
    missing: expected.filter((code) => !actualSet.has(code)),
    unexpected: Array.from(actualSet).filter((code) => !expectedSet.has(code)),
  };
}

function TestCaseVerdict({
  preview,
  testCase,
  onOpenInspector,
}: {
  preview: EDIDocumentPreview;
  testCase: EDITestCaseRow;
  onOpenInspector: () => void;
}) {
  const { expectedWarnings, expectedErrors } = testCase;
  const warningDiagnostics = preview.diagnostics.filter(
    (diagnostic) => diagnostic.severity === "Warning",
  );
  const errorDiagnostics = preview.diagnostics.filter(
    (diagnostic) => diagnostic.severity === "Error",
  );
  const actualWarnings = warningDiagnostics.length;
  const actualErrors = errorDiagnostics.length;
  const warningDiff = diffCodes(
    testCase.expectedWarningCodes ?? [],
    warningDiagnostics.map((diagnostic) => diagnostic.code),
  );
  const errorDiff = diffCodes(
    testCase.expectedErrorCodes ?? [],
    errorDiagnostics.map((diagnostic) => diagnostic.code),
  );
  const codesPass =
    warningDiff.missing.length === 0 &&
    warningDiff.unexpected.length === 0 &&
    errorDiff.missing.length === 0 &&
    errorDiff.unexpected.length === 0;
  const passed =
    actualWarnings === expectedWarnings && actualErrors === expectedErrors && codesPass;

  return (
    <div className="mb-4 flex items-center justify-between gap-3 rounded-md border bg-muted/20 p-3">
      <div className="flex items-center gap-3">
        <EDITestCaseVerdictBadge passed={passed} />
        <div className="text-sm">
          <p className={passed ? "text-muted-foreground" : "font-medium"}>
            {verdictCountLine("Warnings", actualWarnings, expectedWarnings)} ·{" "}
            {verdictCountLine("Errors", actualErrors, expectedErrors)}
          </p>
          {!passed && (
            <p className="text-xs text-muted-foreground">
              Review the inspector diagnostics, then either fix the payload/template or update the
              expected counts and codes.
            </p>
          )}
          <CodeDiffLine label="Missing warning codes" codes={warningDiff.missing} />
          <CodeDiffLine label="Unexpected warning codes" codes={warningDiff.unexpected} />
          <CodeDiffLine label="Missing error codes" codes={errorDiff.missing} />
          <CodeDiffLine label="Unexpected error codes" codes={errorDiff.unexpected} />
        </div>
      </div>
      <Button type="button" variant="ghost" size="sm" onClick={onOpenInspector}>
        Open Inspector
      </Button>
    </div>
  );
}

function TestCaseForm({
  id,
  form,
  disabled,
  onSubmit,
}: {
  id: string;
  form: UseFormReturn<EDITestCaseFormValues>;
  disabled: boolean;
  onSubmit: (values: EDITestCaseFormValues) => void;
}) {
  const { control, handleSubmit } = form;

  return (
    <Form
      id={id}
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit(onSubmit)(event);
      }}
    >
      <FormSection
        title="Test Case"
        description="Name the scenario and pick the partner document profile it certifies."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label="Name"
              placeholder="Partner 204 happy path"
              rules={{ required: true }}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <EDIDocumentProfileAutocompleteField
              control={control}
              name="partnerDocumentProfileId"
              label="Document Profile"
              placeholder="Select a document profile"
              rules={{ required: true }}
              disabled={disabled}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label="Description"
              placeholder="What this scenario certifies"
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Expected Outcome"
        description="Diagnostics the rendered document is expected to produce."
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="expectedWarnings"
              label="Expected Warnings"
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="expectedErrors"
              label="Expected Errors"
              disabled={disabled}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="expectedWarningCodes"
              label="Expected Warning Codes"
              disabled={disabled}
              placeholder="missing_optional_element, value_truncated"
              description="Optional comma-separated diagnostic codes. When set, the verdict also requires the preview's warning codes to match exactly."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="expectedErrorCodes"
              label="Expected Error Codes"
              disabled={disabled}
              placeholder="missing_required_element"
              description="Optional comma-separated diagnostic codes. When set, the verdict also requires the preview's error codes to match exactly."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Document Payload"
        description="Structured payload rendered through the profile's template when the preview runs."
      >
        <FormGroup cols={1}>
          <FormControl cols="full">
            <JsonEditorField
              control={control}
              name="payloadJson"
              label="Payload"
              description='JSON document payload, e.g. {"transactionSet":"204","loadTender":{...}}'
              disabled={disabled}
              height="320px"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </Form>
  );
}
