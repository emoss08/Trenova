import { http } from "@/lib/http-client";
import type { TractorSchema } from "@/lib/schemas/tractor-schema";
import type { TractorAssignment } from "@/types/assignment";
import { AccessorialChargeAPI } from "./accessorial-charge";
import { AIAPI } from "./ai";
import { AnalyticsAPI } from "./analytics";
import { AuditEntryAPI } from "./audit-entry";
import { AuthAPI } from "./auth";
import { BillingAPI } from "./billing";
import { BillingControlAPI } from "./billing-control";
import { ConsolidationAPI } from "./consolidation";
import { ConsolidationSettingsAPI } from "./consolidation-setting";
import { CustomerAPI } from "./customer";
import { DataRetentionAPI } from "./data-retention";
import { DatabaseBackupAPI } from "./database-backups";
import { DedicatedLaneAPI, DedicatedLaneSuggestionAPI } from "./dedicated-lane";
import { DispatchControlAPI } from "./dispatch-control";
import { DistanceOverrideAPI } from "./distance-override";
import { DockerAPI } from "./docker";
import { DocumentAPI } from "./document";
import { EmailProfileAPI } from "./email-profile";
import { FavoriteAPI } from "./favorite";
import { FiscalYearAPI } from "./fiscal-year";
import { GoogleMapsAPI } from "./google-maps";
import { HoldReasonAPI } from "./hold-reason";
import { IntegrationAPI } from "./integration";
import { LocationAPI } from "./location";
import { NotificationAPI } from "./notification";
import { OrganizationAPI } from "./organization";
import { PatternConfigAPI } from "./pattern-config";
import { PermissionAPI } from "./permission";
import { RoleAPI } from "./role";
import { SearchAPI } from "./search";
import { ShipmentAPI } from "./shipment";
import { ShipmentControlAPI } from "./shipment-control";
import { TableConfigurationAPI } from "./table-configuration";
import { UsStateAPI } from "./us-state";
import { UserAPI } from "./user";
import { VariableAPI } from "./variable";
import { WorkerAPI } from "./worker";

class AssignmentsAPI {
  async getTractorAssignments(tractorId?: TractorSchema["id"]) {
    const response = await http.get<TractorAssignment>(
      `/tractors/${tractorId}/assignment/`,
    );

    return response.data;
  }
}

class API {
  assignments: AssignmentsAPI;
  auth: AuthAPI;
  user: UserAPI;
  shipments: ShipmentAPI;
  usStates: UsStateAPI;
  customers: CustomerAPI;
  billing: BillingAPI;
  documents: DocumentAPI;
  favorites: FavoriteAPI;
  integrations: IntegrationAPI;
  googleMaps: GoogleMapsAPI;
  analytics: AnalyticsAPI;
  organization: OrganizationAPI;
  shipmentControl: ShipmentControlAPI;
  dispatchControl: DispatchControlAPI;
  billingControl: BillingControlAPI;
  databaseBackups: DatabaseBackupAPI;
  auditEntries: AuditEntryAPI;
  tableConfigurations: TableConfigurationAPI;
  permissions: PermissionAPI;
  roles: RoleAPI;
  dedicatedLane: DedicatedLaneAPI;
  dedicatedLaneSuggestions: DedicatedLaneSuggestionAPI;
  patternConfig: PatternConfigAPI;
  notifications: NotificationAPI;
  accessorialCharge: AccessorialChargeAPI;
  consolidations: ConsolidationAPI;
  consolidationSettings: ConsolidationSettingsAPI;
  ai: AIAPI;
  holdReasons: HoldReasonAPI;
  docker: DockerAPI;
  dataRetention: DataRetentionAPI;
  worker: WorkerAPI;
  variables: VariableAPI;
  emailProfile: EmailProfileAPI;
  locations: LocationAPI;
  fiscalYear: FiscalYearAPI;
  distanceOverride: DistanceOverrideAPI;
  search: SearchAPI;
  constructor() {
    this.assignments = new AssignmentsAPI();
    this.auth = new AuthAPI();
    this.shipments = new ShipmentAPI();
    this.usStates = new UsStateAPI();
    this.customers = new CustomerAPI();
    this.billing = new BillingAPI();
    this.documents = new DocumentAPI();
    this.favorites = new FavoriteAPI();
    this.integrations = new IntegrationAPI();
    this.googleMaps = new GoogleMapsAPI();
    this.analytics = new AnalyticsAPI();
    this.organization = new OrganizationAPI();
    this.shipmentControl = new ShipmentControlAPI();
    this.billingControl = new BillingControlAPI();
    this.dispatchControl = new DispatchControlAPI();
    this.databaseBackups = new DatabaseBackupAPI();
    this.auditEntries = new AuditEntryAPI();
    this.tableConfigurations = new TableConfigurationAPI();
    this.permissions = new PermissionAPI();
    this.roles = new RoleAPI();
    this.dedicatedLane = new DedicatedLaneAPI();
    this.dedicatedLaneSuggestions = new DedicatedLaneSuggestionAPI();
    this.patternConfig = new PatternConfigAPI();
    this.notifications = new NotificationAPI();
    this.user = new UserAPI();
    this.accessorialCharge = new AccessorialChargeAPI();
    this.consolidations = new ConsolidationAPI();
    this.consolidationSettings = new ConsolidationSettingsAPI();
    this.ai = new AIAPI();
    this.holdReasons = new HoldReasonAPI();
    this.docker = new DockerAPI();
    this.dataRetention = new DataRetentionAPI();
    this.worker = new WorkerAPI();
    this.variables = new VariableAPI();
    this.emailProfile = new EmailProfileAPI();
    this.locations = new LocationAPI();
    this.distanceOverride = new DistanceOverrideAPI();
    this.search = new SearchAPI();
    this.fiscalYear = new FiscalYearAPI();
  }
}

export const api = new API();
