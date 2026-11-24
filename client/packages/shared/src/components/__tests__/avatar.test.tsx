import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { onExpiredPhoto, resetExpiredPhotos } from "../../lib/photo-expiry";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";

afterEach(() => {
  cleanup();
  resetExpiredPhotos();
});

function face(src: string | undefined) {
  return (
    <Avatar data-testid="avatar">
      <AvatarImage src={src} alt="Maria Alvarez" />
      <AvatarFallback>MA</AvatarFallback>
    </Avatar>
  );
}

function renderAvatar(src: string | undefined) {
  const view = render(face(src));
  const image = () => view.container.querySelector("img");
  const images = () => Array.from(view.container.querySelectorAll("img"));
  return { image, images, rerender: (next: string | undefined) => view.rerender(face(next)) };
}

describe("an avatar's picture", () => {
  it("is a native lazy image, hidden behind the initials until it has loaded", () => {
    const { image } = renderAvatar("https://files.example/face.jpg");

    const img = image();
    expect(img?.getAttribute("loading")).toBe("lazy");
    expect(img?.getAttribute("decoding")).toBe("async");
    expect(img?.getAttribute("aria-hidden")).toBe("true");
    expect(img?.getAttribute("alt")).toBe("");
    expect(img?.className).toContain("opacity-0");
    expect(img?.className).toContain("absolute");
    expect(screen.getByText("MA")).toBeTruthy();
    expect(screen.queryByRole("img")).toBeNull();
  });

  it("takes the initials' place once it has loaded, in the same box", () => {
    const { image } = renderAvatar("https://files.example/face.jpg");

    fireEvent.load(image() as HTMLImageElement);

    expect(screen.queryByText("MA")).toBeNull();
    expect(screen.getByRole("img", { name: "Maria Alvarez" })).toBe(image());
    expect(image()?.className).not.toContain("opacity-0");
    expect(screen.getByTestId("avatar").getAttribute("data-image-status")).toBe("loaded");
  });

  it("goes away and leaves the initials when it fails", () => {
    const { image } = renderAvatar("https://files.example/missing.jpg");

    fireEvent.error(image() as HTMLImageElement);

    expect(image()).toBeNull();
    expect(screen.getByText("MA")).toBeTruthy();
  });

  it("draws nothing without a source, so only the initials show", () => {
    const { image } = renderAvatar(undefined);

    expect(image()).toBeNull();
    expect(screen.getByText("MA")).toBeTruthy();
  });

  it("keeps the loaded picture on screen while the same picture, signed again, loads", () => {
    const first = "https://files.example/face.jpg?sig=a";
    const again = "https://files.example/face.jpg?sig=b";
    const { images, rerender } = renderAvatar(first);
    fireEvent.load(images()[0]);

    rerender(again);

    const [shown, loading] = images();
    expect(shown.getAttribute("src")).toBe(first);
    expect(shown.className).not.toContain("opacity-0");
    expect(loading.getAttribute("src")).toBe(again);
    expect(loading.className).toContain("opacity-0");
    expect(screen.queryByText("MA")).toBeNull();

    fireEvent.load(loading);

    expect(images().map((img) => img.getAttribute("src"))).toEqual([again]);
  });

  it("keeps the loaded picture when its re-signed link fails, and reports that link once", () => {
    const reported = vi.fn();
    onExpiredPhoto(reported);
    const first = "https://files.example/face.jpg?sig=a";
    const again = "https://files.example/face.jpg?sig=b";
    const { images, rerender } = renderAvatar(first);
    fireEvent.load(images()[0]);
    rerender(again);

    fireEvent.error(images()[1]);

    expect(images().map((img) => img.getAttribute("src"))).toEqual([first]);
    expect(screen.queryByText("MA")).toBeNull();
    expect(reported).toHaveBeenCalledTimes(1);
    expect(reported).toHaveBeenCalledWith(again);
  });

  it("falls back to the initials when a different picture fails", () => {
    const { images, rerender } = renderAvatar("https://files.example/old.jpg?sig=a");
    fireEvent.load(images()[0]);
    rerender("https://files.example/new.jpg?sig=a");

    fireEvent.error(images()[1]);

    expect(images()).toEqual([]);
    expect(screen.getByText("MA")).toBeTruthy();
  });

  it("reports a failed link to the refreshers once, however often it fails", () => {
    const reported = vi.fn();
    onExpiredPhoto(reported);
    const link = "https://files.example/face.jpg?sig=a";

    const first = renderAvatar(link);
    fireEvent.error(first.image() as HTMLImageElement);
    cleanup();
    const second = renderAvatar(link);
    fireEvent.error(second.image() as HTMLImageElement);

    expect(reported).toHaveBeenCalledTimes(1);
  });
});
