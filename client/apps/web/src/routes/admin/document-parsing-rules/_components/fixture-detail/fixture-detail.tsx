import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Form, FormSection, FormGroup, FormControl } from "@/components/ui/form";
import { FormSaveDock } from "@/components/form-save-dock";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { NumberField } from "@/components/fields/number-field";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  fixtureSchema,
  type Fixture,
  type FixtureFormValues,
} from "@/types/document-parsing-rule";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  FileTextIcon,
  PlusIcon,
  TrashIcon,
} from "lucide-react";
import { useCallback, useMemo } from "react";
import { FormProvider, useFieldArray, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { AssertionsEditor } from "./assertions-editor";

type FixtureDetailProps = {
  fixtureId: string;
  ruleSetId: string;
  onBack: () => void;
  onDeleted: () => void;
};

export function FixtureDetail({
  fixtureId,
  ruleSetId,
  onBack,
  onDeleted,
}: FixtureDetailProps) {
  const { data } = useQuery({
    ...queries.documentParsingRule.fixture(fixtureId),
  });

  if (!data) return null;

  return (
    <FixtureForm
      fixture={data}
      ruleSetId={ruleSetId}
      onBack={onBack}
      onDeleted={onDeleted}
    />
  );
}

function FixtureForm({
  fixture,
  onBack,
  onDeleted,
}: {
  fixture: Fixture;
  ruleSetId: string;
  onBack: () => void;
  onDeleted: () => void;
}) {
  const queryClient = useQueryClient();
  const { allowed: canDelete } = usePermission(
    Resource.DocumentParsingRule,
    Operation.Delete,
  );

  const form = useForm<FixtureFormValues, unknown, Fixture>({
    resolver: zodResolver(fixtureSchema),
    defaultValues: fixture,
  });

  const { handleSubmit, reset, setError, control } = form;
  const { fields: pageFields, append: appendPage, remove: removePage } =
    useFieldArray({ control, name: "pageSnapshots" });

  const textSnapshot = useWatch({ control, name: "textSnapshot" });

  const lineCount = useMemo(() => {
    if (!textSnapshot) return 0;
    return textSnapshot.split("\n").length;
  }, [textSnapshot]);

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.documentParsingRule.fixture._def,
    mutationFn: async (values: Fixture) =>
      apiService.documentParsingRuleService.updateFixture(fixture.id!, values),
    resourceName: "Fixture",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [
      queries.documentParsingRule.fixture._def,
      queries.documentParsingRule.fixtures._def,
    ],
  });

  const deleteMutation = useMutation({
    mutationFn: () =>
      apiService.documentParsingRuleService.deleteFixture(fixture.id!),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.fixtures._def,
      });
      toast.success("Fixture deleted");
      onDeleted();
    },
  });

  const onSubmit = useCallback(
    async (values: Fixture) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={onBack}
              >
                <ArrowLeftIcon className="size-4" />
              </Button>
              <h3 className="text-base font-semibold">{fixture.name}</h3>
            </div>
            {canDelete && (
              <AlertDialog>
                <AlertDialogTrigger
                  render={
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="text-destructive"
                    >
                      <TrashIcon className="size-4" />
                    </Button>
                  }
                />
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Delete Fixture</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will permanently delete &quot;{fixture.name}&quot;.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={() => deleteMutation.mutate()}
                      className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                      Delete
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
          </div>

          <FormSection
            title="Fixture Details"
            description="Identify this fixture with a name and optional metadata used during provider matching."
          >
            <FormGroup cols={2}>
              <FormControl>
                <InputField
                  control={control}
                  name="name"
                  label="Name"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="fileName"
                  label="File Name"
                  placeholder="e.g. rate_confirmation.pdf"
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="providerFingerprint"
                  label="Provider Fingerprint"
                  placeholder="e.g. ch_robinson"
                />
              </FormControl>
              <FormControl cols={2}>
                <TextareaField
                  control={control}
                  name="description"
                  label="Description"
                />
              </FormControl>
            </FormGroup>
          </FormSection>

          <FormSection
            title="Text Snapshot"
            description="The full extracted text of the document. This is the primary input the parser operates on during simulation."
            action={
              lineCount > 0 ? (
                <Badge variant="outline" className="font-normal">
                  {lineCount} line{lineCount !== 1 ? "s" : ""}
                </Badge>
              ) : undefined
            }
          >
            <TextareaField
              control={control}
              name="textSnapshot"
              label="Full Document Text"
              placeholder="Paste the full document text here..."
              rules={{ required: true }}
              className="font-mono text-xs"
            />
          </FormSection>

          <FormSection
            title="Page Snapshots"
            titleCount={pageFields.length}
            description="Per-page text used when rules target specific pages. If omitted, the parser uses the full text snapshot."
            action={
              <Button
                type="button"
                variant="outline"
                size="xxs"
                className="gap-1"
                onClick={() =>
                  appendPage({
                    pageNumber: pageFields.length + 1,
                    text: "",
                  })
                }
              >
                <PlusIcon className="size-3.5" />
                Add Page
              </Button>
            }
          >
            <div className="space-y-2">
              {pageFields.length === 0 && (
                <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-8 text-center">
                  <FileTextIcon className="size-5 text-muted-foreground" />
                  <p className="text-xs text-muted-foreground">
                    No page snapshots defined. Add pages if the document has
                    page-specific content.
                  </p>
                </div>
              )}
              {pageFields.length > 0 && (
                <div className="mb-2 flex flex-wrap gap-1">
                  {pageFields.map((pf, idx) => (
                    <Badge key={pf.id} variant="secondary" className="font-mono text-xs">
                      Page {pf.pageNumber || idx + 1}
                    </Badge>
                  ))}
                </div>
              )}
              {pageFields.map((pf, idx) => (
                <Collapsible key={pf.id}>
                  <div className="rounded-md border">
                    <CollapsibleTrigger className="flex w-full items-center justify-between p-3 text-sm hover:bg-muted/50">
                      <span>Page {pf.pageNumber || idx + 1}</span>
                      <div className="flex items-center gap-1">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="size-7"
                          onClick={(e) => {
                            e.stopPropagation();
                            removePage(idx);
                          }}
                        >
                          <TrashIcon className="size-3.5 text-destructive" />
                        </Button>
                        <ChevronDownIcon className="size-4 transition-transform [[data-state=open]>&]:rotate-180" />
                      </div>
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <div className="space-y-2 border-t p-3">
                        <NumberField
                          control={control}
                          name={`pageSnapshots.${idx}.pageNumber`}
                          label="Page Number"
                        />
                        <TextareaField
                          control={control}
                          name={`pageSnapshots.${idx}.text`}
                          label="Page Text"
                          className="font-mono text-xs"
                        />
                      </div>
                    </CollapsibleContent>
                  </div>
                </Collapsible>
              ))}
            </div>
          </FormSection>

          <AssertionsEditor />

          <FormSaveDock saveButtonContent="Save Fixture" />
        </div>
      </Form>
    </FormProvider>
  );
}
