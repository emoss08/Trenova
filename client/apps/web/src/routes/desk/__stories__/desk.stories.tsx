import type { Meta, StoryObj } from "@storybook/react-vite";
import { DeskStory, THREAD_ID } from "./desk-harness";

const meta = {
  title: "Desk/Desk",
  component: DeskStory,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof DeskStory>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Home: Story = { args: { path: "/desk" } };
export const Conversation: Story = { args: { path: `/desk/t/${THREAD_ID}` } };
export const ConversationRailFolded: Story = {
  args: { path: `/desk/t/${THREAD_ID}`, rail: "closed" },
};
export const ConversationNoWorkspace: Story = {
  args: { path: `/desk/t/${THREAD_ID}`, pane: "closed" },
};
export const Decisions: Story = { args: { path: "/desk/decisions" } };
export const Watchtower: Story = { args: { path: "/desk/watchtower" } };
export const ConversationDecisionsTab: Story = {
  args: { path: `/desk/t/${THREAD_ID}`, tab: "decisions" },
};
export const ConversationActivityTab: Story = {
  args: { path: `/desk/t/${THREAD_ID}`, tab: "activity" },
};
