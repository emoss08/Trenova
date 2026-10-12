import { DeskInlineArtifact } from "@/components/desk-chat/conversation/desk-turns";
import { ArtifactLinkContext, AiMarkdown } from "@/components/elements/ai-markdown";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const card = {
  id: "art_01M4P458BMVHJ2KZ19Z2BTXCHV",
  kind: "entity_card",
  title: "Shipment S26100090038417",
} as const;

// The words of an artifact link are the model's; the badge is the artifact's.
// One reply called this card "shipment card" and the next "full record".
const reply =
  "Opened the full record; its [shipment card](artifact:art_01M4P458BMVHJ2KZ19Z2BTXCHV) has the details.";

function renderReply(onOpen = vi.fn()) {
  return render(
    <ArtifactLinkContext
      value={(id) =>
        id === card.id ? (
          <DeskInlineArtifact artifact={card} active={false} onOpen={onOpen} />
        ) : null
      }
    >
      <AiMarkdown content={reply} />
    </ArtifactLinkContext>,
  );
}

describe("DeskInlineArtifact", () => {
  it("reads as the artifact's title whatever words the reply wrote", () => {
    renderReply();

    expect(screen.getByRole("button")).toHaveTextContent("Shipment S26100090038417");
    expect(screen.queryByText("shipment card")).toBeNull();
  });

  it("stays inside the sentence that names it", () => {
    renderReply();

    expect(screen.getByRole("button").closest("p")).toHaveTextContent(
      "Opened the full record; its Shipment S26100090038417 has the details.",
    );
  });

  it("opens the artifact it names", () => {
    const onOpen = vi.fn();
    renderReply(onOpen);

    fireEvent.click(screen.getByRole("button"));
    expect(onOpen).toHaveBeenCalledWith(card.id);
  });
});
