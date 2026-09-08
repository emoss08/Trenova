import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { useController, type Control, type FieldValues, type Path } from "react-hook-form";
import { MemoryRouter } from "react-router";

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
  return (
    <div>
      <h1>{pageHeaderProps.title}</h1>
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

export function renderAccountingPage(ui: ReactElement, initialEntries: string[] = ["/"]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
}
