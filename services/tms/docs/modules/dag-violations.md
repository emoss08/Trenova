# Gate G2: DAG-violation report

Every non-test import that breaks the dependency rules under the proposed module map
(`module-map.yml`), grouped by module pair, with up to five examples each. Imports of the two
`ports` hub packages are attributed to the module of the entity whose file declares the symbol
used (`ports/repositories/shipment.go` belongs to `operations`); a symbol declared in a file not
named after an entity is listed at the end and counted nowhere, so port edges are approximate
until Phase 3 splits the hubs.

| Violation | Imports |
|---|---|
| imports service of another module (rule 6) | 268 |
| upward (rule 3) | 221 |
| upward via port (rule 3) | 208 |
| pkg imports a module (rule 7) | 44 |
| same level (rule 2) | 18 |
| imports job of another module (rule 6) | 4 |
| same level via port (rule 2) | 4 |
| imports repository of another module (rule 6) | 3 |

## agent -> workspace: imports service of another module (rule 6), 36
- `core/services/agentaccessservice` -> `core/services/auditservice`
- `core/services/agentcontrolservice` -> `core/services/auditservice`
- `core/services/agentdecisionservice` -> `core/services/auditservice`
- `core/services/agentdefinitionservice` -> `core/services/auditservice`
- `core/services/agentevalcaseservice` -> `core/services/auditservice`

## identity -> workspace: upward via port (rule 3), 33
- `core/services/accountingcontrolservice` -> `ports/services.AuditService (audit)`
- `core/services/accountingcontrolservice` -> `ports/services.LogActionParams (audit)`
- `core/services/accountingcontrolservice` -> `ports/repositories.IntegrationRepository (integration)`
- `core/services/billingcontrolservice` -> `ports/services.AuditService (audit)`
- `core/services/billingcontrolservice` -> `ports/services.LogActionParams (audit)`

## documents -> agent: upward (rule 3), 30
- `core/domain/extractioneval` -> `core/domain/aicorrection`
- `core/domain/extractionshadow` -> `core/domain/aicorrection`
- `core/services/documentsearchprojectionservice` -> `core/domain/airetrieval`
- `core/services/documentservice` -> `core/domain/aicorrection`
- `core/services/extractionevalservice` -> `core/domain/aicorrection`

## documents -> agent: upward via port (rule 3), 24
- `core/services/documentservice` -> `ports/services.AICorrectionService (aicorrection)`
- `core/services/documentservice` -> `ports/services.CaptureShipmentDraftCorrectionRequest (aicorrection)`
- `core/services/extractionevalservice` -> `ports/repositories.ListAICorrectionsForAccuracyRequest (aicorrection)`
- `core/services/extractionevalservice` -> `ports/services.PurgeExpiredAICorrectionsRequest (aicorrection)`
- `core/services/extractionevalservice` -> `ports/repositories.GetAIProviderByIDRequest (aiprovider)`

## agent -> identity: imports service of another module (rule 6), 23
- `core/services/agentdefinitionservice` -> `core/services/planservice`
- `core/services/agentevents` -> `core/services/planservice`
- `core/services/agentextensionservice` -> `core/services/planservice`
- `core/services/agentquerytoolservice` -> `core/services/fieldsensitivity`
- `core/services/agentrunservice` -> `core/services/planservice`

## rating -> operations: upward via port (rule 3), 22
- `core/services/carrierassignmentservice` -> `ports/repositories.ShipmentRepository (shipment)`
- `core/services/carrierassignmentservice` -> `ports/repositories.ShipmentOptions (shipment)`
- `core/services/carrierassignmentservice` -> `ports/repositories.GetShipmentByIDRequest (shipment)`
- `core/services/carrierassignmentservice` -> `ports/services.RecordShipmentEventParams (shipmentevent)`
- `core/services/carrierassignmentservice` -> `ports/services.TenderGuard (tender)`

## identity -> workspace: upward (rule 3), 20
- `core/services/accountingcontrolservice` -> `core/domain/integration`
- `core/services/accountingcontrolservice` -> `core/services/auditservice`
- `core/services/billingcontrolservice` -> `core/services/auditservice`
- `core/services/dashcontrolservice` -> `core/services/auditservice`
- `core/services/documentcontrolservice` -> `core/services/auditservice`

## rating -> operations: upward (rule 3), 15
- `core/domain/rateconfirmation` -> `core/domain/shipment`
- `core/services/carrierassignmentservice` -> `core/domain/shipment`
- `core/services/carrierassignmentservice` -> `core/domain/shipmentstate`
- `core/services/carrierassignmentservice` -> `core/services/shipmenteventservice`
- `core/services/carrierassignmentservice` -> `core/services/shipmentservice`

## billing -> workspace: imports service of another module (rule 6), 15
- `core/services/accountingdriftservice` -> `core/services/auditservice`
- `core/services/bankreceiptbatchservice` -> `core/services/auditservice`
- `core/services/bankreceiptservice` -> `core/services/auditservice`
- `core/services/bankreceiptservice` -> `core/services/notificationservice`
- `core/services/bankreceiptworkitemservice` -> `core/services/auditservice`

## documents -> workspace: imports service of another module (rule 6), 15
- `core/services/captureservice` -> `core/services/auditservice`
- `core/services/captureservice` -> `core/services/notificationservice`
- `core/services/documentpacketruleservice` -> `core/services/auditservice`
- `core/services/documentparsingruleservice` -> `core/services/auditservice`
- `core/services/documentservice` -> `core/services/auditservice`

## identity -> billing: upward via port (rule 3), 14
- `core/temporaljobs/billingjobs` -> `ports/repositories.GetInvoiceByIDRequest (invoice)`
- `core/temporaljobs/billingjobs` -> `ports/repositories.InvoiceRepository (invoice)`
- `core/temporaljobs/billingjobs` -> `ports/repositories.LateChargeRepository (latecharge)`
- `core/temporaljobs/billingjobs` -> `ports/services.InvoicePreviewResult (invoice)`
- `core/temporaljobs/billingjobs` -> `ports/services.LateChargeService (latecharge)`

## operations -> workspace: imports service of another module (rule 6), 13
- `core/services/detentionservice` -> `core/services/auditservice`
- `core/services/dispatchcontrolservice` -> `core/services/auditservice`
- `core/services/orderservice` -> `core/services/auditservice`
- `core/services/recurringshipmentservice` -> `core/services/auditservice`
- `core/services/servicefailureservice` -> `core/services/auditservice`

## accounting -> billing: upward via port (rule 3), 12
- `core/services/accountingsyncservice` -> `ports/repositories.GetInvoiceAdjustmentRequest (invoiceadjustment)`
- `core/services/accountingsyncservice` -> `ports/repositories.GetInvoiceByIDRequest (invoice)`
- `core/services/accountingsyncservice` -> `ports/repositories.GetCreditMemoApplicationRequest (customerpayment)`
- `core/services/accountingsyncservice` -> `ports/repositories.GetInvoicesByIDsRequest (invoice)`
- `core/services/accountingsyncservice` -> `ports/repositories.GetCustomerPaymentByIDRequest (customerpayment)`

## operations -> agent: upward (rule 3), 12
- `core/services/assignmentservice` -> `core/domain/agent`
- `core/services/detentionservice` -> `core/domain/agent`
- `core/services/detentionservice` -> `core/domain/watchtower`
- `core/services/detentionservice` -> `core/services/watchtowersources`
- `core/services/servicefailureservice` -> `core/domain/agent`

## places -> operations: upward via port (rule 3), 12
- `core/services/distancecalculationservice` -> `ports/repositories.ShipmentRepository (shipment)`
- `core/services/distancecalculationservice` -> `ports/repositories.ShipmentOptions (shipment)`
- `core/services/distancecalculationservice` -> `ports/repositories.GetShipmentByIDRequest (shipment)`
- `core/services/distancecalculationservice` -> `ports/services.RecalculateMoveJurisdictionMilesRequest (shipment)`
- `core/services/distancecalculationservice` -> `ports/services.MoveJurisdictionMilesPlan (shipment)`

## accounting -> billing: upward (rule 3), 10
- `core/domain/accountingsync` -> `core/domain/billingqueue`
- `core/domain/accountingsync` -> `core/domain/customerpayment`
- `core/domain/accountingsync` -> `core/domain/invoice`
- `core/services/accountingsyncservice` -> `core/domain/customerpayment`
- `core/services/accountingsyncservice` -> `core/domain/invoice`

