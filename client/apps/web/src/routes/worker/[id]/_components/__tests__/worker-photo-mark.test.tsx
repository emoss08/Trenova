import { worker as workerQueries } from "@/lib/queries/worker";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ImageCropDialogProps } from "@trenova/shared/components/image-upload/image-crop-dialog";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkerPhotoMark } from "../worker-photo-mark";
import { mariaRecord } from "./worker-fixtures";

const mocks = vi.hoisted(() => ({
  uploadPhoto: vi.fn(),
  removePhoto: vi.fn(),
  toastError: vi.fn(),
  allowed: { value: true },
}));

vi.mock("@/services/worker", () => ({
  workerService: { uploadPhoto: mocks.uploadPhoto, removePhoto: mocks.removePhoto },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed.value, isLoading: false }),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: mocks.toastError, success: vi.fn(), info: vi.fn() },
}));

// The crop dialog needs a browser's image decoding and canvas. It is stood in for by its contract:
// open on the picture it was given, and uploading when confirmed.
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

const PHOTO = {
  url: "https://files.example/wrk_1/photo.jpg?sig=a",
  thumbnailUrl: "https://files.example/wrk_1/thumb.jpg?sig=a",
};

function renderMark(record: WorkerRecord, { allowed = true }: { allowed?: boolean } = {}) {
  mocks.allowed.value = allowed;
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const key = workerQueries.record(record.id).queryKey;
  client.setQueryData(key, record);
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const view = render(
    <QueryClientProvider client={client}>
      <WorkerPhotoMark worker={record} />
    </QueryClientProvider>,
  );
  const input = view.container.querySelector('input[type="file"]') as HTMLInputElement | null;
  const cached = () => client.getQueryData<WorkerRecord>(key);
  return { input, cached, invalidate };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("the worker's photo in their record header", () => {
  it("shows the face alone, with no control, to someone who may not update the worker", () => {
    const { input } = renderMark(mariaRecord({ photo: PHOTO }), { allowed: false });

    expect(screen.queryByRole("button")).toBeNull();
    expect(input).toBeNull();
    expect(screen.getByText("MA")).toBeTruthy();
  });

  it("offers to add a photo when there is none, and uploads the positioned photo with the worker's version", async () => {
    mocks.uploadPhoto.mockResolvedValue({ ...PHOTO, version: 4 });
    const { input, cached, invalidate } = renderMark(mariaRecord({ photo: null }));

    const control = screen.getByRole("button", { name: "Add a photo" });
    expect(control).toHaveTextContent("MA");

    const file = new File(["data"], "maria.png", { type: "image/png" });
    fireEvent.change(input as HTMLInputElement, { target: { files: [file] } });
    const dialog = await screen.findByRole("dialog", { name: "Position Maria Alvarez's photo" });
    expect(dialog).toHaveTextContent("maria.png");

    await userEvent.click(screen.getByRole("button", { name: "Save photo" }));

    await waitFor(() => expect(mocks.uploadPhoto).toHaveBeenCalledWith("wrk_1", file, 3));
    await waitFor(() => expect(cached()?.version).toBe(4));
    expect(cached()?.photo).toEqual(PHOTO);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: workerQueries.record("wrk_1").queryKey,
    });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["worker-list"] });
  });

  it("says what was wrong with a file that cannot be a photo, without opening the crop dialog", () => {
    const { input } = renderMark(mariaRecord({ photo: null }));

    const file = new File(["data"], "notes.pdf", { type: "application/pdf" });
    fireEvent.change(input as HTMLInputElement, { target: { files: [file] } });

    expect(screen.queryByRole("dialog")).toBeNull();
    expect(mocks.toastError).toHaveBeenCalledWith("That file cannot be their photo", {
      description: "Choose a PNG, JPG, WebP or SVG picture.",
    });
  });

  it("takes a dropped photo and marks the face while one is dragged over it", async () => {
    renderMark(mariaRecord({ photo: null }));
    const control = screen.getByRole("button", { name: "Add a photo" });
    const file = new File(["data"], "dropped.jpg", { type: "image/jpeg" });

    const state = () => control.querySelector("[data-state]")?.getAttribute("data-state");
    expect(state()).toBe("idle");

    fireEvent.dragEnter(control, { dataTransfer: { types: ["Files"], files: [file] } });
    expect(state()).toBe("dragging");

    fireEvent.drop(control, { dataTransfer: { types: ["Files"], files: [file] } });

    expect(state()).toBe("idle");
    expect(await screen.findByRole("dialog")).toHaveTextContent("dropped.jpg");
  });

  it("removes the photo from the menu and leaves the initials, with the new version cached", async () => {
    mocks.removePhoto.mockResolvedValue({ url: null, thumbnailUrl: null, version: 5 });
    const { cached } = renderMark(mariaRecord({ photo: PHOTO }));

    await userEvent.click(screen.getByRole("button", { name: "Change photo" }));
    expect(await screen.findByRole("menuitem", { name: "Replace photo" })).toBeTruthy();
    await userEvent.click(screen.getByRole("menuitem", { name: "Remove photo" }));

    await waitFor(() => expect(mocks.removePhoto).toHaveBeenCalledWith("wrk_1", 3));
    await waitFor(() => expect(cached()?.version).toBe(5));
    expect(cached()?.photo).toBeNull();
  });

  it("tells the person when the photo could not be removed and reads the record again", async () => {
    mocks.removePhoto.mockRejectedValue(new Error("stale"));
    const { cached, invalidate } = renderMark(mariaRecord({ photo: PHOTO }));

    await userEvent.click(screen.getByRole("button", { name: "Change photo" }));
    await userEvent.click(await screen.findByRole("menuitem", { name: "Remove photo" }));

    await waitFor(() =>
      expect(mocks.toastError).toHaveBeenCalledWith("The photo could not be removed", {
        description: expect.any(String),
      }),
    );
    expect(cached()?.photo).toEqual(PHOTO);
    expect(cached()?.version).toBe(3);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: workerQueries.record("wrk_1").queryKey,
    });
    expect(screen.getByRole("button", { name: "Change photo" })).not.toBeDisabled();
  });
});
