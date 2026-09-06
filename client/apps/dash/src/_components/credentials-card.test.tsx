import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CredentialsCard } from "./credentials-card";

const { fetchMyCredentials, uploadMyCredentialDocument } = vi.hoisted(() => ({
  fetchMyCredentials: vi.fn(),
  uploadMyCredentialDocument: vi.fn(),
}));
const features = vi.hoisted(() => ({ allowProfileDocumentUpload: true }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyCredentials,
}));

vi.mock("@trenova/shared/lib/portal", () => ({
  uploadMyCredentialDocument,
}));

vi.mock("./use-dash-features", () => ({
  useDashFeatures: () => ({ allowProfileDocumentUpload: features.allowProfileDocumentUpload }),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const rows = [
  {
    id: "wcred_cdl",
    credentialTypeId: "wct_cdl",
    name: "Commercial Driver's License",
    category: "License",
    health: "Valid",
    daysUntilExpiry: 400,
    expiresAt: 1_800_000_000,
    numberMasked: "••••4567",
    required: true,
    verified: true,
    requiresDocument: true,
    documentId: "doc_1",
  },
  {
    id: "wcred_med",
    credentialTypeId: "wct_med",
    name: "DOT Medical Card",
    category: "Medical",
    health: "ExpiringSoon",
    daysUntilExpiry: 9,
    expiresAt: 1_700_800_000,
    numberMasked: "",
    required: true,
    verified: false,
    requiresDocument: true,
    documentId: null,
  },
  {
    id: null,
    credentialTypeId: "wct_mvr",
    name: "Motor Vehicle Record Review",
    category: "Background",
    health: "Missing",
    daysUntilExpiry: null,
    expiresAt: null,
    numberMasked: "",
    required: true,
    verified: false,
    requiresDocument: false,
    documentId: null,
  },
  {
    id: "wcred_fork",
    credentialTypeId: "wct_fork",
    name: "Forklift Certification",
    category: "Certification",
    health: "Expired",
    daysUntilExpiry: -3,
    expiresAt: 1_699_000_000,
    numberMasked: "",
    required: false,
    verified: false,
    requiresDocument: true,
    documentId: null,
  },
];

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <CredentialsCard />
    </QueryClientProvider>,
  );
  return queryClient;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  features.allowProfileDocumentUpload = true;
});

describe("CredentialsCard", () => {
  it("lists every slot worst-first with a plain-language status", async () => {
    fetchMyCredentials.mockResolvedValue(rows);
    renderCard();

    const items = await screen.findAllByTestId(/^dash-credential-(?!file-)/);
    expect(items.map((item) => item.getAttribute("data-testid"))).toEqual([
      "dash-credential-wct_mvr",
      "dash-credential-wct_fork",
      "dash-credential-wct_med",
      "dash-credential-wct_cdl",
    ]);

    const med = screen.getByTestId("dash-credential-wct_med");
    expect(within(med).getByText("9 days left")).toBeInTheDocument();
    const cdl = screen.getByTestId("dash-credential-wct_cdl");
    expect(within(cdl).getByText("••••4567")).toBeInTheDocument();
    expect(within(cdl).getByText("Verified")).toBeInTheDocument();
    expect(screen.getByText("3 need attention")).toBeInTheDocument();
  });

  it("offers upload-to-renew on expiring and expired credentials only", async () => {
    fetchMyCredentials.mockResolvedValue(rows);
    renderCard();

    const med = await screen.findByTestId("dash-credential-wct_med");
    expect(
      within(med).getByRole("button", { name: "Upload renewed DOT Medical Card" }),
    ).toBeInTheDocument();
    const fork = screen.getByTestId("dash-credential-wct_fork");
    expect(
      within(fork).getByRole("button", { name: "Upload renewed Forklift Certification" }),
    ).toBeInTheDocument();
    const cdl = screen.getByTestId("dash-credential-wct_cdl");
    expect(within(cdl).queryByRole("button", { name: /Upload renewed/ })).toBeNull();
    const mvr = screen.getByTestId("dash-credential-wct_mvr");
    expect(within(mvr).queryByRole("button", { name: /Upload renewed/ })).toBeNull();
  });

  it("uploads the chosen file against the credential and refreshes", async () => {
    fetchMyCredentials.mockResolvedValue(rows);
    uploadMyCredentialDocument.mockResolvedValue({ id: "doc_9" });
    renderCard();

    const med = await screen.findByTestId("dash-credential-wct_med");
    fireEvent.click(within(med).getByRole("button", { name: "Upload renewed DOT Medical Card" }));
    const input = within(med).getByTestId("dash-credential-file-wct_med") as HTMLInputElement;
    const file = new File(["img"], "card.jpg", { type: "image/jpeg" });
    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() =>
      expect(uploadMyCredentialDocument).toHaveBeenCalledExactlyOnceWith("wcred_med", file),
    );
    await waitFor(() => expect(fetchMyCredentials).toHaveBeenCalledTimes(2));
  });

  it("hides uploads when the carrier disabled profile uploads", async () => {
    features.allowProfileDocumentUpload = false;
    fetchMyCredentials.mockResolvedValue(rows);
    renderCard();

    await screen.findByTestId("dash-credential-wct_med");
    expect(screen.queryByRole("button", { name: /Upload renewed/ })).toBeNull();
  });
});