## agent -> rating: imports service of another module (rule 6), 10
- `core/services/agentquerytoolservice` -> `core/services/fuelsurchargeservice`
- `core/services/agentquerytoolservice` -> `core/services/rateagreementservice`
- `core/services/agentquerytoolservice` -> `core/services/ratematrixservice`
- `core/services/agentquerytoolservice` -> `core/services/ratequoteservice`
- `core/services/agenttoolservice` -> `core/services/carrierassignmentservice`

## rating -> workspace: imports service of another module (rule 6), 10
- `core/services/carrierassignmentservice` -> `core/services/auditservice`
- `core/services/formulatemplateservice` -> `core/services/auditservice`
- `core/services/formulatemplateservice` -> `core/services/notificationservice`
- `core/services/fuelsurchargeservice` -> `core/services/auditservice`
- `core/services/fuelsurchargeservice` -> `core/services/integrationservice`

## pkg/* -> rating: pkg imports a module (rule 7), 10
- `pkg/domainregistry` -> `core/domain/formulatemplate`
- `pkg/domainregistry` -> `core/domain/rateagreement`
- `pkg/domainregistry` -> `core/domain/rateimport`
- `pkg/domainregistry` -> `core/domain/ratematrix`
- `pkg/domainregistry` -> `core/domain/ratequote`

## reference -> workspace: imports service of another module (rule 6), 9
- `core/services/accessorialchargeservice` -> `core/services/auditservice`
- `core/services/commodityservice` -> `core/services/auditservice`
- `core/services/documenttypeservice` -> `core/services/auditservice`
- `core/services/exchangerateservice` -> `core/services/integrationservice`
- `core/services/hazardousmaterialservice` -> `core/services/auditservice`

## billing -> agent: upward (rule 3), 9
- `core/services/accountingdriftservice` -> `core/domain/agent`
- `core/services/accountingdriftservice` -> `core/domain/watchtower`
- `core/services/accountingdriftservice` -> `core/services/watchtowersources`
- `core/services/bankreceiptservice` -> `core/domain/agent`
- `core/services/billingqueueservice` -> `core/domain/agent`

## agent -> operations: imports service of another module (rule 6), 9
- `core/services/agentquerytoolservice` -> `core/services/detentionservice`
- `core/services/agentquerytoolservice` -> `core/services/dispatchconsoleservice`
- `core/services/agentquerytoolservice` -> `core/services/orderservice`
- `core/services/agenttoolservice` -> `core/services/detentionservice`
- `core/services/agenttoolservice` -> `core/services/dispatchconsoleservice`

## identity -> documents: upward via port (rule 3), 9
- `core/services/passwordresetservice` -> `ports/services.DocumentTemplateResolver (documenttemplate)`
- `core/services/passwordresetservice` -> `ports/services.RenderMessageRequest (documenttemplate)`
- `core/temporaljobs/billingjobs` -> `ports/repositories.GetDocumentByIDRequest (document)`
- `core/temporaljobs/billingjobs` -> `ports/services.DocumentUploadService (documentupload)`
- `core/temporaljobs/billingjobs` -> `ports/services.CreateSessionRequest (documentupload)`

## accounting -> workspace: imports service of another module (rule 6), 8
- `core/services/accountingsyncservice` -> `core/services/auditservice`
- `core/services/accounttypeservice` -> `core/services/auditservice`
- `core/services/fiscalperiodservice` -> `core/services/auditservice`
- `core/services/fiscalyearservice` -> `core/services/auditservice`
- `core/services/glaccountservice` -> `core/services/auditservice`

## agent -> billing: imports service of another module (rule 6), 8
- `core/services/agentquerytoolservice` -> `core/services/bankreceiptservice`
- `core/services/agentquerytoolservice` -> `core/services/invoicerunservice`
- `core/services/agenttoolservice` -> `core/services/bankreceiptservice`
- `core/services/agenttoolservice` -> `core/services/bankreceiptworkitemservice`
- `core/services/agenttoolservice` -> `core/services/billingqueueservice`

## agent -> settlement: imports service of another module (rule 6), 8
- `core/services/agentquerytoolservice` -> `core/services/carriersettlementservice`
- `core/services/agentquerytoolservice` -> `core/services/driverpayservice`
- `core/services/agentquerytoolservice` -> `core/services/driversettlementservice`
- `core/services/agenttoolservice` -> `core/services/carriersettlementservice`
- `core/services/agenttoolservice` -> `core/services/driverpayservice`

## places -> workspace: imports service of another module (rule 6), 8
- `core/services/distancecalculationservice` -> `core/services/integrationservice`
- `core/services/distancecontrolservice` -> `core/services/auditservice`
- `core/services/distanceoverrideservice` -> `core/services/auditservice`
- `core/services/distanceprofileservice` -> `core/services/auditservice`
- `core/services/jurisdictionruleservice` -> `core/services/auditservice`

## places -> workforce: upward via port (rule 3), 8
- `core/services/jurisdictionruleservice` -> `ports/repositories.DeleteOverrideRequest (permit)`
- `core/services/jurisdictionruleservice` -> `ports/repositories.ListOverridesRequest (permit)`
- `core/services/jurisdictionruleservice` -> `ports/repositories.GetOverrideByIDRequest (permit)`
- `core/services/jurisdictionruleservice` -> `ports/repositories.GetJurisdictionRulesRequest (permit)`
- `core/services/jurisdictionruleservice` -> `ports/repositories.GetJurisdictionRuleByIDRequest (permit)`

## identity -> equipment: upward (rule 3), 8
- `core/services/onboardingservice` -> `core/domain/equipmentmanufacturer`
- `core/services/onboardingservice` -> `core/domain/equipmenttype`
- `core/services/onboardingservice` -> `core/domain/tractor`
- `core/services/onboardingservice` -> `core/domain/trailer`
- `core/services/onboardingservice` -> `core/services/equipmentmanufacturerservice`

## pkg/* -> identity: pkg imports a module (rule 7), 8
- `pkg/approvalworkflow` -> `core/domain/permission`
- `pkg/domainregistry` -> `core/domain/permission`
- `pkg/domainregistry` -> `core/domain/tenant`
- `pkg/filtercatalog` -> `core/domain/permission`
- `pkg/realtimeinvalidation` -> `core/domain/permission`

## workspace -> parties: upward via port (rule 3), 7
- `core/services/bulkeditservice` -> `ports/repositories.BulkUpdateCustomerStatusRequest (customer)`
- `core/services/bulkeditservice` -> `ports/repositories.GetCarriersByIDsRequest (carrier)`
- `core/services/bulkeditservice` -> `ports/repositories.BulkUpdateCarrierStatusRequest (carrier)`
- `core/services/bulkeditservice` -> `ports/repositories.CustomerRepository (customer)`
- `core/services/bulkeditservice` -> `ports/repositories.GetCustomersByIDsRequest (customer)`

## documents -> operations: upward (rule 3), 7
- `core/services/documentintelligenceservice` -> `core/domain/documentshipmentdraft`
- `core/services/documentoperationsservice` -> `core/domain/documentshipmentdraft`
- `core/services/documentservice` -> `core/domain/documentshipmentdraft`
- `core/temporaljobs/documentintelligencejobs` -> `core/domain/documentshipmentdraft`
- `core/temporaljobs/documentuploadjobs` -> `core/domain/documentshipmentdraft`

## equipment -> workspace: imports service of another module (rule 6), 7
- `core/services/equipmentmanufacturerservice` -> `core/services/auditservice`
- `core/services/equipmenttypeservice` -> `core/services/auditservice`
- `core/services/fleetcodeservice` -> `core/services/auditservice`
- `core/services/tractorservice` -> `core/services/auditservice`
- `core/services/tractorservice` -> `core/services/customfieldservice`

## workforce -> workspace: imports service of another module (rule 6), 7
- `core/services/permitservice` -> `core/services/auditservice`
- `core/services/samsarasyncservice` -> `core/services/integrationservice`
- `core/services/telematicsservice` -> `core/services/customfieldservice`
- `core/services/telematicsservice` -> `core/services/integrationservice`
- `core/services/telematicsservice` -> `core/services/notificationservice`

## workforce -> operations: upward via port (rule 3), 7
- `core/services/permitservice` -> `ports/repositories.ShipmentRepository (shipment)`
- `core/services/permitservice` -> `ports/repositories.ShipmentOptions (shipment)`
- `core/services/permitservice` -> `ports/repositories.GetShipmentByIDRequest (shipment)`
- `core/services/telematicsservice` -> `ports/repositories.GetShipmentByIDRequest (shipment)`
- `core/services/telematicsservice` -> `ports/repositories.ShipmentRepository (shipment)`

