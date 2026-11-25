import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ImageCropDialogProps } from "@trenova/shared/components/image-upload/image-crop-dialog";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProfilePhoto } from "./profile-photo";

const mocks = vi.hoisted(() => ({
  fetchMyProfileChangeRequests: vi.fn(),
  withdrawMyProfileChange: vi.fn(),
  submitMyPhoto: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
  features: { allowContactInfoEdit: true },
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyProfileChangeRequests: mocks.fetchMyProfileChangeRequests,
  withdrawMyProfileChange: mocks.withdrawMyProfileChange,
}));

vi.mock("@trenova/shared/lib/portal", () => ({ submitMyPhoto: mocks.submitMyPhoto }));

vi.mock("./use-dash-features", () => ({ useDashFeatures: () => mocks.features }));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));

// The crop dialog needs a browser's image decoding and canvas; it is stood in for by its contract.
vi.mock("@trenova/shared/components/image-upload/image-crop-dialog", () => ({
  ImageCropDialog: ({
    open,
    file,
    title,
    confirmLabel,
    onConfirm,
    onClose,
  }: ImageCropDialogProps) =>
    open && file ? (
      <div role="dialog" aria-label={title}>
        <span>{file.name}</span>
        <button type="button" onClick={() => void onConfirm(file).then(onClose)}>
          {confirmLabel}
        </button>
      </div>
    ) : null,
}));

const PROPOSED = {
  url: "https://files.example/pcr_9/photo.jpg?sig=b",
  thumbnailUrl: "https://files.example/pcr_9/thumb.jpg?sig=b",
};

// Shaped after MyProfileChangeRequests: newest first; a photo request names the field "photo".
function photoRequest(overrides: Record<string, unknown> = {}) {
  return {
    id: "pcr_9",
    status: "Pending",
    note: null,
    submittedAt: 1_795_000_000,
    decidedAt: null,
    decisionNote: null,
    changes: [{ field: "photo", label: "Photo", from: "", to: "" }],
    proposedPhoto: PROPOSED,
    ...overrides,
  };
}

const contactRequest = {
  id: "pcr_8",
  status: "Pending",
  note: null,
  submittedAt: 1_794_000_000,
  decidedAt: null,
  decisionNote: null,
  changes: [{ field: "city", label: "City", from: "Joliet", to: "Aurora" }],
  proposedPhoto: null,
};

function renderPhoto({ allowEdit = true }: { allowEdit?: boolean } = {}) {
  mocks.features.allowContactInfoEdit = allowEdit;
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <ProfilePhoto
        person={{ firstName: "Maria", lastName: "Alvarez", photo: null }}
        profileReadAt={0}
      >
        <p>Maria Alvarez</p>
      </ProfilePhoto>
    </QueryClientProvider>,
  );
  const input = () => view.container.querySelector('input[type="file"]') as HTMLInputElement;
  return { input };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("the driver's photo on their Dash profile", () => {
  it("sends a positioned photo for approval and shows it waiting on the office", async () => {
    mocks.fetchMyProfileChangeRequests
      .mockResolvedValueOnce([contactRequest])
      .mockResolvedValue([photoRequest(), contactRequest]);
    mocks.submitMyPhoto.mockResolvedValue({
      id: "pcr_9",
      status: "Pending",
      submittedAt: 1_795_000_000,
      proposedPhoto: PROPOSED,
    });
    const { input } = renderPhoto();

    const control = await screen.findByRole("button", { name: "Add your photo" });
    expect(control).toHaveTextContent("MA");
    const file = new File(["data"], "me.jpg", { type: "image/jpeg" });
    fireEvent.change(input(), { target: { files: [file] } });

    expect(await screen.findByRole("dialog", { name: "Position your photo" })).toHaveTextContent(
      "me.jpg",
    );
    fireEvent.click(screen.getByRole("button", { name: "Send for approval" }));

    await waitFor(() => expect(mocks.submitMyPhoto).toHaveBeenCalledWith(file));
    expect(await screen.findByText("Waiting on the office")).toBeTruthy();
    fireEvent.load(
      document.querySelector(`img[src="${PROPOSED.thumbnailUrl}"]`) as HTMLImageElement,
    );
    expect(screen.getByRole("img", { name: "The photo you sent" }).getAttribute("src")).toBe(
      PROPOSED.thumbnailUrl,
    );
    expect(screen.getByText("Your new photo shows here once the office approves it.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add your photo" })).toBeNull();
  });

  it("hides the control when the carrier does not let drivers edit their details", async () => {
    mocks.fetchMyProfileChangeRequests.mockResolvedValue([]);
    const { input } = renderPhoto({ allowEdit: false });

    await waitFor(() => expect(mocks.fetchMyProfileChangeRequests).toHaveBeenCalled());
    expect(screen.queryByRole("button", { name: "Add your photo" })).toBeNull();
    expect(input()).toBeNull();
    expect(screen.getByText("MA")).toBeTruthy();
  });

  it("withdraws a waiting photo request", async () => {
    mocks.fetchMyProfileChangeRequests.mockResolvedValue([photoRequest()]);
    mocks.withdrawMyProfileChange.mockResolvedValue({ id: "pcr_9" });
    renderPhoto();

    fireEvent.click(await screen.findByRole("button", { name: "Withdraw" }));

    await waitFor(() => expect(mocks.withdrawMyProfileChange).toHaveBeenCalledWith("pcr_9"));
  });

  it("says when the office turned the last photo down, with its reason", async () => {
    mocks.fetchMyProfileChangeRequests.mockResolvedValue([
      photoRequest({
        status: "Rejected",
        decidedAt: 1_795_100_000,
        decisionNote: "Face the camera",
        proposedPhoto: null,
      }),
    ]);
    renderPhoto();

    expect(await screen.findByText("Not approved")).toBeTruthy();
    expect(screen.getByText("Your carrier said: Face the camera")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add your photo" })).toBeTruthy();
  });
});
