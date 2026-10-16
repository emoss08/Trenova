import type { WatchtowerSourceKind } from "@/lib/graphql/watchtower";
import {
  ActivityIcon,
  AlertTriangleIcon,
  BankNote01Icon,
  BotIcon,
  CheckDone01Icon,
  ClipboardIcon,
  ClockIcon,
  CloudRaining02Icon,
  Dataflow02Icon,
  File02Icon,
  FileX02Icon,
  Headphones01Icon,
  Lightbulb02Icon,
  Mail01Icon,
  MarkerPin02Icon,
  PackageIcon,
  PhoneCall01Icon,
  ReceiptIcon,
  Repeat04Icon,
  Shield01Icon,
  Speedometer03Icon,
  Truck01Icon,
  UserX01Icon,
  Users01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The mark for each kind of item. Keyed by every kind the server raises, so a
 * new kind fails the typecheck until it has a mark rather than borrowing one.
 */
const KIND_GLYPHS: Record<WatchtowerSourceKind, IconComponent> = {
  Insight: Lightbulb02Icon,
  AgentProposal: CheckDone01Icon,
  AgentPlan: Dataflow02Icon,
  AgentRunFailed: BotIcon,
  AgentException: AlertTriangleIcon,
  ServiceFailure: Headphones01Icon,
  CarrierIntelEvent: Shield01Icon,
  HOSViolation: ClockIcon,
  WeatherAlert: CloudRaining02Icon,
  EDIInboundQuarantined: File02Icon,
  BillingException: ReceiptIcon,
  DetentionOccurrence: Speedometer03Icon,
  InboundMessage: Mail01Icon,
  WorkerCredential: ClipboardIcon,
  MoveCoverage: UserX01Icon,
  TelematicsStopVisit: MarkerPin02Icon,
  AgentQualityRegression: ActivityIcon,
  AccountingSync: Repeat04Icon,
  NoCheckCall: PhoneCall01Icon,
  EtaAtRisk: Truck01Icon,
  MissedPickup: PackageIcon,
  MissedDelivery: Truck01Icon,
  PodMissing: FileX02Icon,
  TenderExhausted: Users01Icon,
  Factoring: BankNote01Icon,
};

export function WatchtowerKindGlyph({
  kind,
  className,
}: {
  kind: WatchtowerSourceKind;
  className?: string;
}) {
  const Glyph = KIND_GLYPHS[kind];

  return <Glyph aria-hidden className={cn("size-3", className)} strokeWidth={2} />;
}