## accounting -> agent: upward (rule 3), 6
- `core/services/accountingsyncservice` -> `core/domain/agent`
- `core/services/accountingsyncservice` -> `core/domain/watchtower`
- `core/services/accountingsyncservice` -> `core/services/watchtowersources`
- `core/temporaljobs/accountingsyncjobs` -> `core/domain/agent`
- `core/temporaljobs/accountingsyncjobs` -> `core/temporaljobs/agentflow/agentflowcontract`

## agent -> documents: imports service of another module (rule 6), 6
- `core/services/agentquerytoolservice` -> `core/services/captureservice`
- `core/services/agentquerytoolservice` -> `core/services/inboundmessageservice`
- `core/services/agentredteam` -> `core/services/inboundmessageservice`
- `core/services/agenttoolservice` -> `core/services/captureservice`
- `core/services/agenttoolservice` -> `core/services/documentservice`

## parties -> workspace: imports service of another module (rule 6), 6
- `core/services/carriercapacityservice` -> `core/services/auditservice`
- `core/services/carrierintelservice` -> `core/services/auditservice`
- `core/services/carrierintelservice` -> `core/services/integrationservice`
- `core/services/carrierintelservice` -> `core/services/notificationservice`
- `core/services/carrierservice` -> `core/services/auditservice`

## settlement -> workspace: imports service of another module (rule 6), 6
- `core/services/carriersettlementservice` -> `core/services/auditservice`
- `core/services/driverpayservice` -> `core/services/auditservice`
- `core/services/driversettlementservice` -> `core/services/auditservice`
- `core/services/driversettlementservice` -> `core/services/notificationservice`
- `core/services/fuelpurchaseservice` -> `core/services/auditservice`

## documents -> operations: upward via port (rule 3), 6
- `core/services/documentintelligenceservice` -> `ports/repositories.DocumentShipmentDraftRepository (documentshipmentdraft)`
- `core/services/documentservice` -> `ports/repositories.ShipmentRepository (shipment)`
- `core/services/documentservice` -> `ports/repositories.DocumentShipmentDraftRepository (documentshipmentdraft)`
- `core/services/documentservice` -> `ports/repositories.ListShipmentSummariesRequest (shipment)`
- `core/temporaljobs/documentintelligencejobs` -> `ports/repositories.DocumentShipmentDraftRepository (documentshipmentdraft)`

## billing -> settlement: same level (rule 2), 6
- `infrastructure/postgres/repositories/journalrepairrepository` -> `core/domain/driversettlement`
- `infrastructure/postgres/repositories/recordversionrepository` -> `core/domain/carriersettlement`
- `infrastructure/postgres/repositories/recordversionrepository` -> `core/domain/driverpay`
- `infrastructure/postgres/repositories/recordversionrepository` -> `core/domain/driversettlement`
- `infrastructure/postgres/repositories/recordversionrepository` -> `core/domain/fuelpurchase`

## workforce -> operations: upward (rule 3), 5
- `core/domain/telematics` -> `core/domain/shipment`
- `core/services/permitservice` -> `core/domain/shipment`
- `core/services/telematicsservice` -> `core/domain/shipment`
- `core/services/workerservice` -> `core/domain/dispatchcontrol`
- `infrastructure/postgres/repositories/telematicsrepository` -> `core/domain/shipment`

## agent -> accounting: imports service of another module (rule 6), 5
- `core/services/agentquerytoolservice` -> `core/services/fiscalperiodservice`
- `core/services/agenttoolservice` -> `core/services/accountingsyncservice`
- `core/services/agenttoolservice` -> `core/services/fiscalperiodservice`
- `core/services/agenttoolservice` -> `core/services/journalreversalservice`
- `core/services/agenttoolservice` -> `core/services/manualjournalservice`

## workspace -> parties: upward (rule 3), 5
- `core/services/bulkeditservice` -> `core/domain/carrier`
- `core/services/bulkeditservice` -> `core/domain/customer`
- `core/services/bulkeditservice` -> `core/services/carrierservice`
- `core/services/bulkeditservice` -> `core/services/customerservice`
- `core/services/integrationservice` -> `core/domain/carrierintel`

## billing -> identity: imports service of another module (rule 6), 5
- `core/services/customerpaymentservice` -> `core/services/exchangeratestamp`
- `core/services/invoiceadjustmentservice` -> `core/services/accountingcontrolpolicyservice`
- `core/services/invoiceadjustmentservice` -> `core/services/exchangeratestamp`
- `core/services/invoiceservice` -> `core/services/accountingcontrolpolicyservice`
- `core/services/invoiceservice` -> `core/services/exchangeratestamp`

## operations -> identity: imports service of another module (rule 6), 5
- `core/services/detentionservice` -> `core/services/orgzone`
- `core/services/dispatchconsoleservice` -> `core/services/orgzone`
- `core/services/shipmentservice` -> `core/services/quotaservice`
- `infrastructure/postgres/repositories/recurringshipmentrepository` -> `core/services/quotaservice`
- `infrastructure/postgres/repositories/shipmentrepository` -> `core/services/quotaservice`

## identity -> places: upward (rule 3), 5
- `core/services/onboardingservice` -> `core/domain/geofence`
- `core/services/onboardingservice` -> `core/domain/location`
- `core/services/onboardingservice` -> `core/domain/locationcategory`
- `core/services/onboardingservice` -> `core/services/locationcategoryservice`
- `core/services/onboardingservice` -> `core/services/locationservice`

## identity -> workforce: upward (rule 3), 5
- `core/services/onboardingservice` -> `core/domain/worker`
- `core/services/onboardingservice` -> `core/services/workerservice`
- `core/services/ptoledgerservice` -> `core/domain/worker`
- `core/services/workerptoservice` -> `core/domain/worker`
- `infrastructure/postgres/repositories/driverportalrepository` -> `core/domain/worker`

## identity -> documents: upward (rule 3), 5
- `core/services/passwordresetservice` -> `core/domain/documenttemplate`
- `core/services/workerptoservice` -> `core/domain/documenttemplate`
- `core/temporaljobs/billingjobs` -> `core/domain/documenttemplate`
- `core/temporaljobs/billingjobs` -> `core/temporaljobs/documentuploadjobs`
- `core/temporaljobs/billingjobs` -> `core/temporaljobs/documentuploadjobs/documentuploadcontract`

## equipment -> operations: upward (rule 3), 5
- `core/services/trailerservice` -> `core/domain/shipment`
- `core/services/trailerservice` -> `core/domain/shipmentstate`
- `core/services/trailerservice` -> `core/services/shipmentservice`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/servicefailure`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/shipment`

## pkg/* -> operations: pkg imports a module (rule 7), 5
- `pkg/domainregistry` -> `core/domain/dispatchcontrol`
- `pkg/domainregistry` -> `core/domain/recurringshipment`
- `pkg/domainregistry` -> `core/domain/servicefailure`
- `pkg/domainregistry` -> `core/domain/shipment`
- `pkg/lanequery` -> `core/domain/shipment`

## agent -> kernel: imports service of another module (rule 6), 4
- `core/services/agentcontrolservice` -> `core/services/editconflict`
- `core/services/agentdefinitionservice` -> `core/services/editconflict`
- `core/services/agentextensionservice` -> `core/services/secretconfig`
- `core/services/aiproviderservice` -> `core/services/editconflict`

## agent -> parties: imports service of another module (rule 6), 4
- `core/services/agenttoolservice` -> `core/services/carriercapacityservice`
- `core/services/agenttoolservice` -> `core/services/carrierintelservice`
- `core/services/agenttoolservice` -> `core/services/carrierservice`
- `core/services/agenttoolservice` -> `core/services/customerservice`

## workspace -> operations: upward via port (rule 3), 4
- `core/services/bulkeditservice` -> `ports/repositories.ShipmentRepository (shipment)`
- `core/services/bulkeditservice` -> `ports/repositories.TransferOwnershipRequest (shipment)`
- `core/services/bulkeditservice` -> `ports/repositories.GetShipmentsByIDsRequest (shipment)`
- `core/services/bulkeditservice` -> `ports/services.ShipmentService (shipment)`

## operations -> settlement: upward via port (rule 3), 4
- `core/services/detentionservice` -> `ports/repositories.WorkerPayAssignmentRepository (driverpay)`
- `core/services/detentionservice` -> `ports/repositories.PayProfileRepository (driverpay)`
- `core/services/detentionservice` -> `ports/repositories.GetWorkerPayAssignmentRequest (driverpay)`
- `core/services/detentionservice` -> `ports/repositories.GetPayProfileByIDRequest (driverpay)`

