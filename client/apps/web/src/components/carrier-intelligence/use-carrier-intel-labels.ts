import { useT } from "@trenova/shared/i18n/use-t";
import type {
  CarrierInsurancePolicyType,
  CarrierIntelAuthorityStatus,
  CarrierIntelDepth,
  CarrierIntelDesiredState,
  CarrierIntelEnrollmentMode,
  CarrierIntelEnrollmentReason,
  CarrierIntelEventResolution,
  CarrierIntelEventSource,
  CarrierIntelEventStatus,
  CarrierIntelFeedType,
  CarrierIntelInsuranceChangeKind,
  CarrierIntelInsuranceFilingType,
  CarrierIntelNetworkKind,
  CarrierIntelSection,
  CarrierIntelSeverity,
  CarrierIntelSyncField,
  CarrierIntelVendorState,
} from "@trenova/graphql/generated/graphql";
import { useMemo } from "react";

export type CarrierIntelLabels = {
  section: Record<CarrierIntelSection, string>;
  severity: Record<CarrierIntelSeverity, string>;
  eventStatus: Record<CarrierIntelEventStatus, string>;
  eventSource: Record<CarrierIntelEventSource, string>;
  resolution: Record<CarrierIntelEventResolution, string>;
  resolutionHint: Record<CarrierIntelEventResolution, string>;
  depth: Record<CarrierIntelDepth, string>;
  depthHint: Record<CarrierIntelDepth, string>;
  depthSource: Record<CarrierIntelDepth, string>;
  syncField: Record<CarrierIntelSyncField, string>;
  networkKind: Record<CarrierIntelNetworkKind, string>;
  filingType: Record<CarrierIntelInsuranceFilingType, string>;
  authorityStatus: Record<CarrierIntelAuthorityStatus, string>;
  insuranceChangeKind: Record<CarrierIntelInsuranceChangeKind, string>;
  policyType: Record<CarrierInsurancePolicyType, string>;
  vendorState: Record<CarrierIntelVendorState, string>;
  desiredState: Record<CarrierIntelDesiredState, string>;
  enrollmentMode: Record<CarrierIntelEnrollmentMode, string>;
  enrollmentReason: Record<CarrierIntelEnrollmentReason, string>;
  feedType: Record<CarrierIntelFeedType, string>;
};

export function useCarrierIntelLabels(): CarrierIntelLabels {
  const t = useT();

  return useMemo(
    () => ({
      section: {
        Identity: t("Identity"),
        Authority: t("Authority"),
        Insurance: t("Insurance"),
        Safety: t("Safety"),
        Basics: t("CSA BASICs"),
        Inspections: t("Inspections"),
        Crashes: t("Crashes"),
        Fleet: t("Fleet"),
        Equipment: t("Equipment"),
        Contacts: t("Contacts"),
        Operations: t("Operations"),
        ChangeHistory: t("Change history"),
        Network: t("Network signals"),
        Lanes: t("Lanes"),
        Benchmarks: t("Benchmarks"),
      },
      severity: {
        Critical: t("Critical"),
        High: t("High"),
        Medium: t("Medium"),
        Low: t("Low"),
        Info: t("Info"),
      },
      eventStatus: {
        Open: t("Open"),
        Acknowledged: t("Acknowledged"),
        Resolved: t("Resolved"),
        Dismissed: t("Dismissed"),
      },
      eventSource: {
        NativeChangeFeed: t("Provider change feed"),
        SnapshotDiff: t("Snapshot comparison"),
        RuleEvaluation: t("Rule evaluation"),
        EquipmentVerification: t("Equipment verification"),
        Enrollment: t("Monitoring enrollment"),
        ProviderError: t("Provider error"),
        Override: t("Override"),
      },
      resolution: {
        CarrierUpdated: t("Carrier updated"),
        CarrierBlocked: t("Carrier blocked"),
        OverrideGranted: t("Override granted"),
        NoActionRequired: t("No action required"),
        FalsePositive: t("False positive"),
      },
      resolutionHint: {
        CarrierUpdated: t("The carrier record was corrected to match the provider."),
        CarrierBlocked: t("The carrier was disqualified or set to do not use."),
        OverrideGranted: t("A time-boxed override lets the carrier through for now."),
        NoActionRequired: t("The change was reviewed and needs nothing further."),
        FalsePositive: t("The provider data is wrong. A note explaining why is required."),
      },
      depth: {
        Full: t("Full profile"),
        Lite: t("Lite profile"),
        FMCSA: t("FMCSA only"),
      },
      depthSource: {
        Full: t("a full profile"),
        Lite: t("a lite profile"),
        FMCSA: t("FMCSA data"),
      },
      depthHint: {
        Full: t("Every section the provider offers, including network signals and lanes."),
        Lite: t("Identity, authority, insurance and safety without the deeper signals."),
        FMCSA: t("Only what the FMCSA publishes: authority, safety and inspections."),
      },
      syncField: {
        safetyRating: t("Safety rating"),
        mcNumber: t("MC number"),
        name: t("Legal name"),
        dbaName: t("DBA name"),
        addressLine1: t("Address"),
        city: t("City"),
        postalCode: t("Postal code"),
        phone: t("Phone"),
        email: t("Email"),
      },
      networkKind: {
        Address: t("Shared address"),
        Phone: t("Shared phone"),
        Email: t("Shared email"),
        EIN: t("Shared EIN"),
        Equipment: t("Shared equipment"),
      },
      filingType: {
        BIPD: t("BIPD"),
        Cargo: t("Cargo"),
        Bond: t("Bond"),
        Other: t("Other"),
      },
      authorityStatus: {
        Active: t("Active"),
        Inactive: t("Inactive"),
        Revoked: t("Revoked"),
        None: t("None"),
        Unknown: t("Unknown"),
      },
      insuranceChangeKind: {
        ShortenExpiration: t("Expiration moved up"),
        CoverageChanged: t("Coverage changed"),
        NewFiling: t("New filing"),
      },
      policyType: {
        AutoLiability: t("Auto liability"),
        CargoLiability: t("Cargo liability"),
        GeneralLiability: t("General liability"),
        Umbrella: t("Umbrella"),
        WorkersComp: t("Workers' comp"),
      },
      vendorState: {
        Unknown: t("Unknown"),
        PendingAdd: t("Enrolling"),
        Active: t("Active"),
        PendingRemove: t("Unenrolling"),
        Removed: t("Removed"),
        Failed: t("Failed"),
      },
      desiredState: {
        Enrolled: t("Monitor"),
        NotEnrolled: t("Do not monitor"),
      },
      enrollmentMode: {
        Native: t("Provider watchlist"),
        SnapshotDiff: t("Scheduled refresh"),
      },
      enrollmentReason: {
        PolicyAllActive: t("All active carriers"),
        PolicyRecentUse: t("Recently used"),
        AssignedOrTendered: t("Assigned or tendered"),
        Manual: t("Enrolled manually"),
        SelfMonitor: t("Our own authority"),
        CustomerBroker: t("Customer broker"),
      },
      feedType: {
        ChangeFeed: t("Change feed"),
        SnapshotRefresh: t("Snapshot refresh"),
      },
    }),
    [t],
  );
}
