import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AssistantThread } from "@/types/assistant";
import { AssistantHeader } from "../assistant-header";

afterEach(cleanup);

const thread: AssistantThread = {
  id: "athr_01M3034Q2N7JD99RA1D8DGH1ZF",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: "usr_1",
  agentDefinitionId: "agd_1",
  title: "Run the revenue report",
  status: "Active",
  lastMessageAt: 1,
  preferredProviderId: "",
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

function renderHeader(activeThread: AssistantThread | null) {
  const onDownloadTranscript = vi.fn();
  render(
    <AssistantHeader
      agents={[]}
      activeAgent={null}
      activeThread={activeThread}
      threads={activeThread ? [activeThread] : []}
      expanded={false}
      isStarting={false}
      onStart={() => {}}
      onSelectThread={() => {}}
      onDeleteThread={() => {}}
      onDownloadTranscript={onDownloadTranscript}
      onToggleExpanded={() => {}}
      onClose={() => {}}
    />,
  );

  return { onDownloadTranscript };
}

/**
 * A conversation worth keeping is worth taking out of the panel: the
 * transcript button hands the open thread to the download, and is not offered
 * on the launch pad, where there is nothing to save.
 */
describe("AssistantHeader transcript", () => {
  it("downloads the open conversation", () => {
    const { onDownloadTranscript } = renderHeader(thread);

    fireEvent.click(screen.getByRole("button", { name: "Download transcript" }));

    expect(onDownloadTranscript).toHaveBeenCalledWith(thread);
  });

  it("offers nothing to download without a conversation", () => {
    renderHeader(null);

    expect(screen.queryByRole("button", { name: "Download transcript" })).not.toBeInTheDocument();
  });
});
