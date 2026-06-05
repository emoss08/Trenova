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
import { audit } from "./audit";
import { billingControl } from "./billing-control";
import { billingQueue } from "./billing-queue";
import { customer } from "./customer";
import { dataEntryControl } from "./data-entry-control";
import { distanceControl } from "./distance-control";
import { documentControl } from "./document-control";
import { documentParsingRule } from "./document-parsing-rule";
import { dispatchControl } from "./dispatch-control";
import { edi } from "./edi";
import { email } from "./email";
import { formulaTemplate } from "./formula-template";
import { googleMaps } from "./google-maps";
import { integration } from "./integration";
import { invoice } from "./invoice";
import { invoiceAdjustment } from "./invoice-adjustment";
import { invoiceAdjustmentControl } from "./invoice-adjustment-control";
import { location } from "./location";
import { organization } from "./organization";
import { pageFavoite } from "./page-favorite";
import { platformBilling } from "./platform-billing";
import { sequenceConfig } from "./sequence-config";
import { serviceFailure } from "./service-failure";
import { serviceFailureReasonCode } from "./service-failure-reason-code";
import { shipment } from "./shipment";
import { shipmentControl } from "./shipment-control";
import { notification } from "./notification";
import { tableChangeAlert } from "./table-change-alert";
import { tableConfiguration } from "./table-configuration";
import { user, userOrganization } from "./user";
import { weatherAlert } from "./weather-alert";
import { weatherRadar } from "./weather-radar";
import { worker } from "./worker";
import { exchangeRate } from "./exchange-rate";

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
  distanceControl,
  dispatchControl,
  edi,
  email,
  documentControl,
  documentParsingRule,
  userOrganization,
  pageFavoite,
  platformBilling,
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
  serviceFailure,
  serviceFailureReasonCode,
  shipmentControl,
  sequenceConfig,
  audit,
  notification,
  tableChangeAlert,
  weatherAlert,
  weatherRadar,
  exchangeRate,
);
