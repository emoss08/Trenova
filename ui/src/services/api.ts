import { http } from "@/lib/http-client";
import type { TractorSchema } from "@/lib/schemas/tractor-schema";
import type { TractorAssignment } from "@/types/assignment";
import { AccessorialChargeAPI } from "./accessorial-charge";
import { AnalyticsAPI } from "./analytics";
import { AuditEntryAPI } from "./audit-entry";
import { AuthAPI } from "./auth";
import { BillingAPI } from "./billing";
import { BillingControlAPI } from "./billing-control";
import { ConsolidationAPI } from "./consolidation";
import { ConsolidationSettingsAPI } from "./consolidation-setting";
import { CustomerAPI } from "./customer";
import { DatabaseBackupAPI } from "./database-backups";
import { DedicatedLaneAPI, DedicatedLaneSuggestionAPI } from "./dedicated-lane";
import { DocumentAPI } from "./document";
import { FavoriteAPI } from "./favorite";
import { GoogleMapsAPI } from "./google-maps";
import { IntegrationAPI } from "./integration";
import { NotificationAPI } from "./notification";
import { OrganizationAPI } from "./organization";
import { PatternConfigAPI } from "./pattern-config";
import { PermissionAPI } from "./permission";
import { RoleAPI } from "./role";
import { ShipmentAPI } from "./shipment";
import { ShipmentControlAPI } from "./shipment-control";
import { TableConfigurationAPI } from "./table-configuration";
import { UsStateAPI } from "./us-state";
import { UserAPI } from "./user";
import { AIAPI } from "./ai";

class AssignmentsAPI {
  // Get a tractor's assignments from the API
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
  }
}

export const api = new API();
