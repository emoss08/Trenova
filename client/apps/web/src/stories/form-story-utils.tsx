import { Button } from "@trenova/shared/components/ui/button";
import type { QueryKey } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, type ReactNode } from "react";
import {
  FormProvider,
  useForm,
  useWatch,
  type Control,
  type DefaultValues,
  type FieldErrors,
  type FieldValues,
  type Path,
  type UseFormReturn,
} from "react-hook-form";

export type StoryFormRenderProps<T extends FieldValues> = {
  control: Control<T>;
  form: UseFormReturn<T>;
};

type StoryFormProps<T extends FieldValues> = {
  defaultValues: DefaultValues<T>;
  children: (props: StoryFormRenderProps<T>) => ReactNode;
  forcedErrors?: Record<string, string>;
};

type QuerySeed = {
  queryKey: QueryKey;
  value: unknown;
};

type QuerySeedProviderProps = {
  seeds: QuerySeed[];
  children: ReactNode;
};

function formatErrors(errors: FieldErrors<FieldValues>) {
  return Object.entries(errors).reduce<Record<string, string>>((acc, [key, value]) => {
    if (!value) return acc;
    const message = "message" in value ? value.message : undefined;
    acc[key] = typeof message === "string" ? message : "Invalid value";
    return acc;
  }, {});
}

function DebugPanel<T extends FieldValues>({
  values,
  errors,
}: {
  values: T;
  errors: FieldErrors<T>;
}) {
  return (
    <aside className="rounded-md border bg-muted p-3">
      <div className="mb-2 flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium">Form State</h3>
        <span className="text-2xs text-muted-foreground">react-hook-form</span>
      </div>
      <div className="grid gap-3">
        <div>
          <p className="mb-1 text-2xs font-medium text-muted-foreground">Values</p>
          <pre className="max-h-80 overflow-auto rounded-md bg-background p-2 text-2xs">
            {JSON.stringify(values, null, 2)}
          </pre>
        </div>
        <div>
          <p className="mb-1 text-2xs font-medium text-muted-foreground">Errors</p>
          <pre className="max-h-40 overflow-auto rounded-md bg-background p-2 text-2xs">
            {JSON.stringify(formatErrors(errors as FieldErrors<FieldValues>), null, 2)}
          </pre>
        </div>
      </div>
    </aside>
  );
}

function ForcedErrors<T extends FieldValues>({
  form,
  forcedErrors,
}: {
  form: UseFormReturn<T>;
  forcedErrors?: Record<string, string>;
}) {
  useEffect(() => {
    if (!forcedErrors) return;

    Object.entries(forcedErrors).forEach(([name, message]) => {
      if (typeof message === "string") {
        form.setError(name as Path<T>, { type: "storybook", message });
      }
    });
  }, [forcedErrors, form]);

  return null;
}

export function QuerySeedProvider({ seeds, children }: QuerySeedProviderProps) {
  const queryClient = useQueryClient();

  useEffect(() => {
    seeds.forEach((seed) => {
      queryClient.setQueryData(seed.queryKey, seed.value);
    });
  }, [queryClient, seeds]);

  return children;
}

export function StoryForm<T extends FieldValues>({
  defaultValues,
  children,
  forcedErrors,
}: StoryFormProps<T>) {
  const form = useForm<T>({
    defaultValues,
    mode: "onChange",
  });
  const values = useWatch({ control: form.control }) as T;
  const errors = form.formState.errors;

  return (
    <FormProvider {...form}>
      <ForcedErrors form={form} forcedErrors={forcedErrors} />
      <form
        className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]"
        onSubmit={(event) => event.preventDefault()}
      >
        <div className="rounded-md border bg-background p-4">
          <div className="grid gap-4">{children({ control: form.control, form })}</div>
          <div className="mt-4 flex justify-end">
            <Button type="button" variant="outline" onClick={() => void form.trigger()}>
              Validate
            </Button>
          </div>
        </div>
        <DebugPanel values={values} errors={errors} />
      </form>
    </FormProvider>
  );
}

export function StorySection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="grid gap-3">
      <h2 className="text-base font-semibold">{title}</h2>
      <div className="grid gap-4 md:grid-cols-2">{children}</div>
    </section>
  );
}
