import { mergeQueryKeys } from "@lukemorales/query-key-factory";
import { accountingControl } from "./accounting-control";
import { analytics } from "./analytics";
import { audit } from "./audit";
import { billingControl } from "./billing-control";
import { customer } from "./customer";
import { dispatchControl } from "./dispatch-control";
import { formulaTemplate } from "./formula-template";
import { googleMaps } from "./google-maps";
import { integration } from "./integration";
import { organization } from "./organization";
import { pageFavoite } from "./page-favorite";
import { sequenceConfig } from "./sequence-config";
import { shipment } from "./shipment";
import { shipmentControl } from "./shipment-control";
import { tableConfiguration } from "./table-configuration";
import { user, userOrganization } from "./user";
import { worker } from "./worker";

export const queries = mergeQueryKeys(
  accountingControl,
  billingControl,
  dispatchControl,
  userOrganization,
  pageFavoite,
  tableConfiguration,
  user,
  worker,
  organization,
  integration,
  customer,
  shipment,
  formulaTemplate,
  googleMaps,
  shipmentControl,
  sequenceConfig,
  analytics,
  audit,
);