## identity -> reference: upward via port (rule 3), 4
- `core/services/exchangeratestamp` -> `ports/services.ExchangeRateService (exchangerate)`
- `core/services/exchangeratestamp/exchangeratestamptest` -> `ports/services.RateConversionResult (exchangerate)`
- `core/temporaljobs/billingjobs` -> `ports/repositories.DocumentTypeRepository (documenttype)`
- `core/temporaljobs/billingjobs` -> `ports/repositories.GetDocumentTypeByCodeRequest (documenttype)`

## operations -> edi: upward (rule 3), 4
- `core/services/internaledilifecycle` -> `core/domain/edi`
- `core/services/internaledistatussync` -> `core/domain/edi`
- `core/services/servicefailureservice` -> `core/domain/edi`
- `core/services/servicefailureservice` -> `core/services/edix12`

## identity -> reference: upward (rule 3), 4
- `core/services/onboardingservice` -> `core/domain/servicetype`
- `core/services/onboardingservice` -> `core/domain/shipmenttype`
- `core/services/onboardingservice` -> `core/services/servicetypeservice`
- `core/services/onboardingservice` -> `core/services/shipmenttypeservice`

## identity -> workforce: upward via port (rule 3), 4
- `core/services/ptoledgerservice` -> `ports/repositories.GetWorkerByIDRequest (worker)`
- `core/services/ptoledgerservice` -> `ports/repositories.WorkerRepository (worker)`
- `core/services/workerptoservice` -> `ports/repositories.GetWorkerByIDRequest (worker)`
- `core/services/workerptoservice` -> `ports/repositories.WorkerRepository (worker)`

## operations -> edi: upward via port (rule 3), 4
- `core/services/servicefailureservice` -> `ports/services.ServiceFailure214LifecycleRequest (edi)`
- `core/services/servicefailureservice` -> `ports/services.ServiceFailure214LifecycleResult (edi)`
- `core/services/servicefailureservice` -> `ports/services.ServiceFailureEDITrigger (edi)`
- `core/services/servicefailureservice` -> `ports/services.EDIService (edi)`

## workforce -> agent: upward (rule 3), 4
- `core/services/telematicsservice` -> `core/domain/agent`
- `core/services/telematicsservice` -> `core/domain/watchtower`
- `core/services/telematicsservice` -> `core/services/watchtowersources`
- `core/temporaljobs/telematicsjobs` -> `core/temporaljobs/modelcall`

## equipment -> places: same level via port (rule 2), 4
- `core/services/tractorservice` -> `ports/repositories.GetLocationByIDRequest (location)`
- `core/services/tractorservice` -> `ports/repositories.LocationRepository (location)`
- `core/services/trailerservice` -> `ports/repositories.GetLocationByIDRequest (location)`
- `core/services/trailerservice` -> `ports/repositories.LocationRepository (location)`

## pkg/* -> places: pkg imports a module (rule 7), 4
- `pkg/domainregistry` -> `core/domain/distancecontrol`
- `pkg/domainregistry` -> `core/domain/distanceoverride`
- `pkg/domainregistry` -> `core/domain/storedmileage`
- `pkg/rateimport` -> `core/domain/rategeo`

## workspace -> settlement: upward (rule 3), 3
- `core/domain/integration` -> `core/domain/fuelpurchase`
- `core/services/integrationservice` -> `core/domain/fuelpurchase`
- `core/temporaljobs/fuelcardjobs` -> `core/services/fuelpurchaseservice`

## workforce -> documents: same level (rule 2), 3
- `core/domain/worker` -> `core/domain/document`
- `core/domain/worker` -> `core/domain/documentpacketrule`
- `core/services/telematicsservice` -> `core/domain/documenttemplate`

## accounting -> identity: imports service of another module (rule 6), 3
- `core/services/accountingsyncservice` -> `core/services/accountingcontrolpolicyservice`
- `core/services/accountingsyncservice` -> `core/services/exchangeratestamp`
- `core/services/fiscalperiodservice` -> `core/services/accountingcontrolpolicyservice`

## workspace -> identity: imports service of another module (rule 6), 3
- `core/services/bulkeditservice` -> `core/services/tableinsightservice`
- `core/services/emailservice` -> `core/services/planservice`
- `core/services/integrationservice` -> `core/services/planservice`

## workspace -> places: upward via port (rule 3), 3
- `core/services/bulkeditservice` -> `ports/repositories.BulkUpdateLocationStatusRequest (location)`
- `core/services/bulkeditservice` -> `ports/repositories.LocationRepository (location)`
- `core/services/bulkeditservice` -> `ports/repositories.GetLocationsByIDsRequest (location)`

## parties -> agent: upward (rule 3), 3
- `core/services/carrierintelservice` -> `core/domain/agent`
- `core/services/carrierintelservice` -> `core/domain/watchtower`
- `core/services/carrierintelservice` -> `core/services/watchtowersources`

## settlement -> identity: imports service of another module (rule 6), 3
- `core/services/carriersettlementservice` -> `core/services/exchangeratestamp`
- `core/services/driversettlementservice` -> `core/services/exchangeratestamp`
- `core/services/iftaservice` -> `core/services/orgzone`

## reference -> accounting: upward (rule 3), 3
- `core/services/datatransformer` -> `core/domain/accounttype`
- `core/services/datatransformer` -> `core/domain/fiscalyear`
- `core/services/datatransformer` -> `core/domain/glaccount`

## operations -> agent: upward via port (rule 3), 3
- `core/services/detentionservice` -> `ports/services.WatchtowerProjector (watchtower)`
- `core/services/servicefailureservice` -> `ports/services.WatchtowerProjector (watchtower)`
- `core/services/shipmentsuggestionservice` -> `ports/repositories.ShipmentBriefRepository (shipmentbrief)`

## documents -> kernel: imports service of another module (rule 6), 3
- `core/services/documentservice` -> `core/services/usageservice`
- `core/services/documentuploadservice` -> `core/services/usageservice`
- `core/temporaljobs/documentuploadjobs` -> `core/services/usageservice`

## edi -> billing: upward via port (rule 3), 3
- `core/services/ediservice` -> `ports/repositories.UpdateInvoiceEDISendStatusRequest (invoice)`
- `core/services/ediservice` -> `ports/repositories.GetInvoiceByIDRequest (invoice)`
- `core/services/ediservice` -> `ports/repositories.InvoiceRepository (invoice)`

## operations -> billing: upward via port (rule 3), 3
- `core/services/shipmentservice` -> `ports/services.TransferToBillingResult (billingqueue)`
- `core/services/shipmentservice` -> `ports/services.TransferToBillingRequest (billingqueue)`
- `core/services/shipmentservice` -> `ports/services.BillingQueueService (billingqueue)`

## equipment -> places: same level (rule 2), 3
- `core/services/tractorservice` -> `core/domain/location`
- `core/services/trailerservice` -> `core/domain/location`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/location`

## equipment -> operations: upward via port (rule 3), 3
- `core/services/trailerservice` -> `ports/repositories.ShipmentOptions (shipment)`
- `core/services/trailerservice` -> `ports/repositories.GetShipmentByIDRequest (shipment)`
- `core/services/trailerservice` -> `ports/repositories.ShipmentRepository (shipment)`

## accounting -> settlement: upward (rule 3), 3
- `infrastructure/postgres/repositories/accountingsyncrepository` -> `core/domain/carriersettlement`
- `infrastructure/postgres/repositories/accountingsyncrepository` -> `core/domain/driverpay`
- `infrastructure/postgres/repositories/accountingsyncrepository` -> `core/domain/driversettlement`

## equipment -> billing: upward (rule 3), 3
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/bankreceipt`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/billingqueue`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/invoice`

## billing -> edi: imports repository of another module (rule 6), 3
- `infrastructure/postgres/repositories/recordversionrepository` -> `infrastructure/postgres/repositories/editenderchangerepository`
- `infrastructure/postgres/repositories/recordversionrepository` -> `infrastructure/postgres/repositories/editransferchangerepository`
- `infrastructure/postgres/repositories/recordversionrepository` -> `infrastructure/postgres/repositories/editransferrepository`

## pkg/* -> billing: pkg imports a module (rule 7), 3
- `pkg/domainregistry` -> `core/domain/bankreceipt`
- `pkg/domainregistry` -> `core/domain/invoice`
- `pkg/domainregistry` -> `core/domain/invoicerun`

## pkg/* -> equipment: pkg imports a module (rule 7), 3
- `pkg/domainregistry` -> `core/domain/equipmentcontinuity`
- `pkg/domainregistry` -> `core/domain/fleetcode`
- `pkg/domainregistry` -> `core/domain/trailer`

