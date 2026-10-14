import { deskMemorySettingsQueryOptions } from "@/lib/queries/desk-memories";
import { agentProfileQueryOptions } from "@/lib/queries/agent-profile";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DeskMemoryPage } from "./_components/memory/desk-memory-page";

export const prefetch: RoutePrefetch = () => [
  agentProfileQueryOptions("mine"),
  deskMemorySettingsQueryOptions(),
];

export function DeskMemoryRoutePage() {
  return <DeskMemoryPage />;
}
