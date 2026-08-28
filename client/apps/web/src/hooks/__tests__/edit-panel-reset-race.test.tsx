import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { useEffect, type ReactNode } from "react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { useEditRecordReset } from "../use-edit-record-reset";

type Rule = { id: string; label: string };
type Agreement = {
  id?: string;
  version?: number;
  name: string;
  rules: Rule[];
};

/** What the list endpoint returns: the header columns, and no children. */
const listRow = (version: number): Agreement =>
  ({ id: "rag_01", version, name: "Acme TL 2026" }) as Agreement;

/** What the detail endpoint returns. */
const fullRecord = (version: number): Agreement => ({
  id: "rag_01",
  version,
  name: "Acme TL 2026",
  rules: [{ id: "ragr_01", label: "Dallas to Chicago" }],
});

let formHandle: UseFormReturn<Agreement> | null = null;
let readyHandle: boolean | null = null;

/**
 * The reset TabbedFormEditPanel runs, in a child component, exactly as the real
 * panel does: the parent owns the detail fetch and the child re-seats the form
 * from the bare table row.
 */
function ChildPanel({
  open,
  row,
  form,
}: {
  open: boolean;
  row?: Agreement;
  form: UseFormReturn<Agreement>;
}) {
  const { reset } = form;

  useEffect(() => {
    if (open && row) {
      reset(row);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, row?.version, reset]);

  return null;
}

function ParentPanel({
  open,
  row,
  fetch,
}: {
  open: boolean;
  row?: Agreement;
  fetch: (id: string) => Promise<Agreement>;
}) {
  const form = useForm<Agreement>({
    defaultValues: { name: "", rules: [] },
  });
  formHandle = form;

  const { isSeated } = useEditRecordReset(form, {
    open,
    mode: "edit",
    queryKey: "rate-agreement",
    id: row?.id,
    version: row?.version,
    fetch,
  });
  readyHandle = isSeated;

  return <ChildPanel open={open} row={row} form={form} />;
}

function renderPanel(props: {
  open: boolean;
  row?: Agreement;
  fetch: (id: string) => Promise<Agreement>;
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  const utils = render(
    <Wrapper>
      <ParentPanel {...props} />
    </Wrapper>,
  );

  return {
    ...utils,
    rerenderWith: (next: typeof props) =>
      utils.rerender(
        <Wrapper>
          <ParentPanel {...next} />
        </Wrapper>,
      ),
  };
}

/** A detail fetch that only resolves when the test says so. */
function deferredFetch() {
  let release: (value: Agreement) => void = () => {};
  const promise = new Promise<Agreement>((resolve) => {
    release = resolve;
  });

  return { fetch: vi.fn(() => promise), release: () => release(fullRecord(3)) };
}

describe("edit panel reset race", () => {
  // The baseline the panel is supposed to deliver.
  it("seats the lanes from the detail record once it lands", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord(3));

    renderPanel({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(formHandle?.getValues("rules")).toHaveLength(1));
  });

  // The list refetches constantly — on window focus, after any save, after any
  // invalidation. Whenever it comes back with a bumped version, the panel's own
  // reset re-seats the form from the bare row, and the lanes have to come back.
  it("restores the lanes after the list row's version bumps under it", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord(3));

    const { rerenderWith } = renderPanel({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(formHandle?.getValues("rules")).toHaveLength(1));

    fetch.mockResolvedValue(fullRecord(4));
    rerenderWith({ open: true, row: listRow(4), fetch });

    await waitFor(() => expect(formHandle?.getValues("rules")).toHaveLength(1));
  });

  // Re-opening a record already in the query cache is the common case: the
  // detail data is there synchronously, so both resets land in the same commit
  // and the order between them decides whether the lanes survive.
  it("keeps the lanes when the record is served from cache on re-open", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord(3));

    const { rerenderWith } = renderPanel({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(formHandle?.getValues("rules")).toHaveLength(1));

    rerenderWith({ open: false, row: listRow(3), fetch });
    rerenderWith({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(formHandle?.getValues("rules")).toHaveLength(1));
  });

  // The panel opens seated from the bare list row, so until the detail record
  // lands the form holds no lanes at all. Submitting in that window sends
  // `rules: []`, and the server reads a lane it was not sent as a lane the user
  // deleted: planRuleAmendment closes out every one of them. The record has to
  // say it is not ready so the panel can refuse to submit.
  it("reports the form as unseated until the detail record lands", async () => {
    const { fetch, release } = deferredFetch();

    renderPanel({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(fetch).toHaveBeenCalled());

    expect(formHandle?.getValues("rules") ?? []).toHaveLength(0);
    expect(readyHandle).toBe(false);

    release();

    await waitFor(() => expect(readyHandle).toBe(true));
    expect(formHandle?.getValues("rules")).toHaveLength(1);
  });

  // A detail fetch that fails leaves the form holding the bare row. Reporting
  // it as seated would let the panel save that emptiness back over a contract
  // that has lanes.
  it("never reports a failed detail fetch as seated", async () => {
    const fetch = vi.fn().mockRejectedValue(new Error("boom"));

    renderPanel({ open: true, row: listRow(3), fetch });

    await waitFor(() => expect(fetch).toHaveBeenCalled());

    expect(readyHandle).toBe(false);
  });

  // A create panel has no record to wait for, so it must never be held back.
  it("reports a create panel as seated immediately", async () => {
    const fetch = vi.fn();

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <CreatePanel fetch={fetch} />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(readyHandle).toBe(true));
    expect(fetch).not.toHaveBeenCalled();
  });
});

function CreatePanel({ fetch }: { fetch: (id: string) => Promise<Agreement> }) {
  const form = useForm<Agreement>({ defaultValues: { name: "", rules: [] } });
  formHandle = form;

  const { isSeated } = useEditRecordReset(form, {
    open: true,
    mode: "create",
    queryKey: "rate-agreement",
    fetch,
  });
  readyHandle = isSeated;

  return null;
}
