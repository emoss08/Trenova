import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { FormProvider, useForm, useFormContext, type UseFormReturn } from "react-hook-form";
import { Form } from "@trenova/shared/components/ui/form";
import { GraphQLRequestError, type NormalizedGraphQLError } from "@trenova/shared/lib/graphql";
import { handleMutationError, type MutationFormTarget } from "@/hooks/use-api-mutation";

const PROBLEM_BASE = "https://api.trenova.app/problems/";

const toastError = vi.hoisted(() => vi.fn());
vi.mock("sonner", () => ({ toast: { error: toastError, success: vi.fn() } }));

type FieldError = { field: string; message: string; code?: string };

function validationError(fieldErrors: FieldError[]): GraphQLRequestError {
  const graphQLError: NormalizedGraphQLError = {
    extensions: {
      code: "REQUIRED",
      type: `${PROBLEM_BASE}validation-error`,
      errors: fieldErrors,
    },
    code: "REQUIRED",
    errors: fieldErrors,
    message: "Validation failed",
    type: `${PROBLEM_BASE}validation-error`,
  };

  return new GraphQLRequestError({
    graphQLErrors: [graphQLError],
    kind: "graphql",
    message: "Validation failed",
    status: 200,
  });
}

function errorsOf(form: MutationFormTarget<never>) {
  return (form.control as unknown as { _formState: { errors: Record<string, any> } })._formState
    .errors;
}

// ── The three form shapes in production ───────────────────────────────────────────
//
// Each captures its own UseFormReturn into `captured` so the test can drive
// handleMutationError with the same object the real call site passes.

type Shape = {
  name: string;
  // The registered leaf both this shape and the server agree on.
  leaf: string;
  // A path the server can emit that this shape has no input for.
  nested?: string;
  mount: () => UseFormReturn<any>;
};

function mountOwnUseForm(): UseFormReturn<any> {
  let captured!: UseFormReturn<any>;

  function OwnForm() {
    const form = useForm({ defaultValues: { reason: "", currencyCode: "USD" } });
    captured = form;
    return (
      <FormProvider {...form}>
        <Form onSubmit={() => undefined}>
          <input aria-label="reason" {...form.register("reason")} />
          <input aria-label="currencyCode" {...form.register("currencyCode")} />
        </Form>
      </FormProvider>
    );
  }

  render(<OwnForm />);
  return captured;
}

// Mirrors the 23 forms whose fields are registered by a child reached through a parent's
// FormProvider — the shape where inline root rendering depends on the tree, not the caller.
function mountParentProvider(): UseFormReturn<any> {
  let captured!: UseFormReturn<any>;

  function Fields() {
    const form = useFormContext();
    return (
      <>
        <input aria-label="reason" {...form.register("reason")} />
        <input aria-label="currencyCode" {...form.register("currencyCode")} />
      </>
    );
  }

  function Parent() {
    const form = useForm({ defaultValues: { reason: "", currencyCode: "USD" } });
    captured = form;
    return (
      <FormProvider {...form}>
        <Form onSubmit={() => undefined}>
          <Fields />
        </Form>
      </FormProvider>
    );
  }

  render(<Parent />);
  return captured;
}

// The manual-journal shape: a field array, where the server addresses leaves as
// lines[0].glAccountId and whole elements as lines[0].
function mountNestedArray(): UseFormReturn<any> {
  let captured!: UseFormReturn<any>;

  function JournalForm() {
    const form = useForm({
      defaultValues: {
        reason: "",
        currencyCode: "USD",
        lines: [
          { glAccountId: "", debitAmount: 0 },
          { glAccountId: "", debitAmount: 0 },
        ],
      },
    });
    captured = form;
    return (
      <FormProvider {...form}>
        <Form onSubmit={() => undefined}>
          <input aria-label="reason" {...form.register("reason")} />
          <input aria-label="currencyCode" {...form.register("currencyCode")} />
          {[0, 1].map((index) => (
            <input
              key={index}
              aria-label={`lines.${index}.glAccountId`}
              {...form.register(`lines.${index}.glAccountId` as const)}
            />
          ))}
        </Form>
      </FormProvider>
    );
  }

  render(<JournalForm />);
  return captured;
}

