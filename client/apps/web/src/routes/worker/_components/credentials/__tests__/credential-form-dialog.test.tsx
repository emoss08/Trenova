import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { getTodayDate } from "@trenova/shared/lib/date";
import { suggestExpiryUnix } from "@trenova/shared/lib/credential";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CredentialFormDialog } from "../credential-form-dialog";

const { createWorkerCredential, updateWorkerCredential, fetchGraphQLSelectedOption } = vi.hoisted(
  () => ({
    createWorkerCredential: vi.fn(),
    updateWorkerCredential: vi.fn(),
    fetchGraphQLSelectedOption: vi.fn(),
  }),
);

vi.mock("@/lib/graphql/worker-credential", () => ({
  createWorkerCredential,
  updateWorkerCredential,
  WORKER_CREDENTIALS_KEY: "worker-credentials",
  WORKER_CREDENTIAL_SUMMARY_KEY: "worker-credential-summary",
  CREDENTIAL_EXPIRY_FORECAST_KEY: "credential-expiry-forecast",
}));

// The picker reads the selected type back one row at a time, so the fixture is
// the select option the server would return, meta and all.
vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectedOption,
  fetchGraphQLSelectOptions: vi.fn(),
}));

vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: () => ({
    uploads: [],
    uploadFiles: vi.fn(),
    cancelUpload: vi.fn(),
    retryUpload: vi.fn(),
    removeUpload: vi.fn(),
    clearCompleted: vi.fn(),
  }),
}));

vi.mock("@/components/documents/document-upload-zone", () => ({
  DocumentUploadZone: () => <div data-testid="upload-zone" />,
}));

vi.mock("@/components/autocomplete-fields", () => ({
  WorkerCredentialTypeAutocompleteField: ({
    control,
    name,
    label,
    disabled,
  }: {
    control: Control;
    name: string;
    label: string;
    disabled?: boolean;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        disabled={disabled}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({
    control,
    name,
    label,
  }: {
    control: Control;
    name: string;
    label: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        type="number"
        value={(field.value as number) || ""}
        onChange={(event) =>
          field.onChange(event.target.value === "" ? null : Number(event.target.value))
        }
      />
    );
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const baseType = {
  description: null,
  meta: {
    category: "License",
    isRequired: true,
    requiresDocument: false,
  },
};
const types = [
  {
    ...baseType,
    id: "wct_cdl",
    label: "Commercial Driver's License",
    meta: {
      ...baseType.meta,
      code: "CDL",
      validityMonths: null,
      requiresNumber: true,
      profileField: "LicenseExpiry",
    },
  },
  {
    ...baseType,
    id: "wct_med",
    label: "DOT Medical Card",
    meta: {
      ...baseType.meta,
      code: "MED_CARD",
      validityMonths: 24,
      requiresNumber: false,
      profileField: "MedicalCardExpiry",
    },
  },
  {
    ...baseType,
    id: "wct_fork",
    label: "Forklift Certification",
    meta: {
      ...baseType.meta,
      code: "FORKLIFT",
      isRequired: false,
      validityMonths: 36,
      requiresNumber: false,
      profileField: "",
    },
  },
];

function selectedTypeIs(id: string) {
  fetchGraphQLSelectedOption.mockImplementation(async (_resource: string, requestedId: string) =>
    requestedId === id ? (types.find((type) => type.id === id) ?? null) : null,
  );
}

function inputById(name: string): HTMLInputElement {
  const element = document.getElementById(`input-${name}`);
  if (!(element instanceof HTMLInputElement)) {
    throw new Error(`input-${name} not rendered`);
  }
  return element;
}

function renderDialog(props: Partial<React.ComponentProps<typeof CredentialFormDialog>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <CredentialFormDialog
        open
        onOpenChange={onOpenChange}
        workerId="wrk_1"
        mode="create"
        {...props}
      />
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("CredentialFormDialog", () => {
  it("prefills issue date today and the expiry from the type's validity", async () => {
    selectedTypeIs("wct_fork");
    createWorkerCredential.mockResolvedValue({ id: "wcred_new" });
    const today = getTodayDate();
    const { onOpenChange } = renderDialog({ credentialTypeId: "wct_fork" });

    await waitFor(() =>
      expect(screen.getByLabelText("Expires")).toHaveValue(suggestExpiryUnix(today, 36)),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add credential" }));

    await waitFor(() => expect(createWorkerCredential).toHaveBeenCalledTimes(1));
    expect(createWorkerCredential).toHaveBeenCalledWith({
      workerId: "wrk_1",
      credentialTypeId: "wct_fork",
      number: null,
      issuingAuthority: null,
      issuedAt: today,
      expiresAt: suggestExpiryUnix(today, 36),
      documentId: null,
      notes: null,
      renew: false,
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("renews with the type locked and the renew flag set", async () => {
    selectedTypeIs("wct_med");
    createWorkerCredential.mockResolvedValue({ id: "wcred_new" });
    renderDialog({
      mode: "renew",
      credentialTypeId: "wct_med",
      credential: {
        id: "wcred_old",
        number: null,
        issuingAuthority: "Dr. Who",
        version: 3,
      } as never,
    });

    const select = await screen.findByLabelText("Credential type");
    await waitFor(() => expect(select).toHaveValue("wct_med"));
    expect(select).toBeDisabled();
    expect(inputById("issuingAuthority")).toHaveValue("Dr. Who");

    fireEvent.click(screen.getByRole("button", { name: "Renew credential" }));
    await waitFor(() => expect(createWorkerCredential).toHaveBeenCalledTimes(1));
    expect(createWorkerCredential.mock.calls[0][0]).toMatchObject({
      credentialTypeId: "wct_med",
      issuingAuthority: "Dr. Who",
      renew: true,
    });
  });

  it("refuses to save a type that requires a number without one", async () => {
    selectedTypeIs("wct_cdl");
    renderDialog({ credentialTypeId: "wct_cdl" });

    const select = await screen.findByLabelText("Credential type");
    await waitFor(() => expect(select).toHaveValue("wct_cdl"));
    fireEvent.click(screen.getByRole("button", { name: "Add credential" }));

    expect(await screen.findByText("This credential type requires a number")).toBeInTheDocument();
    expect(createWorkerCredential).not.toHaveBeenCalled();
  });

  it("edits in place with the version for optimistic locking", async () => {
    selectedTypeIs("wct_fork");
    updateWorkerCredential.mockResolvedValue({ id: "wcred_1" });
    renderDialog({
      mode: "edit",
      credential: {
        id: "wcred_1",
        credentialTypeId: "wct_fork",
        number: "F-9",
        issuingAuthority: null,
        issuedAt: 1_700_000_000,
        expiresAt: 1_800_000_000,
        documentId: "doc_1",
        notes: "class IV",
        version: 4,
      } as never,
    });

    const select = await screen.findByLabelText("Credential type");
    await waitFor(() => expect(select).toHaveValue("wct_fork"));
    expect(select).toBeDisabled();
    fireEvent.change(inputById("number"), { target: { value: "F-10" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(updateWorkerCredential).toHaveBeenCalledTimes(1));
    expect(updateWorkerCredential).toHaveBeenCalledWith({
      id: "wcred_1",
      number: "F-10",
      issuingAuthority: null,
      issuedAt: 1_700_000_000,
      expiresAt: 1_800_000_000,
      documentId: "doc_1",
      notes: "class IV",
      version: 4,
    });
    expect(createWorkerCredential).not.toHaveBeenCalled();
  });
});
