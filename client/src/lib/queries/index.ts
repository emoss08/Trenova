import { mergeQueryKeys } from "@lukemorales/query-key-factory";
import { accountingControl } from "./accounting-control";
import { analytics } from "./analytics";
import { audit } from "./audit";
import { billingControl } from "./billing-control";
import { billingQueue } from "./billing-queue";
import { customer } from "./customer";
import { dataEntryControl } from "./data-entry-control";
import { documentControl } from "./document-control";
import { documentParsingRule } from "./document-parsing-rule";
import { dispatchControl } from "./dispatch-control";
import { formulaTemplate } from "./formula-template";
import { googleMaps } from "./google-maps";
import { integration } from "./integration";
import { invoice } from "./invoice";
import { organization } from "./organization";
import { pageFavoite } from "./page-favorite";
import { sequenceConfig } from "./sequence-config";
import { shipment } from "./shipment";
import { shipmentControl } from "./shipment-control";
import { notification } from "./notification";
import { tableChangeAlert } from "./table-change-alert";
import { tableConfiguration } from "./table-configuration";
import { user, userOrganization } from "./user";
import { worker } from "./worker";

export const queries = mergeQueryKeys(
  accountingControl,
  billingControl,
  billingQueue,
  dataEntryControl,
  dispatchControl,
  documentControl,
  documentParsingRule,
  userOrganization,
  pageFavoite,
  tableConfiguration,
  user,
  worker,
  organization,
  integration,
  invoice,
  customer,
  shipment,
  formulaTemplate,
  googleMaps,
  shipmentControl,
  sequenceConfig,
  analytics,
  audit,
  notification,
  tableChangeAlert,
);
