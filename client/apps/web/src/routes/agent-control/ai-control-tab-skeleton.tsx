import { ActivityTabSkeleton } from "./_components/activity/activity-skeleton";
import { AgentsTabSkeleton } from "./_components/agents/agents-skeleton";
import { AnalyticsTabSkeleton } from "./_components/analytics/analytics-skeleton";
import { AuditTabSkeleton } from "./_components/audit/audit-skeleton";
import { ExtensionsTabSkeleton } from "./_components/extensions/extensions-skeleton";
import { MemoryTabSkeleton } from "./_components/memory/memory-skeleton";
import { OverviewTabSkeleton } from "./_components/overview/overview-skeleton";
import { ProvidersTabSkeleton } from "./_components/providers/providers-skeleton";
import { QualityTabSkeleton } from "./_components/quality/quality-skeleton";
import { RetrievalTabSkeleton } from "./_components/retrieval/retrieval-skeleton";
import { SafetyTabSkeleton } from "./_components/safety/safety-skeleton";
import type {
  AIControlTab,
  ActivityView,
  AuditView,
  QualityView,
  RailView,
  SafetyView,
} from "./ai-control-tabs";

type AIControlTabSkeletonProps = {
  tab: AIControlTab;
  view: RailView | null;
};

/** The open tab while its code or its first answer is read, drawn in the tab's own layout. */
export function AIControlTabSkeleton({ tab, view }: AIControlTabSkeletonProps) {
  switch (tab) {
    case "overview":
      return <OverviewTabSkeleton />;
    case "agents":
      return <AgentsTabSkeleton />;
    case "providers":
      return <ProvidersTabSkeleton />;
    case "extensions":
      return <ExtensionsTabSkeleton />;
    case "memory":
      return <MemoryTabSkeleton />;
    case "retrieval":
      return <RetrievalTabSkeleton />;
    case "safety":
      return <SafetyTabSkeleton view={view as SafetyView} />;
    case "quality":
      return <QualityTabSkeleton view={view as QualityView} />;
    case "analytics":
      return <AnalyticsTabSkeleton />;
    case "activity":
      return <ActivityTabSkeleton view={view as ActivityView} />;
    case "audit":
      return <AuditTabSkeleton view={view as AuditView} />;
  }
}
