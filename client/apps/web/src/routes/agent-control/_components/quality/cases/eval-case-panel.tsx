import { useT } from "@trenova/shared/i18n/use-t";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_EVAL_CASE_DETAIL_KEY,
  AGENT_EVAL_CASE_LIST_KEY,
  createAgentEvalCase,
  fetchAgentEvalCase,
  replayAgentEvalCase,
  setAgentEvalCaseStatus,
  updateAgentEvalCase,
  type AgentEvalCaseDetail,
  type AgentEvalCaseRow,
} from "@/lib/graphql/agent-eval-cases";
import { AGENT_EVALUATION_LIST_KEY } from "@/lib/graphql/agent-evaluations";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import type { AgentEvalCaseStatus } from "@trenova/graphql/generated/graphql";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { PlayIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { CASE_STATUS_ACTION, CASE_STATUS_MOVED } from "./eval-case-badges";
import { EvalCaseEditor } from "./eval-case-editor";
import {
  evalCaseFormDefaults,
  evalCaseFormSchema,
  nextStatuses,
  toCuratedInput,
  toFormValues,
  toUpdateInput,
  type EvalCaseFormValues,
} from "./eval-case-model";
import { EvalCaseProvenance } from "./eval-case-provenance";

const FORM_ID = "agent-eval-case-form";

export function evalCaseDetailKey(id: string | undefined) {
  return [AGENT_EVAL_CASE_DETAIL_KEY, id] as const;
}

/**
 * One evaluation case: where it came from, what it asks, and what a good answer
 * does. A new case is written by hand; a captured one keeps its frozen input and
 * only its expectations are edited.
 */
export function EvalCasePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AgentEvalCaseRow>) {
  const t = useT();
  const queryClient = useQueryClient();
  const creating = mode !== "edit";
  const caseId = creating ? undefined : row?.id;
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);

  const detail = useQuery({
    queryKey: evalCaseDetailKey(caseId),
    queryFn: ({ signal }) => fetchAgentEvalCase(caseId ?? "", { signal }),
    enabled: open && caseId !== undefined,
  });

  const form = useForm<EvalCaseFormValues>({
    resolver: zodResolver(evalCaseFormSchema) as Resolver<EvalCaseFormValues>,
    defaultValues: evalCaseFormDefaults,
    mode: "onChange",
  });
  const { reset, handleSubmit } = form;

  useEffect(() => {
    if (!open) return;
    if (creating) {
      reset(evalCaseFormDefaults);
    } else if (detail.data) {
      reset(toFormValues(detail.data));
    }
  }, [open, creating, detail.data, reset]);

  const refresh = async (updated?: AgentEvalCaseDetail) => {
    if (updated) {
      queryClient.setQueryData(evalCaseDetailKey(updated.id), updated);
    }
    await queryClient.invalidateQueries({ queryKey: [AGENT_EVAL_CASE_LIST_KEY] });
  };

  const save = useApiMutation<
    { evalCase: AgentEvalCaseDetail; duplicate: boolean },
    EvalCaseFormValues,
    unknown,
    EvalCaseFormValues
  >({
    mutationFn: async (values) => {
      if (creating) {
        return createAgentEvalCase(toCuratedInput(values));
      }
      if (!detail.data) {
        throw new Error("The case is still loading");
      }
      const evalCase = await updateAgentEvalCase(
        detail.data.id,
        toUpdateInput(values, detail.data.expiresAt ?? null),
      );
      return { evalCase, duplicate: false };
    },
    onSuccess: async ({ evalCase, duplicate }) => {
      await refresh(evalCase);
      if (duplicate) {
        toast.info(t("A case already asks exactly this"), {
          description: t("The existing case was kept instead of adding a second one."),
        });
      } else {
        toast.success(creating ? t("Case added") : t("Changes have been saved"));
      }
      if (creating) {
        reset(evalCaseFormDefaults);
        onOpenChange(false);
        return;
      }
      reset(toFormValues(evalCase));
    },
    form,
    resourceName: t("Evaluation case"),
  });

  const moveTo = useApiMutation<AgentEvalCaseDetail, AgentEvalCaseStatus>({
    mutationFn: (status) => setAgentEvalCaseStatus(caseId ?? "", status),
    onSuccess: async (evalCase) => {
      await refresh(evalCase);
      reset(toFormValues(evalCase));
      toast.success(t(CASE_STATUS_MOVED[evalCase.status]));
    },
    resourceName: t("Evaluation case"),
  });

  const replay = useApiMutation({
    mutationFn: () => replayAgentEvalCase(caseId ?? ""),
    onSuccess: async () => {
      toast.success(t("Replay started"), {
        description: t("The agent as it is now answers the case with every write simulated."),
      });
      await queryClient.invalidateQueries({ queryKey: [AGENT_EVALUATION_LIST_KEY] });
    },
    resourceName: t("Evaluation case"),
  });

  const close = () => {
    reset(evalCaseFormDefaults);
    onOpenChange(false);
  };

  const evalCase = detail.data;
  const editable = creating ? canCreate : canUpdate && evalCase?.status !== "Retired";
  const title = creating
    ? t("New evaluation case")
    : evalCase?.title || evalCase?.input || t("Evaluation case");

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={(next) => (next ? onOpenChange(true) : close())}
      title={title}
      size="xl"
      headerActions={
        evalCase && canUpdate ? (
          <div className="flex items-center gap-2">
            {nextStatuses(evalCase.status).map((status) => (
              <Button
                key={status}
                type="button"
                size="sm"
                variant={status === "Retired" ? "outline" : "secondary"}
                isLoading={moveTo.isPending && moveTo.variables === status}
                onClick={() => moveTo.mutate(status)}
              >
                {t(CASE_STATUS_ACTION[status])}
              </Button>
            ))}
            {canCreate && evalCase.status !== "Retired" ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                isLoading={replay.isPending}
                onClick={() => replay.mutate(undefined)}
              >
                <PlayIcon />
                {t("Replay now")}
              </Button>
            ) : null}
          </div>
        ) : undefined
      }
      footer={
        <>
          <Button type="button" variant="outline" onClick={close}>
            {t("Cancel")}
          </Button>
          {editable ? (
            <Button
              type="submit"
              form={FORM_ID}
              isLoading={save.isPending}
              loadingText={t("Saving...")}
            >
              {creating ? t("Add case") : t("Save")}
            </Button>
          ) : null}
        </>
      }
    >
      {!creating && !evalCase ? (
        <ComponentLoader message={t("Loading the case")} />
      ) : (
        <FormProvider {...form}>
          <Form
            id={FORM_ID}
            className="flex flex-col gap-4"
            onSubmit={handleSubmit((values) => save.mutate(values))}
          >
            {evalCase ? <EvalCaseProvenance evalCase={evalCase} /> : null}
            <fieldset disabled={!editable} className="contents">
              <EvalCaseEditor creating={creating} />
            </fieldset>
          </Form>
        </FormProvider>
      )}
    </DataTablePanelContainer>
  );
}
