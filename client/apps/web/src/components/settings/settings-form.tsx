import { FormSaveDock } from "@/components/form-save-dock";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { zodResolver } from "@hookform/resolvers/zod";
import type { QueryKey } from "@tanstack/react-query";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import {
  FormProvider,
  useForm,
  type DefaultValues,
  type FieldPath,
  type FieldValues,
  type Resolver,
} from "react-hook-form";
import { settingsFillClass } from "./settings-layout";
import { tabHoldingError, useSettingsTab, type SettingsTabSpec } from "./settings-tab";
import { SettingsTabs } from "./settings-tabs";

/** What a tab's content may do beyond its own fields: know it is showing, open another tab. */
export type SettingsTabNav<TTab extends string> = {
  openTab: TTab;
  selectTab: (tab: TTab) => void;
};

export type SettingsFormTabSpec<TValues extends FieldValues, TTab extends string> = SettingsTabSpec<
  FieldPath<TValues>
> & { value: TTab };

export type SettingsFormProps<TValues extends FieldValues, TTab extends string> = {
  /** The record as the server sent it; the form starts from it and resets to what is saved. */
  values: TValues;
  schema: Parameters<typeof zodResolver>[0];
  /** The query the record was read with, updated optimistically and refetched after a save. */
  queryKey: QueryKey;
  save: (values: TValues) => Promise<unknown>;
  /** Other queries that read this record and refetch after a save, whether it landed or not. */
  refresh?: readonly QueryKey[];
  /** The name a save's toast and errors use, such as "Settlement Control". */
  resourceName: string;
  /**
   * The record as it is sent: settings the page hides because another makes them meaningless
   * are set the way the server requires then, so a save is never refused for a field the
   * person cannot see.
   */
  prepare?: (values: TValues) => TValues;
  /** The tabs the organization sees, in order, from the page's own tab spec. */
  tabs: readonly SettingsFormTabSpec<TValues, TTab>[];
  content: (tab: TTab, nav: SettingsTabNav<TTab>) => ReactNode;
};

/**
 * A settings page's form: one record, one form, one save, split into a tab for each job a
 * person comes to do. The open tab lives in the address, a save that is refused opens the
 * tab holding the problem, and the save dock appears once something changed. A page brings
 * its record, its tabs and what each tab holds; everything else is here.
 */
export function SettingsForm<TValues extends FieldValues, TTab extends string>({
  values,
  schema,
  queryKey,
  save,
  refresh = [],
  resourceName,
  prepare,
  tabs,
  content,
}: SettingsFormProps<TValues, TTab>) {
  const t = useT();

  const form = useForm<TValues>({
    resolver: zodResolver(schema) as unknown as Resolver<TValues>,
    defaultValues: values as DefaultValues<TValues>,
  });
  const { handleSubmit, reset, getFieldState } = form;

  const [open, select] = useSettingsTab(tabs);
  const openTab = open as TTab;
  const selectTab = select as (tab: TTab) => void;

  const { mutateAsync } = useOptimisticMutation({
    queryKey,
    mutationFn: save,
    resourceName,
    resetForm: reset,
    form,
    invalidateQueries: [queryKey, ...refresh],
  });

  // A save refused for a field on another tab would otherwise fail out of sight. An error on
  // one entry of a list (`events.3`) makes the list itself invalid, so its tab opens too.
  const openRefusedTab = () => {
    const tab = tabHoldingError(tabs, openTab, (field) => getFieldState(field).invalid);
    if (tab) selectTab(tab as TTab);
  };

  const onSubmit = async (submitted: TValues) => {
    try {
      await mutateAsync(prepare ? prepare(submitted) : submitted);
    } catch {
      // The mutation has already put the refusal on the form or in a toast.
      openRefusedTab();
    }
  };

  const nav = { openTab, selectTab };

  return (
    <FormProvider {...form}>
      <Form className={settingsFillClass} onSubmit={handleSubmit(onSubmit, openRefusedTab)}>
        <SettingsTabs
          tabs={tabs.map((spec) => ({ ...spec, content: content(spec.value, nav) }))}
          value={openTab}
          onValueChange={selectTab as (value: string) => void}
        />
        <FormSaveDock saveButtonContent={t("Save changes")} />
      </Form>
    </FormProvider>
  );
}