## parties -> accounting: upward (rule 3), 2
- `core/domain/customer` -> `core/domain/glaccount`
- `core/services/carrierservice` -> `core/domain/accountingsync`

## kernel -> identity: upward (rule 3), 2
- `core/domain/settingversion` -> `core/domain/tenant`
- `core/services/editconflict` -> `core/domain/tenant`

## equipment -> workforce: upward (rule 3), 2
- `core/domain/tractor` -> `core/domain/worker`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/worker`

## billing -> agent: upward via port (rule 3), 2
- `core/services/accountingdriftservice` -> `ports/services.WatchtowerProjector (watchtower)`
- `core/services/billingqueueservice` -> `ports/services.WatchtowerProjector (watchtower)`

## agent -> edi: imports service of another module (rule 6), 2
- `core/services/agentquerytoolservice` -> `core/services/ediservice`
- `core/services/agenttoolservice` -> `core/services/ediservice`

## agent -> workforce: imports service of another module (rule 6), 2
- `core/services/agentquerytoolservice` -> `core/services/telematicsservice`
- `core/services/agenttoolservice` -> `core/services/workerservice`

## agent -> reference: imports service of another module (rule 6), 2
- `core/services/agenttoolservice` -> `core/services/commodityservice`
- `core/services/agenttoolservice` -> `core/services/hazardousmaterialservice`

## agent -> places: imports service of another module (rule 6), 2
- `core/services/agenttoolservice` -> `core/services/locationservice`
- `core/services/shipmentimportassistantservice` -> `core/services/locationservice`

## agent -> equipment: imports service of another module (rule 6), 2
- `core/services/agenttoolservice` -> `core/services/tractorservice`
- `core/services/agenttoolservice` -> `core/services/trailerservice`

## workspace -> agent: upward via port (rule 3), 2
- `core/services/auditservice` -> `ports/repositories.GetAgentDefinitionByIDRequest (agentdefinition)`
- `core/services/auditservice` -> `ports/repositories.AgentDefinitionRepository (agentdefinition)`

## workspace -> places: upward (rule 3), 2
- `core/services/bulkeditservice` -> `core/domain/location`
- `core/services/bulkeditservice` -> `core/services/locationservice`

## parties -> identity: imports service of another module (rule 6), 2
- `core/services/carrierintelservice` -> `core/services/planservice`
- `infrastructure/postgres/repositories/customerrepository` -> `core/services/quotaservice`

## reference -> equipment: upward (rule 3), 2
- `core/services/datatransformer` -> `core/domain/equipmenttype`
- `core/services/datatransformer` -> `core/domain/fleetcode`

## places -> operations: upward (rule 3), 2
- `core/services/distancecalculationservice` -> `core/domain/shipment`
- `infrastructure/postgres/repositories/distancecalculationrepository` -> `core/domain/shipment`

## documents -> identity: imports service of another module (rule 6), 2
- `core/services/documentintelligenceservice` -> `core/services/planservice`
- `infrastructure/postgres/repositories/documentrepository` -> `core/services/quotaservice`

## edi -> workspace: imports service of another module (rule 6), 2
- `core/services/ediservice` -> `core/services/auditservice`
- `core/services/ediservice` -> `core/services/notificationservice`

## workspace -> rating: upward (rule 3), 2
- `core/services/formulaassistantservice` -> `core/services/formula`
- `core/services/formulaassistantservice` -> `core/services/formulatemplateservice`

## workspace -> rating: upward via port (rule 3), 2
- `core/services/formulaassistantservice` -> `ports/repositories.GetRateMatrixLookupDataRequest (ratematrix)`
- `core/services/formulaassistantservice` -> `ports/repositories.RateMatrixRepository (ratematrix)`

## billing -> rating: imports service of another module (rule 6), 2
- `core/services/invoiceadjustmentservice` -> `core/services/formula/contextvariablecache`
- `core/services/invoiceadjustmentservice` -> `core/services/formula/effectiveversioncache`

## identity -> parties: upward (rule 3), 2
- `core/services/onboardingservice` -> `core/domain/customer`
- `core/services/onboardingservice` -> `core/services/customerservice`

## identity -> rating: upward (rule 3), 2
- `core/services/onboardingservice` -> `core/domain/formulatemplate`
- `core/services/onboardingservice` -> `core/services/formulatemplateservice`

## identity -> operations: upward (rule 3), 2
- `core/services/onboardingservice` -> `core/domain/shipment`
- `infrastructure/postgres/repositories/driverportalrepository` -> `core/domain/shipment`

## identity -> rating: upward via port (rule 3), 2
- `core/services/onboardingservice` -> `ports/repositories.GetFormulaTemplatesByNamesRequest (formulatemplate)`
- `core/services/onboardingservice` -> `ports/repositories.FormulaTemplateRepository (formulatemplate)`

## operations -> billing: upward (rule 3), 2
- `core/services/shipmentquickfilterservice` -> `core/domain/costingcontrol`
- `core/services/shipmentservice` -> `core/domain/billingqueue`

## workforce -> agent: upward via port (rule 3), 2
- `core/services/telematicsservice` -> `ports/services.AgentWaitNotifier (agentwait)`
- `core/services/telematicsservice` -> `ports/services.WatchtowerProjector (watchtower)`

## workforce -> identity: imports service of another module (rule 6), 2
- `core/services/workeroverviewservice` -> `core/services/ptoledgerservice`
- `infrastructure/postgres/repositories/workerrepository` -> `core/services/quotaservice`

## identity -> edi: upward via port (rule 3), 2
- `core/temporaljobs/billingjobs` -> `ports/services.EDIService (edi)`
- `core/temporaljobs/billingjobs` -> `ports/services.GenerateEDIDocumentRequest (edi)`

## operations -> documents: imports job of another module (rule 6), 2
- `core/temporaljobs/detentionjobs` -> `core/temporaljobs/documentuploadjobs`
- `core/temporaljobs/detentionjobs` -> `core/temporaljobs/documentuploadjobs/documentuploadcontract`

## workspace -> documents: upward via port (rule 3), 2
- `core/temporaljobs/reportjobs` -> `ports/services.DocumentTemplateResolver (documenttemplate)`
- `core/temporaljobs/reportjobs` -> `ports/services.RenderMessageRequest (documenttemplate)`

## parties -> operations: upward (rule 3), 2
- `infrastructure/postgres/repositories/carrierintelrepository` -> `core/domain/shipment`
- `infrastructure/postgres/repositories/carrierintelrepository` -> `core/domain/tender`

## equipment -> parties: same level (rule 2), 2
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/carrier`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/customer`

## equipment -> settlement: upward (rule 3), 2
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/carriersettlement`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/driversettlement`

## equipment -> documents: upward (rule 3), 2
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/document`
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/inboundmessage`

## equipment -> identity: imports service of another module (rule 6), 2
- `infrastructure/postgres/repositories/tractorrepository` -> `core/services/quotaservice`
- `infrastructure/postgres/repositories/trailerrepository` -> `core/services/quotaservice`

## pkg/* -> reference: pkg imports a module (rule 7), 2
- `pkg/domainregistry` -> `core/domain/accessorialcharge`
- `pkg/domainregistry` -> `core/domain/servicetype`

## pkg/* -> workspace: pkg imports a module (rule 7), 2
- `pkg/domainregistry` -> `core/domain/audit`
- `pkg/domainregistry` -> `core/domain/report`

## pkg/* -> kernel: pkg imports a module (rule 7), 1
- `pkg/domainregistry` -> `core/domain/usstate`

## parties -> edi: upward (rule 3), 1
- `core/domain/carrier` -> `core/domain/edi`

## parties -> equipment: same level (rule 2), 1
- `core/domain/carriercapacity` -> `core/domain/equipmenttype`

## parties -> places: same level (rule 2), 1
- `core/domain/carriercapacity` -> `core/domain/location`

## parties -> workforce: upward (rule 3), 1
- `core/domain/carrierintel` -> `core/domain/worker`

## parties -> rating: upward (rule 3), 1
- `core/domain/customer` -> `core/domain/fuelsurcharge`

## places -> parties: same level (rule 2), 1
- `core/domain/distanceoverride` -> `core/domain/customer`

## accounting -> agent: upward via port (rule 3), 1
- `core/services/accountingsyncservice` -> `ports/services.WatchtowerProjector (watchtower)`

## workspace -> agent: upward (rule 3), 1
- `core/services/auditservice` -> `core/domain/agentdefinition`

## workspace -> operations: upward (rule 3), 1
- `core/services/bulkeditservice` -> `core/domain/shipment`

## parties -> agent: upward via port (rule 3), 1
- `core/services/carrierintelservice` -> `ports/services.WatchtowerProjector (watchtower)`

## reference -> parties: upward (rule 3), 1
- `core/services/datatransformer` -> `core/domain/customer`

## reference -> places: upward (rule 3), 1
- `core/services/datatransformer` -> `core/domain/location`

## operations -> settlement: upward (rule 3), 1
- `core/services/detentionservice` -> `core/domain/driverpay`

## places -> settlement: upward (rule 3), 1
- `core/services/distancecalculationservice` -> `core/temporaljobs/iftajobs/iftacontract`

## edi -> billing: upward (rule 3), 1
- `core/services/ediservice` -> `core/domain/invoice`

## edi -> operations: imports service of another module (rule 6), 1
- `core/services/ediservice` -> `core/services/internaledilifecycle`

## edi -> identity: imports service of another module (rule 6), 1
- `core/services/ediservice` -> `core/services/planservice`

## workspace -> billing: upward via port (rule 3), 1
- `core/services/emailservice` -> `ports/repositories.InvoiceRepository (invoice)`

## rating -> identity: imports service of another module (rule 6), 1
- `core/services/fuelsurchargeservice` -> `core/services/orgzone`

## workspace -> kernel: imports service of another module (rule 6), 1
- `core/services/integrationservice` -> `core/services/secretconfig`

## billing -> identity: imports job of another module (rule 6), 1
- `core/services/invoiceservice` -> `core/temporaljobs/billingjobs/billingcontract`

## identity -> accounting: upward (rule 3), 1
- `core/services/journalreviewservice` -> `core/domain/journalentry`

## identity -> accounting: upward via port (rule 3), 1
- `core/services/journalreviewservice` -> `ports/repositories.FiscalPeriodRepository (fiscalperiod)`

## identity -> operations: upward via port (rule 3), 1
- `core/services/onboardingservice` -> `ports/services.ShipmentService (shipment)`

## identity -> agent: upward (rule 3), 1
- `core/services/permission` -> `core/domain/agentdefinition`

## workspace -> accounting: upward (rule 3), 1
- `core/services/reporting/canned` -> `core/domain/accountingsync`

## operations -> reference: imports service of another module (rule 6), 1
- `core/services/shipmentservice` -> `core/services/modeprofileservice`

## operations -> workforce: imports service of another module (rule 6), 1
- `core/services/shipmentservice` -> `core/services/permitservice`

## workforce -> accounting: upward (rule 3), 1
- `core/services/workerservice` -> `core/domain/accountingsync`

## agent -> workspace: imports job of another module (rule 6), 1
- `core/temporaljobs/aiauditjobs` -> `core/temporaljobs/reportjobs`

## identity -> edi: upward (rule 3), 1
- `core/temporaljobs/billingjobs` -> `core/domain/edi`

## identity -> billing: upward (rule 3), 1
- `core/temporaljobs/billingjobs` -> `core/domain/invoice`

## workspace -> documents: upward (rule 3), 1
- `core/temporaljobs/reportjobs` -> `core/domain/documenttemplate`

## documents -> workforce: same level (rule 2), 1
- `infrastructure/postgres/repositories/capturerepository` -> `core/domain/worker`

## places -> identity: imports service of another module (rule 6), 1
- `infrastructure/postgres/repositories/locationrepository` -> `core/services/quotaservice`

## equipment -> agent: upward (rule 3), 1
- `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/insight`

## workforce -> settlement: upward (rule 3), 1
- `infrastructure/postgres/repositories/workerrepository` -> `core/domain/driverpay`

## pkg/* -> parties: pkg imports a module (rule 7), 1
- `pkg/domainregistry` -> `core/domain/customer`

## pkg/* -> documents: pkg imports a module (rule 7), 1
- `pkg/domainregistry` -> `core/domain/documenttemplate`

## pkg/* -> edi: pkg imports a module (rule 7), 1
- `pkg/domainregistry` -> `core/domain/edi`

## pkg/* -> agent: pkg imports a module (rule 7), 1
- `pkg/domainregistry` -> `core/domain/shipmentimportchat`

## pkg/* -> settlement: pkg imports a module (rule 7), 1
- `pkg/fuelimport` -> `core/domain/fuelpurchase`

## Same-level pairs (rule 2), for a decision

- billing <-> settlement (same level (rule 2)): 6, e.g. `infrastructure/postgres/repositories/journalrepairrepository` -> `core/domain/driversettlement`
- documents <-> workforce (same level (rule 2)): 1, e.g. `infrastructure/postgres/repositories/capturerepository` -> `core/domain/worker`
- equipment <-> parties (same level (rule 2)): 2, e.g. `infrastructure/postgres/repositories/recordlabelrepository` -> `core/domain/carrier`
- equipment <-> places (same level (rule 2)): 3, e.g. `core/services/tractorservice` -> `core/domain/location`
- equipment <-> places (same level via port (rule 2)): 4, e.g. `core/services/tractorservice` -> `ports/repositories.GetLocationByIDRequest (location)`
- parties <-> equipment (same level (rule 2)): 1, e.g. `core/domain/carriercapacity` -> `core/domain/equipmenttype`
- parties <-> places (same level (rule 2)): 1, e.g. `core/domain/carriercapacity` -> `core/domain/location`
- places <-> parties (same level (rule 2)): 1, e.g. `core/domain/distanceoverride` -> `core/domain/customer`
- workforce <-> documents (same level (rule 2)): 3, e.g. `core/domain/worker` -> `core/domain/document`

## Proposed inversion ports for upward edges (rule 3)

Each line: the lower module declares a port in its own `ports` covering what it uses today; the higher module's `module.go` provides the implementation with `fx.As`.

- **documents -> agent** (54 uses): `documents` declares a port for `aicorrection`, `aiprovider`, `modelcall`, `airetrieval`, `aicorrectionretention`, `agent`, `services.PurgeExpiredAICorrectionsRequest`, `repositories.GetAIProviderByIDRequest`; `agent` provides it.
- **identity -> workspace** (53 uses): `identity` declares a port for `auditservice`, `services.AuditService`, `services.LogActionParams`, `audit`, `integration`, `email`, `notification`, `repositories.IntegrationRepository`; `workspace` provides it.
- **rating -> operations** (37 uses): `rating` declares a port for `shipment`, `repositories.ShipmentRepository`, `repositories.ShipmentOptions`, `repositories.GetShipmentByIDRequest`, `shipmentstate`, `shipmenteventservice`, `shipmentservice`, `tender`; `operations` provides it.
- **accounting -> billing** (22 uses): `accounting` declares a port for `customerpayment`, `invoice`, `billingqueue`, `invoiceadjustment`, `repositories.InvoiceRepository`, `repositories.GetInvoiceAdjustmentRequest`, `repositories.GetInvoiceByIDRequest`, `repositories.GetCreditMemoApplicationRequest`; `billing` provides it.
- **operations -> agent** (15 uses): `operations` declares a port for `agent`, `watchtower`, `watchtowersources`, `services.WatchtowerProjector`, `shipmentbrief`, `shipmentboardbrief`, `repositories.ShipmentBriefRepository`; `agent` provides it.
- **identity -> billing** (15 uses): `identity` declares a port for `invoice`, `repositories.GetInvoiceByIDRequest`, `repositories.InvoiceRepository`, `repositories.LateChargeRepository`, `services.InvoicePreviewResult`, `services.LateChargeService`, `services.PostInvoiceRequest`, `services.InvoicePreviewRequest`; `billing` provides it.
- **places -> operations** (14 uses): `places` declares a port for `shipment`, `repositories.ShipmentRepository`, `repositories.ShipmentOptions`, `repositories.GetShipmentByIDRequest`, `services.RecalculateMoveJurisdictionMilesRequest`, `services.MoveJurisdictionMilesPlan`, `services.JurisdictionMileResult`, `services.DistanceMoveResult`; `operations` provides it.
- **identity -> documents** (14 uses): `identity` declares a port for `documenttemplate`, `services.DocumentTemplateResolver`, `documentuploadjobs`, `documentuploadcontract`, `services.RenderMessageRequest`, `repositories.GetDocumentByIDRequest`, `services.DocumentUploadService`, `services.CreateSessionRequest`; `documents` provides it.
- **documents -> operations** (13 uses): `documents` declares a port for `documentshipmentdraft`, `repositories.DocumentShipmentDraftRepository`, `shipment`, `repositories.ShipmentRepository`, `repositories.ListShipmentSummariesRequest`, `services.ShipmentService`; `operations` provides it.
- **workforce -> operations** (12 uses): `workforce` declares a port for `shipment`, `repositories.ShipmentRepository`, `repositories.GetShipmentByIDRequest`, `repositories.DispatchControlRepository`, `dispatchcontrol`, `repositories.ShipmentOptions`; `operations` provides it.
- **workspace -> parties** (12 uses): `workspace` declares a port for `carrier`, `customer`, `carrierservice`, `customerservice`, `carrierintel`, `repositories.BulkUpdateCustomerStatusRequest`, `repositories.GetCarriersByIDsRequest`, `repositories.BulkUpdateCarrierStatusRequest`; `parties` provides it.
- **billing -> agent** (11 uses): `billing` declares a port for `agent`, `watchtower`, `watchtowersources`, `services.WatchtowerProjector`, `insight`; `agent` provides it.
- **identity -> workforce** (9 uses): `identity` declares a port for `worker`, `repositories.GetWorkerByIDRequest`, `repositories.WorkerRepository`, `workerservice`; `workforce` provides it.
- **identity -> reference** (8 uses): `identity` declares a port for `services.ExchangeRateService`, `services.RateConversionResult`, `repositories.DocumentTypeRepository`, `repositories.GetDocumentTypeByCodeRequest`, `servicetype`, `shipmenttype`, `servicetypeservice`, `shipmenttypeservice`; `reference` provides it.
- **operations -> edi** (8 uses): `operations` declares a port for `edi`, `edix12`, `services.ServiceFailure214LifecycleRequest`, `services.ServiceFailure214LifecycleResult`, `services.ServiceFailureEDITrigger`, `services.EDIService`; `edi` provides it.
- **places -> workforce** (8 uses): `places` declares a port for `repositories.DeleteOverrideRequest`, `repositories.ListOverridesRequest`, `repositories.GetOverrideByIDRequest`, `repositories.GetJurisdictionRulesRequest`, `repositories.GetJurisdictionRuleByIDRequest`, `repositories.ListJurisdictionRulesRequest`, `repositories.VerifyJurisdictionRuleRequest`, `repositories.JurisdictionRuleRepository`; `workforce` provides it.
- **identity -> equipment** (8 uses): `identity` declares a port for `equipmentmanufacturer`, `equipmenttype`, `tractor`, `trailer`, `equipmentmanufacturerservice`, `equipmenttypeservice`, `tractorservice`, `trailerservice`; `equipment` provides it.
- **equipment -> operations** (8 uses): `equipment` declares a port for `shipment`, `shipmentstate`, `shipmentservice`, `servicefailure`, `repositories.ShipmentOptions`, `repositories.GetShipmentByIDRequest`, `repositories.ShipmentRepository`; `operations` provides it.
- **accounting -> agent** (7 uses): `accounting` declares a port for `agent`, `watchtower`, `watchtowersources`, `agentflowcontract`, `modelcall`, `services.WatchtowerProjector`; `agent` provides it.
- **workforce -> agent** (6 uses): `workforce` declares a port for `agent`, `watchtower`, `watchtowersources`, `modelcall`, `services.AgentWaitNotifier`, `services.WatchtowerProjector`; `agent` provides it.
- **workspace -> places** (5 uses): `workspace` declares a port for `location`, `locationservice`, `repositories.BulkUpdateLocationStatusRequest`, `repositories.LocationRepository`, `repositories.GetLocationsByIDsRequest`; `places` provides it.
- **workspace -> operations** (5 uses): `workspace` declares a port for `shipment`, `repositories.ShipmentRepository`, `repositories.TransferOwnershipRequest`, `repositories.GetShipmentsByIDsRequest`, `services.ShipmentService`; `operations` provides it.
- **operations -> settlement** (5 uses): `operations` declares a port for `driverpay`, `repositories.WorkerPayAssignmentRepository`, `repositories.PayProfileRepository`, `repositories.GetWorkerPayAssignmentRequest`, `repositories.GetPayProfileByIDRequest`; `settlement` provides it.
- **identity -> places** (5 uses): `identity` declares a port for `geofence`, `location`, `locationcategory`, `locationcategoryservice`, `locationservice`; `places` provides it.
- **operations -> billing** (5 uses): `operations` declares a port for `costingcontrol`, `billingqueue`, `services.TransferToBillingResult`, `services.TransferToBillingRequest`, `services.BillingQueueService`; `billing` provides it.
- **parties -> agent** (4 uses): `parties` declares a port for `agent`, `watchtower`, `watchtowersources`, `services.WatchtowerProjector`; `agent` provides it.
- **edi -> billing** (4 uses): `edi` declares a port for `invoice`, `repositories.UpdateInvoiceEDISendStatusRequest`, `repositories.GetInvoiceByIDRequest`, `repositories.InvoiceRepository`; `billing` provides it.
- **workspace -> rating** (4 uses): `workspace` declares a port for `formula`, `formulatemplateservice`, `repositories.GetRateMatrixLookupDataRequest`, `repositories.RateMatrixRepository`; `rating` provides it.
- **identity -> rating** (4 uses): `identity` declares a port for `formulatemplate`, `formulatemplateservice`, `repositories.GetFormulaTemplatesByNamesRequest`, `repositories.FormulaTemplateRepository`; `rating` provides it.
- **workspace -> settlement** (3 uses): `workspace` declares a port for `fuelpurchase`, `fuelpurchaseservice`; `settlement` provides it.
- **workspace -> agent** (3 uses): `workspace` declares a port for `agentdefinition`, `repositories.GetAgentDefinitionByIDRequest`, `repositories.AgentDefinitionRepository`; `agent` provides it.
- **reference -> accounting** (3 uses): `reference` declares a port for `accounttype`, `fiscalyear`, `glaccount`; `accounting` provides it.
- **identity -> operations** (3 uses): `identity` declares a port for `shipment`, `services.ShipmentService`; `operations` provides it.
- **identity -> edi** (3 uses): `identity` declares a port for `edi`, `services.EDIService`, `services.GenerateEDIDocumentRequest`; `edi` provides it.
- **workspace -> documents** (3 uses): `workspace` declares a port for `documenttemplate`, `services.DocumentTemplateResolver`, `services.RenderMessageRequest`; `documents` provides it.
- **accounting -> settlement** (3 uses): `accounting` declares a port for `carriersettlement`, `driverpay`, `driversettlement`; `settlement` provides it.
- **equipment -> billing** (3 uses): `equipment` declares a port for `bankreceipt`, `billingqueue`, `invoice`; `billing` provides it.
- **parties -> accounting** (2 uses): `parties` declares a port for `glaccount`, `accountingsync`; `accounting` provides it.
- **kernel -> identity** (2 uses): `kernel` declares a port for `tenant`; `identity` provides it.
- **equipment -> workforce** (2 uses): `equipment` declares a port for `worker`; `workforce` provides it.
- **reference -> equipment** (2 uses): `reference` declares a port for `equipmenttype`, `fleetcode`; `equipment` provides it.
- **identity -> accounting** (2 uses): `identity` declares a port for `journalentry`, `repositories.FiscalPeriodRepository`; `accounting` provides it.
- **identity -> parties** (2 uses): `identity` declares a port for `customer`, `customerservice`; `parties` provides it.
- **parties -> operations** (2 uses): `parties` declares a port for `shipment`, `tender`; `operations` provides it.
- **equipment -> settlement** (2 uses): `equipment` declares a port for `carriersettlement`, `driversettlement`; `settlement` provides it.
- **equipment -> documents** (2 uses): `equipment` declares a port for `document`, `inboundmessage`; `documents` provides it.
- **parties -> edi** (1 uses): `parties` declares a port for `edi`; `edi` provides it.
- **parties -> workforce** (1 uses): `parties` declares a port for `worker`; `workforce` provides it.
- **parties -> rating** (1 uses): `parties` declares a port for `fuelsurcharge`; `rating` provides it.
- **reference -> parties** (1 uses): `reference` declares a port for `customer`; `parties` provides it.
- **reference -> places** (1 uses): `reference` declares a port for `location`; `places` provides it.
- **places -> settlement** (1 uses): `places` declares a port for `iftacontract`; `settlement` provides it.
- **workspace -> billing** (1 uses): `workspace` declares a port for `repositories.InvoiceRepository`; `billing` provides it.
- **identity -> agent** (1 uses): `identity` declares a port for `agentdefinition`; `agent` provides it.
- **workspace -> accounting** (1 uses): `workspace` declares a port for `accountingsync`; `accounting` provides it.
- **workforce -> accounting** (1 uses): `workforce` declares a port for `accountingsync`; `accounting` provides it.
- **equipment -> agent** (1 uses): `equipment` declares a port for `insight`; `agent` provides it.
- **workforce -> settlement** (1 uses): `workforce` declares a port for `driverpay`; `settlement` provides it.

## Port symbols with no entity module
- services/?: 724
- services/agenttool: 540
- services/auth: 337
- services/toolpolicy: 176
- services/agentquerytool: 150
- services/agentruntime: 132
- services/agentcompletion: 125
- services/assistant: 113
- repositories/agentproposal: 81
- services/accountingsyncrecord: 75
- repositories/organization: 74
- services/realtime: 74
- repositories/accountingsyncrecord: 73
- repositories/?: 69
- services/agentmemory: 65
- repositories/agentrun: 60
- services/plan: 60
- repositories/agentmemory: 60
- services/retrieval: 57
- services/aidocument: 57
- repositories/user: 49
- services/workflowstarter: 49
- services/llmmessage: 49
- services/reporting: 47
- repositories/shipmentmove: 45
- services/accountingdrift: 41
- services/shipmentimportassistant: 41
- repositories/edimessage: 40
- services/agentevents: 39
- repositories/agentplan: 38
- repositories/assignment: 38
- repositories/billingqueuereview: 37
- repositories/agentevalcase: 36
- services/accountingdocument: 35
- repositories/shipmentboard: 35
- repositories/accountingdrift: 33
- services/retrievalquery: 33
- repositories/dispatchconsole: 33
- repositories/editemplate: 33
- repositories/shipmentcontrol: 31
- repositories/formulatemplateversion: 31
- repositories/edipartner: 30
- repositories/workerpto: 30
- services/agentreflection: 30
- services/runsteps: 30
- services/agentsafety: 30
- services/carrierintelprovider: 30
- services/rateengine: 29
- repositories/retrievalsource: 29
- repositories/role: 28
- repositories/workercredential: 26
- repositories/editransfer: 26
- repositories/carrierassignment: 26
- services/proposalpreview: 24
- repositories/agentdecision: 24
- services/platform: 24
- repositories/edimappingprofile: 24
- services/editransport: 24
- repositories/accountingledger: 23
- repositories/accountingmapping: 23
- repositories/journalposting: 23
- repositories/workerdrugalcohol: 22
- services/accountingmapping: 21
- repositories/ediinboundfile: 21
- repositories/workersafety: 21
- repositories/assistantturn: 21
- repositories/accountingcontrol: 20
- services/actor: 20
- services/dispatchconsole: 20
- repositories/ptoledger: 20
- repositories/agentdecisionqueue: 19
- services/agentevalcase: 18
- repositories/dataretention: 18
- repositories/agentexception: 18
- services/securityaudit: 18
- repositories/shipmentcomment: 18
- repositories/shipmenthold: 18
- repositories/ediconnection: 18
- repositories/edicommunicationprofile: 18
- repositories/agenttooltrust: 17
- repositories/edidocumentprofile: 17
- repositories/rbac: 16
- repositories/chargeallocation: 16
- services/telematicsprovider: 16
- repositories/agentevaluation: 15
- services/accountinginbound: 15
- repositories/ptopolicy: 15
- services/datatransformer: 14
- repositories/accountingpayables: 14
- repositories/accountinginbound: 14
- services/invoicedispute: 14
- services/airetrievalstatus: 14
- repositories/billingcontrol: 14
- repositories/tableinsight: 14
- repositories/edipartnersetting: 14
- repositories/routingguide: 14
- services/shipmentwatchlist: 14
- services/agentaccess: 13
- repositories/editenderchange: 13
- services/shipmentinvalidation: 13
- services/mfa: 13
- services/billingqueuereview: 13
- repositories/customfieldvalue: 13
- repositories/edishipmentlink: 13
- repositories/journalreview: 13
- repositories/roleagentgrant: 12
- repositories/agentcontrol: 12
- repositories/edicarrierinvoice: 12
- repositories/workerchecklist: 12
- repositories/performancereview: 12
- services/shipmentcomment: 12
- repositories/mfa: 12
- services/detentionbilling: 12
- services/orderderivation: 12
- repositories/editestcase: 12
- repositories/editransferchange: 11
- services/productguide: 11
- services/workerpto: 11
- repositories/invoicedispute: 11
- repositories/workertraining: 11
- repositories/aicontrolfacts: 11
- services/agenttoolrule: 11
- services/shipmentmove: 11
- services/journalpreview: 11
- repositories/edisourcecontext: 11
- services/shipmentbriefing: 11
- repositories/loadingoptimization: 11
- repositories/assistantqueue: 10
- services/agentcapability: 9
- services/embedding: 9
- repositories/agentrunevent: 9
- repositories/threadowner: 9
- services/documentintelligence: 9
- repositories/service_failure_reason_code: 9
- repositories/timesheet: 9
- repositories/agentreflection: 9
- services/assignment: 9
- services/email_sender: 9
- services/assistantcase: 9
- repositories/editenderrecipient: 9
- services/organization: 9
- repositories/agentproposalbaseline: 9
- repositories/permissioncache: 8
- repositories/agentdefinitionversion: 8
- services/agentdraft: 8
- repositories/carrierinvoicematch: 8
- repositories/workerleave: 8
- repositories/pagethread: 8
- repositories/documentcontrol: 8
- repositories/shipmentmovejurisdictionmile: 8
- repositories/edidocumenttype: 8
- services/shipmentquickfilter: 8
- repositories/tenant_sync: 8
- repositories/selfservice: 8
- services/accountingjournal: 7
- services/search: 7
- repositories/workerinjury: 7
- repositories/workerdqf: 7
- repositories/agentsubject: 7
- repositories/aiproviderkey: 7
- services/assistantturn: 7
- services/authevent: 7
- repositories/driverportal: 7
- repositories/syncedread: 7
- repositories/formulatemplatetestcase: 7
- repositories/agentrunstep: 7
- services/samsarasync: 7
- repositories/benefit: 7
- services/invoiceshare: 6
- repositories/agenttestprompt: 6
- repositories/aiproviderdismissal: 6
- repositories/assistantcase: 6
- services/shipmenteta: 6
- services/assistantstream: 6
- repositories/ssoidentitylink: 6
- services/settlementpayment: 6
- repositories/edicontrolnumber: 6
- repositories/formulatemplatereview: 6
- services/fuelcardprovider: 6
- services/journalreview: 6
- repositories/sequenceconfig: 6
- services/locationcode: 6
- repositories/roleassignment: 6
- repositories/agentactivity: 5
- repositories/scheduling: 5
- services/agentrunevent: 5
- repositories/agenttoolrule: 5
- repositories/sso: 5
- repositories/settlementcontrol: 5
- services/formula: 5
- repositories/orgholiday: 5
- repositories/journalrepair: 5
- repositories/agentmemorysubject: 4
- repositories/agentscorecard: 4
- repositories/casechecklist: 4
- services/pdfrenderer: 4
- services/recordanchor: 4
- repositories/carrierledger: 4
- repositories/dashcontrol: 4
- repositories/editransactionset: 4
- services/autocomplete: 4
- services/inboundcontent: 4
- repositories/invoiceadjustmentcontrol: 4
- repositories/shipmenttracking: 4
- repositories/pushsubscription: 4
- repositories/exchangeratestamp: 3
- services/agentactivity: 3
- services/agentevaluation: 3
- repositories/scopeverdictcache: 3
- repositories/accountsreceivable: 3
- services/accountsreceivable: 3
- services/agentscorecard: 3
- services/driver_notification: 3
- services/extractioncontract: 3
- services/assistanthandoff: 3
- repositories/authevent: 3
- repositories/loginthrottle: 3
- repositories/carriersettlementcontrol: 3
- services/fuelcardfeed: 3
- repositories/tenantbootstrap: 3
- repositories/shipmentboardcache: 3
- services/officedocument: 3
- services/pdfreader: 3
- repositories/workeremploymentevent: 3
- repositories/aranalytics: 2
- services/agenttestprompt: 2
- services/shipmenthold: 2
- repositories/aicontrolsummarycache: 2
- repositories/modelpricecache: 2
- repositories/auditbuffer: 2
- services/carrierinvoicematching: 2
- repositories/passwordreset: 2
- services/platformemail: 2
- services/shipmentboard: 2
- repositories/weatheralertfeedstate: 2
- repositories/auditdlq: 2
- repositories/invoiceshare: 2
- repositories/shipmentadditionalcharge: 2
- repositories/shipmentcommodity: 2
- services/accountingchange: 1
- repositories/aituneupfreshness: 1
- services/carriercostaccrual: 1
- services/employment: 1
