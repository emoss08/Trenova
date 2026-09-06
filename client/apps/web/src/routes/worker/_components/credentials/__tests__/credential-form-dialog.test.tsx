import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { getTodayDate } from "@trenova/shared/lib/date";
import { suggestExpiryUnix } from "@trenova/shared/lib/credential";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CredentialFormDialog } from "../credential-form-dialog";

const { createWorkerCredential, updateWorkerCredential, fetchActiveWorkerCredentialTypes } =
  vi.hoisted(() => ({
    createWorkerCredential: vi.fn(),
    updateWorkerCredential: vi.fn(),
    fetchActiveWorkerCredentialTypes: vi.fn(),
  }));

vi.mock("@/lib/graphql/worker-credential", () => ({
  createWorkerCredential,
  updateWorkerCredential,
  fetchActiveWorkerCredentialTypes,
  WORKER_CREDENTIALS_KEY: "worker-credentials",
  WORKER_CREDENTIAL_SUMMARY_KEY: "worker-credential-summary",
  WORKER_CREDENTIAL_TYPES_KEY: "worker-credential-types",
  CREDENTIAL_EXPIRY_FORECAST_KEY: "credential-expiry-forecast",
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

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({
    control,
    name,
    label,
    options,
    isReadOnly,
  }: {
    control: Control;
    name: string;
    label: string;
    options: { value: string; label: string }[];
    isReadOnly?: boolean;
  }) => {
    const { field } = useController({ control, name });
    return (
      <select
        aria-label={label}
        disabled={isReadOnly}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      >
        <option value="">—</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
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
  businessUnitId: "bu_1",
  organizationId: "org_1",
  description: null,
  category: "License",
  status: "Active",
  isRequired: true,
  requiredForDriverTypes: [],
  renewalWindowDays: 30,
  requiresDocument: false,
  isSystem: true,
  activeCredentialCount: 0,
  version: 0,
  createdAt: 1,
  updatedAt: 1,
};
const types = [
  {
    ...baseType,
    id: "wct_cdl",
    code: "CDL",
    name: "Commercial Driver's License",
    validityMonths: null,
    requiresNumber: true,
    profileField: "LicenseExpiry",
    sortOrder: 10,
  },
  {
    ...baseType,
    id: "wct_med",
    code: "MED_CARD",
    name: "DOT Medical Card",
    validityMonths: 24,
    requiresNumber: false,
    profileField: "MedicalCardExpiry",
    sortOrder: 20,
  },
  {
    ...baseType,
    id: "wct_fork",
    code: "FORKLIFT",
    name: "Forklift Certification",
    isRequired: false,
    validityMonths: 36,
    requiresNumber: false,
    profileField: null,
    sortOrder: 90,
  },
];

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
    fetchActiveWorkerCredentialTypes.mockResolvedValue(types);
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
    fetchActiveWorkerCredentialTypes.mockResolvedValue(types);
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
    fetchActiveWorkerCredentialTypes.mockResolvedValue(types);
    renderDialog({ credentialTypeId: "wct_cdl" });

    const select = await screen.findByLabelText("Credential type");
    await waitFor(() => expect(select).toHaveValue("wct_cdl"));
    fireEvent.click(screen.getByRole("button", { name: "Add credential" }));

    expect(await screen.findByText("This credential type requires a number")).toBeInTheDocument();
    expect(createWorkerCredential).not.toHaveBeenCalled();
  });

  it("edits in place with the version for optimistic locking", async () => {
    fetchActiveWorkerCredentialTypes.mockResolvedValue(types);
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