const SHAPES: Shape[] = [
  { name: "form owning its own useForm", leaf: "reason", mount: mountOwnUseForm },
  { name: "fields registered through a parent FormProvider", leaf: "reason", mount: mountParentProvider },
  {
    name: "form with a nested field array",
    leaf: "reason",
    nested: "lines[0].glAccountId",
    mount: mountNestedArray,
  },
];

describe.each(SHAPES)("handleMutationError contract — $name", (shape) => {
  beforeEach(() => {
    toastError.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("puts a registered leaf's message on that input", () => {
    const form = shape.mount();

    handleMutationError({
      error: validationError([{ field: shape.leaf, message: "Reason is required" }]),
      form,
    });

    expect(errorsOf(form)[shape.leaf].message).toBe("Reason is required");
    expect(errorsOf(form).root).toBeUndefined();
  });

  it("routes an unregistered path to the root error and shows it", async () => {
    const form = shape.mount();

    handleMutationError({
      error: validationError([
        { field: "status", message: "Only draft journals can be submitted" },
      ]),
      form,
    });

    expect(errorsOf(form).root.message).toBe("Only draft journals can be submitted");
    expect(errorsOf(form).status).toBeUndefined();
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Only draft journals can be submitted",
    );
  });

  it("routes a non-leaf path to the root error rather than onto the node", () => {
    const form = shape.mount();

    handleMutationError({
      error: validationError([{ field: "lines[0]", message: "Line is required" }]),
      form,
    });

    expect(errorsOf(form).root.message).toBe("Line is required");
    expect(errorsOf(form).lines).toBeUndefined();
  });

  it("routes an empty field path to the root error", () => {
    const form = shape.mount();

    handleMutationError({
      error: validationError([{ field: "", message: "Journal is unbalanced" }]),
      form,
    });

    expect(errorsOf(form).root.message).toBe("Journal is unbalanced");
  });

  it("applies the first message when the same field is reported twice", () => {
    const form = shape.mount();
    const error = validationError([
      { field: shape.leaf, message: "Reason is required", code: "REQUIRED" },
      { field: shape.leaf, message: "Reason is too long", code: "INVALID_LENGTH" },
    ]);

    handleMutationError({ error, form });

    // All three accessors must agree, which is the whole point of first-wins.
    expect(errorsOf(form)[shape.leaf].message).toBe("Reason is required");
    expect(error.getFieldError(shape.leaf)?.message).toBe("Reason is required");
    expect(error.getFieldErrors(shape.leaf)).toHaveLength(2);
  });

  it("distributes several fields at once and keeps unknowns at the root", () => {
    const form = shape.mount();

    handleMutationError({
      error: validationError([
        { field: shape.leaf, message: "Reason is required" },
        { field: "currencyCode", message: "Currency is not supported" },
        { field: "status", message: "Only draft journals can be submitted" },
      ]),
      form,
    });

    const errors = errorsOf(form);
    expect(errors[shape.leaf].message).toBe("Reason is required");
    expect(errors.currencyCode.message).toBe("Currency is not supported");
    expect(errors.root.message).toBe("Only draft journals can be submitted");
  });
});

describe("handleMutationError contract — nested array leaves", () => {
  it("addresses a bracket-indexed leaf that the form registered with dot indices", () => {
    const form = mountNestedArray();

    handleMutationError({
      error: validationError([
        { field: "lines[0].glAccountId", message: "GL account is required" },
        { field: "lines[1].glAccountId", message: "GL account 4000 is inactive" },
      ]),
      form,
    });

    const errors = errorsOf(form);
    expect(errors.lines[0].glAccountId.message).toBe("GL account is required");
    expect(errors.lines[1].glAccountId.message).toBe("GL account 4000 is inactive");
    expect(errors.root).toBeUndefined();
  });

  it("keeps a leaf the form does not register at the root", () => {
    const form = mountNestedArray();

    handleMutationError({
      error: validationError([
        { field: "lines[0].debitAmount", message: "Debit amount cannot be negative" },
      ]),
      form,
    });

    // debitAmount exists in defaultValues but no input registers it, so it is not a leaf.
    expect(errorsOf(form).root.message).toBe("Debit amount cannot be negative");
  });
});
