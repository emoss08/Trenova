import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { useEditRecordReset } from "../use-edit-record-reset";

type Record = {
  id?: string;
  name: string;
  rules: Array<{ id: string; label: string }>;
};

const bareRow: Record = { id: "rag_01", name: "Acme TL 2026", rules: [] };

const fullRecord: Record = {
  id: "rag_01",
  name: "Acme TL 2026",
  rules: [{ id: "ragr_01", label: "Dallas to Chicago" }],
};

function harness(options: {
  open: boolean;
  mode: "create" | "edit";
  id?: string;
  version?: number;
  fetch: (id: string) => Promise<Record>;
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  return renderHook(
    (props: typeof options) => {
      const form = useForm<Record>({ defaultValues: bareRow });
      useEditRecordReset(form, { ...props, queryKey: "record" });
      return form;
    },
    { wrapper: Wrapper, initialProps: options },
  );
}

describe("useEditRecordReset", () => {
  // The table row a panel opens from carries no children, so a form seated
  // from it alone shows every child editor empty — and saving that emptiness
  // erases the record's children. The hook exists to re-seat the form from
  // the detail endpoint.
  it("re-seats an edit form from the fetched full record", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord);

    const { result } = harness({ open: true, mode: "edit", id: "rag_01", version: 3, fetch });

    await waitFor(() => {
      expect(result.current.getValues("rules")).toHaveLength(1);
    });
    expect(fetch).toHaveBeenCalledWith("rag_01");
    expect(result.current.getValues("rules.0.label")).toBe("Dallas to Chicago");
  });

  it("never fetches for a create panel, which has no record to load", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord);

    harness({ open: true, mode: "create", id: undefined, version: undefined, fetch });

    await Promise.resolve();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("refetches when a save bumps the row version, rather than re-seating from cache", async () => {
    const fetch = vi.fn().mockResolvedValue(fullRecord);

    const { result, rerender } = harness({
      open: true,
      mode: "edit",
      id: "rag_01",
      version: 3,
      fetch,
    });

    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(1));

    const revised: Record = {
      ...fullRecord,
      rules: [{ id: "ragr_02", label: "Dallas to Chicago, revised" }],
    };
    fetch.mockResolvedValue(revised);

    rerender({ open: true, mode: "edit", id: "rag_01", version: 4, fetch });

    await waitFor(() => {
      expect(result.current.getValues("rules.0.label")).toBe("Dallas to Chicago, revised");
    });
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
