import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ImageCropDialogProps } from "../image-crop-dialog";
import type { ImageSpec } from "../image-spec";
import { useImageUpload } from "../use-image-upload";

// The crop dialog needs a browser's image decoding, layout and canvas, none of which this
// environment has. It is stood in for by its contract: it is open with the picture it was
// given, shows the problem it was told about, and uploads when confirmed.
vi.mock("../image-crop-dialog", () => ({
  ImageCropDialog: ({ open, file, title, problem, onConfirm, onClose }: ImageCropDialogProps) =>
    open && file ? (
      <div role="dialog" aria-label={title}>
        <span>{file.name}</span>
        {problem ? <span role="alert">{problem}</span> : null}
        <button type="button" onClick={() => void onConfirm(file).then(onClose)}>
          Confirm
        </button>
      </div>
    ) : null,
}));

const LOGO: ImageSpec = { mask: "rect", frame: "source", fit: "contain", maxDimension: 1024 };

function fileOf(name: string, type: string): File {
  return new File(["data"], name, { type });
}

/** A control built on the hook the way a screen builds one: a drop target that says what was wrong. */
function Control({
  onUpload,
  disabled,
}: {
  onUpload: (file: File) => Promise<void>;
  disabled?: boolean;
}) {
  const [problem, setProblem] = useState<string | null>(null);
  const upload = useImageUpload({
    spec: LOGO,
    title: "Position your logo",
    confirmLabel: "Upload logo",
    disabled,
    onUpload,
    onProblem: setProblem,
  });

  return (
    <div data-testid="target" data-dragging={upload.dragging} {...upload.dropTarget}>
      {problem ? <p role="alert">{problem}</p> : null}
      {upload.element}
    </div>
  );
}

function renderControl(props: { disabled?: boolean } = {}) {
  const onUpload = vi.fn<(file: File) => Promise<void>>().mockResolvedValue(undefined);
  const view = render(<Control onUpload={onUpload} {...props} />);
  const input = view.container.querySelector('input[type="file"]') as HTMLInputElement;
  const choose = (...files: File[]) => fireEvent.change(input, { target: { files } });
  return { onUpload, choose, target: screen.getByTestId("target") };
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("choosing a picture", () => {
  it("opens the crop dialog on the picture and uploads it once confirmed", async () => {
    const logo = fileOf("logo.png", "image/png");
    const { onUpload, choose } = renderControl();

    choose(logo);

    const dialog = await screen.findByRole("dialog", { name: "Position your logo" });
    expect(dialog).toHaveTextContent("logo.png");

    await userEvent.click(screen.getByRole("button", { name: "Confirm" }));

    expect(onUpload).toHaveBeenCalledExactlyOnceWith(logo);
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("takes an SVG", async () => {
    const { choose } = renderControl();

    choose(fileOf("mark.svg", "image/svg+xml"));

    expect(await screen.findByRole("dialog", { name: "Position your logo" })).toHaveTextContent(
      "mark.svg",
    );
  });

  it("says what was wrong with a file that is not a picture, and opens nothing", () => {
    const { onUpload, choose } = renderControl();

    choose(fileOf("rates.pdf", "application/pdf"));

    expect(screen.getByRole("alert")).toHaveTextContent("Choose a PNG, JPG, WebP or SVG picture.");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(onUpload).not.toHaveBeenCalled();
  });

  it("says a picture is too large, naming the limit", () => {
    const photo = fileOf("photo.jpg", "image/jpeg");
    Object.defineProperty(photo, "size", { value: 20 * 1024 * 1024 + 1 });
    const { choose } = renderControl();

    choose(photo);

    expect(screen.getByRole("alert")).toHaveTextContent(
      "That picture is over 20 MB. Choose a smaller one.",
    );
  });

  it("clears the complaint once a picture is taken", async () => {
    const { choose } = renderControl();

    choose(fileOf("rates.pdf", "application/pdf"));
    choose(fileOf("logo.png", "image/png"));

    await screen.findByRole("dialog", { name: "Position your logo" });
    expect(screen.queryByText("Choose a PNG, JPG, WebP or SVG picture.")).not.toBeInTheDocument();
  });

  it("tells the open dialog, not the control, about a replacement it cannot use", async () => {
    const { choose } = renderControl();

    choose(fileOf("logo.png", "image/png"));
    const dialog = await screen.findByRole("dialog", { name: "Position your logo" });
    choose(fileOf("rates.pdf", "application/pdf"));

    expect(dialog).toHaveTextContent("logo.png");
    expect(dialog).toContainElement(screen.getByRole("alert"));
  });
});

describe("dropping and pasting a picture", () => {
  it("takes the first picture among the files dropped on the target", async () => {
    const { target } = renderControl();

    fireEvent.drop(target, {
      dataTransfer: {
        types: ["Files"],
        files: [fileOf("notes.txt", "text/plain"), fileOf("logo.webp", "image/webp")],
      },
    });

    expect(await screen.findByRole("dialog", { name: "Position your logo" })).toHaveTextContent(
      "logo.webp",
    );
  });

  it("says a file is over the target while one is, and stops once it leaves", () => {
    const { target } = renderControl();
    const dataTransfer = { types: ["Files"], files: [] };

    fireEvent.dragEnter(target, { dataTransfer });
    expect(target).toHaveAttribute("data-dragging", "true");

    fireEvent.dragLeave(target, { dataTransfer });
    expect(target).toHaveAttribute("data-dragging", "false");
  });

  it("ignores a drag that carries no file, such as selected text", () => {
    const { target } = renderControl();

    fireEvent.dragEnter(target, { dataTransfer: { types: ["text/plain"], files: [] } });

    expect(target).toHaveAttribute("data-dragging", "false");
  });

  it("takes a picture pasted into the target", async () => {
    const { target } = renderControl();

    fireEvent.paste(target, { clipboardData: { files: [fileOf("shot.png", "image/png")] } });

    expect(await screen.findByRole("dialog", { name: "Position your logo" })).toHaveTextContent(
      "shot.png",
    );
  });

  it("takes nothing while it is disabled", () => {
    const { target } = renderControl({ disabled: true });

    fireEvent.drop(target, {
      dataTransfer: { types: ["Files"], files: [fileOf("logo.png", "image/png")] },
    });

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
