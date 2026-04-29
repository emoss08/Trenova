import { mergeQueryKeys } from "@lukemorales/query-key-factory";
import { accountingControl } from "./accounting-control";
import { accountingReport } from "./accounting-report";
import { ar } from "./ar";
import { bankReceipt } from "./bank-receipt";
import { bankReceiptBatch } from "./bank-receipt-batch";
import { bankReceiptWorkItem } from "./bank-receipt-work-item";
import { journalEntry } from "./journal-entry";
import { journalReversal } from "./journal-reversal";
import { manualJournal } from "./manual-journal";
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
import { invoiceAdjustment } from "./invoice-adjustment";
import { invoiceAdjustmentControl } from "./invoice-adjustment-control";
import { location } from "./location";
import { organization } from "./organization";
import { pageFavoite } from "./page-favorite";
import { sequenceConfig } from "./sequence-config";
import { shipment } from "./shipment";
import { shipmentControl } from "./shipment-control";
import { notification } from "./notification";
import { tableChangeAlert } from "./table-change-alert";
import { tableConfiguration } from "./table-configuration";
import { user, userOrganization } from "./user";
import { weatherAlert } from "./weather-alert";
import { weatherRadar } from "./weather-radar";
import { worker } from "./worker";

export const queries = mergeQueryKeys(
  accountingControl,
  accountingReport,
  ar,
  bankReceipt,
  bankReceiptBatch,
  bankReceiptWorkItem,
  journalEntry,
  journalReversal,
  manualJournal,
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
  invoiceAdjustment,
  invoiceAdjustmentControl,
  location,
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
  weatherAlert,
  weatherRadar,
);
