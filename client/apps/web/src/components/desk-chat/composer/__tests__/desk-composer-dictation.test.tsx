import { FakeRecognition } from "@/components/assistant/__tests__/dictation-fakes";
import { clearSpeechServiceMissing, markSpeechServiceMissing } from "@/lib/speech-service";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { DeskComposer } from "../desk-composer";

type SpeechWindow = { webkitSpeechRecognition?: unknown };

const NO_SERVICE = "Dictation isn't available in this browser";

beforeEach(() => {
  FakeRecognition.instances = [];
  (window as SpeechWindow).webkitSpeechRecognition = FakeRecognition;
});

afterEach(() => {
  cleanup();
  delete (window as SpeechWindow).webkitSpeechRecognition;
  clearSpeechServiceMissing();
});

function Harness() {
  const [value, setValue] = useState("");

  return (
    <DeskComposer
      value={value}
      onChange={setValue}
      onSend={() => undefined}
      agent={null}
      busy={false}
    />
  );
}

function renderComposer() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Harness />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/** Every attempt fails at once on the network, as in a browser with no speech service. */
function failEveryAttempt() {
  for (let attempt = 0; attempt < 3; attempt += 1) {
    act(() => {
      FakeRecognition.latest().emitError("network");
      FakeRecognition.latest().emitEnd();
    });
  }
}

/**
 * Helium, Brave and Arc have the speech API and nothing behind it. The first
 * attempt says so above the composer, and the microphone goes, here and on
 * the next visit, rather than starting and stopping on every click.
 */
describe("DeskComposer in a browser with no speech service", () => {
  it("says why above the composer and takes the microphone away", () => {
    const { container } = renderComposer();

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));
    failEveryAttempt();

    const notice = screen.getByText(NO_SERVICE).closest(".dk-dnote-w");
    const box = container.querySelector(".dk-cmp");
    expect(notice).not.toBeNull();
    expect(box?.contains(notice as Node)).toBe(false);
    expect(notice?.nextElementSibling).toBe(box);
    expect(screen.queryByRole("button", { name: "Dictate" })).toBeNull();
  });

  it("puts the notice away when dismissed, and leaves the microphone gone", () => {
    renderComposer();

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));
    failEveryAttempt();
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));

    expect(screen.queryByText(NO_SERVICE)).toBeNull();
    expect(screen.queryByRole("button", { name: "Dictate" })).toBeNull();
  });

  it("shows no microphone where an earlier visit found no speech service", () => {
    markSpeechServiceMissing();
    renderComposer();

    expect(screen.queryByRole("button", { name: "Dictate" })).toBeNull();
    expect(screen.queryByText(NO_SERVICE)).toBeNull();
  });

  it("shows no microphone in a browser without the speech API", () => {
    delete (window as SpeechWindow).webkitSpeechRecognition;
    renderComposer();

    expect(screen.queryByRole("button", { name: "Dictate" })).toBeNull();
  });

  it("says why when a dictation stops for another reason, and keeps the microphone", () => {
    renderComposer();

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));
    act(() => {
      FakeRecognition.latest().emitError("audio-capture");
      FakeRecognition.latest().emitEnd();
    });

    expect(screen.getByText("No microphone was found.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dictate" })).toBeInTheDocument();
  });
});
