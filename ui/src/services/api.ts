import { http } from "@/lib/http-client";
import type { TractorAssignment } from "@/types/assignment";
import { Tractor } from "@/types/tractor";
import { AnalyticsAPI } from "./analytics";
import { AuditEntryAPI } from "./audit-entry";
import { AuthAPI } from "./auth";
import { BillingAPI } from "./billing";
import { BillingControlAPI } from "./billing-control";
import { CustomerAPI } from "./customer";
import { DatabaseBackupAPI } from "./database-backups";
import { DocumentAPI } from "./document";
import { GoogleMapsAPI } from "./google-maps";
import { IntegrationAPI } from "./integration";
import { OrganizationAPI } from "./organization";
import { ShipmentAPI } from "./shipment";
import { ShipmentControlAPI } from "./shipment-control";
import { UsStateAPI } from "./us-state";

class AssignmentsAPI {
  // Get a tractor's assignments from the API
  async getTractorAssignments(tractorId?: Tractor["id"]) {
    const response = await http.get<TractorAssignment>(
      `/tractors/${tractorId}/assignment/`,
    );

    return response.data;
  }
}

class API {
  assignments: AssignmentsAPI;
  auth: AuthAPI;
  shipments: ShipmentAPI;
  usStates: UsStateAPI;
  customers: CustomerAPI;
  billing: BillingAPI;
  documents: DocumentAPI;
  integrations: IntegrationAPI;
  googleMaps: GoogleMapsAPI;
  analytics: AnalyticsAPI;
  organization: OrganizationAPI;
  shipmentControl: ShipmentControlAPI;
  billingControl: BillingControlAPI;
  databaseBackups: DatabaseBackupAPI;
  auditEntries: AuditEntryAPI;

  constructor() {
    this.assignments = new AssignmentsAPI();
    this.auth = new AuthAPI();
    this.shipments = new ShipmentAPI();
    this.usStates = new UsStateAPI();
    this.customers = new CustomerAPI();
    this.billing = new BillingAPI();
    this.documents = new DocumentAPI();
    this.integrations = new IntegrationAPI();
    this.googleMaps = new GoogleMapsAPI();
    this.analytics = new AnalyticsAPI();
    this.organization = new OrganizationAPI();
    this.shipmentControl = new ShipmentControlAPI();
    this.billingControl = new BillingControlAPI();
    this.databaseBackups = new DatabaseBackupAPI();
    this.auditEntries = new AuditEntryAPI();
  }
}

export const api = new API();
