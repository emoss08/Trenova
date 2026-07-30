import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { GraphQLRequestError, type NormalizedGraphQLError } from "@trenova/shared/lib/graphql";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { isRegisteredLeaf, normalizeFieldPath } from "@/lib/form-errors";

const PROBLEM_BASE = "https://api.trenova.app/problems/";

const toastError = vi.hoisted(() => vi.fn());
vi.mock("sonner", () => ({ toast: { error: toastError, success: vi.fn() } }));

// Mirrors JournalFormValues in routes/manual-journal/_components/manual-journal-panel.tsx.
// Deliberately has no `status` field — the Go validator emits one.
type JournalFormValues = {
  description: string;
  reason: string;
  accountingDate: number | null;
  requestedFiscalYearId: string;
  requestedFiscalPeriodId: string;
  currencyCode: string;
  lines: { glAccountId: string; description: string; debitAmount: number; creditAmount: number }[];
};

const LINE_PATHS = ["glAccountId", "description", "debitAmount", "creditAmount"] as const;
const TOP_LEVEL_PATHS = [
  "description",
  "reason",
  "accountingDate",
  "requestedFiscalYearId",
  "requestedFiscalPeriodId",
  "currencyCode",
] as const;

function renderJournalForm() {
  const { result } = renderHook(() =>
    useForm<JournalFormValues>({
      defaultValues: {
        description: "",
        reason: "",
        accountingDate: null,
        requestedFiscalYearId: "",
        requestedFiscalPeriodId: "",
        currencyCode: "USD",
        lines: [
          { glAccountId: "", description: "", debitAmount: 0, creditAmount: 0 },
          { glAccountId: "", description: "", debitAmount: 0, creditAmount: 0 },
        ],
      },
    }),
  );

  // register() is what an input does on mount. Without it nothing is a registered leaf,
  // which is precisely the distinction under test.
  act(() => {
    for (const path of TOP_LEVEL_PATHS) {
      result.current.register(path);
    }
    for (const index of [0, 1]) {
      for (const leaf of LINE_PATHS) {
        result.current.register(`lines.${index}.${leaf}` as never);
      }
    }
  });

  return result;
}

function errorsOf(form: ReturnType<typeof renderJournalForm>["current"]) {
  return (form.control as unknown as { _formState: { errors: Record<string, any> } })._formState
    .errors;
}

function validationError(
  fieldErrors: { field: string; message: string; code?: string }[],
  message = "Validation failed",
  traceId = "trace-A",
): NormalizedGraphQLError {
  return {
    extensions: {
      code: "REQUIRED",
      type: `${PROBLEM_BASE}validation-error`,
      traceId,
      errors: fieldErrors,
    },
    code: "REQUIRED",
    errors: fieldErrors,
    message,
    traceId,
    type: `${PROBLEM_BASE}validation-error`,
  };
}

function graphQLError(...graphQLErrors: NormalizedGraphQLError[]): GraphQLRequestError {
  return new GraphQLRequestError({
    graphQLErrors,
    kind: "graphql",
    message: graphQLErrors[0]?.message ?? "Validation failed",
    status: 200,
  });
}

// The two-mutation multi-error response characterised in the Phase 3 probe.
const MANUAL_JOURNAL_ERROR = graphQLError(
  validationError(
    [
      { field: "lines[0].glAccountId", message: "GL account is required", code: "REQUIRED" },
      { field: "reason", message: "Reason is required", code: "REQUIRED" },
      {
        field: "lines[1].debitAmount",
        message: "Debit amount cannot be negative",
        code: "INVALID",
      },
    ],
    "Validation failed",
    "trace-A",
  ),
  validationError(
    [
      { field: "lines[0].glAccountId", message: "GL account 4000 is inactive", code: "INVALID" },
      { field: "status", message: "Only draft journals can be submitted", code: "INVALID" },
      { field: "reason", message: "Reason is required", code: "REQUIRED" },
    ],
    "Validation failed",
    "trace-B",
  ),
);

describe("normalizeFieldPath", () => {
  it("translates the server's bracket indices to react-hook-form's dot indices", () => {
    expect(normalizeFieldPath("lines[0].glAccountId")).toBe("lines.0.glAccountId");
    expect(normalizeFieldPath("lines[12].debitAmount")).toBe("lines.12.debitAmount");
    expect(normalizeFieldPath("reason")).toBe("reason");
    expect(normalizeFieldPath("[0].name")).toBe("0.name");
    expect(normalizeFieldPath("")).toBe("");
  });
});

describe("isRegisteredLeaf", () => {
  it("accepts a registered leaf and rejects everything else", () => {
    const form = renderJournalForm();

    expect(isRegisteredLeaf(form.current.control, "reason")).toBe(true);
    expect(isRegisteredLeaf(form.current.control, "lines.0.glAccountId")).toBe(true);

    // The array element itself is not a leaf, even though getValues() returns an object
    // for it — that is exactly why the getValues probe was the wrong instrument.
    expect(form.current.getValues("lines.0" as never)).toBeTruthy();
    expect(isRegisteredLeaf(form.current.control, "lines.0")).toBe(false);

    expect(isRegisteredLeaf(form.current.control, "lines")).toBe(false);
    expect(isRegisteredLeaf(form.current.control, "status")).toBe(false);
    expect(isRegisteredLeaf(form.current.control, "")).toBe(false);
  });
});

