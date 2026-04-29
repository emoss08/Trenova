import { FormSaveDock } from "@/components/form-save-dock";
import { Form } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  sequenceConfigDocumentSchema,
  sequenceTypes,
  type SequenceConfigDocument,
  type SequenceType,
} from "@/types/sequence-config";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useMemo } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { SequenceConfigPanel } from "./sequence-config-panel";
import { SequenceConfigSidebar } from "./sequence-config-sidebar";

const sectionParser = parseAsStringLiteral(sequenceTypes)
  .withOptions({ history: "push", shallow: true })
  .withDefault(sequenceTypes[0]);

export default function SequenceConfigForm() {
  const { data } = useSuspenseQuery({
    ...queries.sequenceConfig.get(),
  });

  const form = useForm({
    resolver: zodResolver(sequenceConfigDocumentSchema),
    defaultValues: data,
  });

  const { handleSubmit, reset, setError } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.sequenceConfig.get._def,
    mutationFn: async (values: SequenceConfigDocument) =>
      apiService.sequenceConfigService.update(values),
    resourceName: "Sequence Configuration",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.sequenceConfig.get._def],
  });

  const onSubmit = useCallback(
    async (values: SequenceConfigDocument) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const [section, setSection] = useQueryState("section", sectionParser);

  const indexByType = useMemo(() => {
    const map = {} as Record<SequenceType, number>;
    for (const type of sequenceTypes) {
      map[type] = data.configs.findIndex((cfg) => cfg.sequenceType === type);
    }
    return map;
  }, [data.configs]);

  const activeIndex = indexByType[section];

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex gap-6 pb-20">
          <SequenceConfigSidebar
            value={section}
            onChange={setSection}
            indexByType={indexByType}
          />

          {activeIndex >= 0 ? (
            <SequenceConfigPanel index={activeIndex} sequenceType={section} />
          ) : null}
        </div>

        <FormSaveDock saveButtonContent="Save Changes" />
      </Form>
    </FormProvider>
  );
}
