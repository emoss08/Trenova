import { useT } from "@trenova/shared/i18n/use-t";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { useController, type Control, type FieldValues, type Path } from "react-hook-form";
import { renderAtRoutes, TestRouter } from "@trenova/shared/testing/test-router";
import type { RouteComponent } from "@tanstack/react-router";

/**
 * A page layout that keeps only what an empty-state test looks for: the
 * header actions and the page body.
 */
export function PageLayoutStub({
  pageHeaderProps,
  children,
}: {
  pageHeaderProps: { title: string; actions?: ReactNode };
  children: ReactNode;
}) {
  const t = useT();

  return (
    <div>
      <h1>{t(pageHeaderProps.title)}</h1>
      {pageHeaderProps.actions}
      {children}
    </div>
  );
}

/**
 * A plain input bound to the form, standing in for the async autocomplete
 * and date pickers so a test can type a value the page filters on.
 */
export function FormFieldStub({
  control,
  name,
  label,
  placeholder,
}: {
  control: Control<FieldValues>;
  name: Path<FieldValues>;
  label?: string;
  placeholder?: string;
}) {
  const { field } = useController({ control, name });
  return (
    <input
      aria-label={label ?? placeholder}
      value={(field.value as string) ?? ""}
      onChange={(event) => field.onChange(event.target.value)}
    />
  );
}

/** Renders `Page` as the route at `path` (with its params), opened at `entry`. */
export async function renderAccountingRoute(path: string, Page: RouteComponent, entry: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderAtRoutes([{ path, component: Page }], {
    initialEntries: [entry],
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
}

export function renderAccountingPage(ui: ReactElement, initialEntries: string[] = ["/"]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <TestRouter initialEntries={initialEntries}>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </TestRouter>,
  );
}