describe("getFieldErrors precedence", () => {
  it("returns every field error, and narrows to one field on request", () => {
    expect(MANUAL_JOURNAL_ERROR.getFieldErrors()).toHaveLength(5);
    expect(MANUAL_JOURNAL_ERROR.getFieldErrors("lines[0].glAccountId")).toEqual([
      { field: "lines[0].glAccountId", message: "GL account is required", code: "REQUIRED" },
      { field: "lines[0].glAccountId", message: "GL account 4000 is inactive", code: "INVALID" },
    ]);
  });

  it("getFieldError returns the first message for a field", () => {
    expect(MANUAL_JOURNAL_ERROR.getFieldError("lines[0].glAccountId")?.message).toBe(
      "GL account is required",
    );
  });

  it("headlines the GraphQL message rather than a field message when several fields failed", () => {
    expect(MANUAL_JOURNAL_ERROR.normalize().detail).toBe("Validation failed");
  });

  it("still headlines the field message when exactly one field failed", () => {
    const single = graphQLError(
      validationError([{ field: "reason", message: "Reason is required", code: "REQUIRED" }]),
    );

    expect(single.normalize().detail).toBe("Reason is required");
  });
});

describe("handleMutationError field distribution", () => {
  beforeEach(() => {
    toastError.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("applies the first message when two errors target the same field", () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });

    expect(errorsOf(form.current).lines[0].glAccountId.message).toBe("GL account is required");
  });

  it("agrees with getFieldError and normalize().detail", () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });

    const fromForm = errorsOf(form.current).lines[0].glAccountId.message;
    const fromAccessor = MANUAL_JOURNAL_ERROR.getFieldError("lines[0].glAccountId")?.message;

    expect(fromForm).toBe(fromAccessor);
    // detail is the GraphQL summary here because more than one field failed; the point is
    // that it is no longer an arbitrary field's message contradicting the field itself.
    expect(MANUAL_JOURNAL_ERROR.normalize().detail).toBe("Validation failed");
  });

  it("routes a field the form does not have to root, visibly", () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });

    const errors = errorsOf(form.current);
    expect(errors.root.message).toContain("Only draft journals can be submitted");
    expect(errors.status).toBeUndefined();
    expect(toastError).toHaveBeenCalledWith("This change could not be saved", {
      description: expect.stringContaining("Only draft journals can be submitted"),
    });
  });

  // react-hook-form treats root as submit-transient: handleSubmit discards it and runs
  // onValid. That is the behaviour that makes root the right destination — the message is
  // shown once for the attempt that earned it and never outlives it.
  it("does not let a root error outlive the submit that produced it", async () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });
    expect(errorsOf(form.current).root).toBeDefined();

    const onSubmit = form.current.handleSubmit(
      () => {},
      () => {},
    );
    await act(async () => {
      await onSubmit();
    });

    expect(errorsOf(form.current).root).toBeUndefined();
  });

  // Regression guard for the pathology this routing exists to prevent: an unregistered
  // path that is NOT root blocks every future submit and no input can ever clear it.
  it("would have blocked the form permanently had the path been set directly", async () => {
    const form = renderJournalForm();

    act(() => {
      form.current.setError("status" as never, {
        message: "Only draft journals",
        type: "validation",
      });
    });

    let submitted = false;
    for (const _attempt of [1, 2]) {
      const onSubmit = form.current.handleSubmit(() => {
        submitted = true;
      });
      await act(async () => {
        await onSubmit();
      });
    }

    expect(submitted).toBe(false);
    expect(errorsOf(form.current).status).toBeDefined();
  });

  it("clears the root error so a corrected form can submit again", async () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });
    expect(errorsOf(form.current).root).toBeDefined();

    // What <Form> does at the start of every submit attempt, for submits that bypass
    // handleSubmit and therefore miss react-hook-form's own root reset.
    act(() => {
      form.current.clearErrors("root");
      form.current.clearErrors();
    });

    let submitted = false;
    const onSubmit = form.current.handleSubmit(() => {
      submitted = true;
    });
    await act(async () => {
      await onSubmit();
    });

    expect(errorsOf(form.current).root).toBeUndefined();
    expect(submitted).toBe(true);
  });

  it("routes a non-leaf array-element path to root rather than onto the element", () => {
    const form = renderJournalForm();
    const error = graphQLError(
      validationError([{ field: "lines[0]", message: "Line is required", code: "REQUIRED" }]),
    );

    act(() => {
      handleMutationError({ error, form: form.current });
    });

    const errors = errorsOf(form.current);
    expect(errors.root.message).toBe("Line is required");
    expect(errors.lines).toBeUndefined();
  });

  it("routes an empty field path to root", () => {
    const form = renderJournalForm();
    const error = graphQLError(
      validationError([{ field: "", message: "Journal is unbalanced", code: "INVALID" }]),
    );

    act(() => {
      handleMutationError({ error, form: form.current });
    });

    expect(errorsOf(form.current).root.message).toBe("Journal is unbalanced");
  });

  it("distributes the remaining fields normally", () => {
    const form = renderJournalForm();

    act(() => {
      handleMutationError({ error: MANUAL_JOURNAL_ERROR, form: form.current });
    });

    const errors = errorsOf(form.current);
    expect(errors.reason.message).toBe("Reason is required");
    expect(errors.lines[1].debitAmount.message).toBe("Debit amount cannot be negative");
  });

  it("does not raise a root error when every field resolves", () => {
    const form = renderJournalForm();
    const error = graphQLError(
      validationError([{ field: "reason", message: "Reason is required", code: "REQUIRED" }]),
    );

    act(() => {
      handleMutationError({ error, form: form.current });
    });

    expect(errorsOf(form.current).root).toBeUndefined();
    expect(toastError).not.toHaveBeenCalled();
  });
});
