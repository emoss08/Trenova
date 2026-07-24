import {
  CancelMyExpenseDocument,
  DashControlDocument,
  CancelMyPtoDocument,
  CreateMyLoadCommentDocument,
  DriverExpenseDetailDocument,
  DriverExpenseTableDocument,
  MyComplianceProfileDocument,
  MyExpensesDocument,
  MyLoadPayEstimateDocument,
  MyPortalFeaturesDocument,
  MyPtoDocument,
  MyYtdPayDocument,
  PendingDriverExpenseCountDocument,
  RequestMyPtoDocument,
  RespondToMyAssignmentDocument,
  ReviewDriverExpenseDocument,
  SubmitMyExpenseDocument,
  UpdateDashControlDocument,
  UpdateMyContactInfoDocument,
  CreateSettlementDisputeDocument,
  InviteWorkerToPortalDocument,
  MyAdvancesDocument,
  MyEscrowDocument,
  MyHosDailyLogsDocument,
  MyHosStateDocument,
  MyHosViolationsDocument,
  MyLoadCommentsDocument,
  MyLoadsDocument,
  MyPeriodSummaryDocument,
  MyPortalProfileDocument,
  MyRecentPayEventsDocument,
  RecordMyStopActionDocument,
  MySettlementDocument,
  MySettlementsDocument,
  MyDisputesDocument,
  OpenSettlementDisputeCountDocument,
  ResolveSettlementDisputeDocument,
  RevokeWorkerPortalAccessDocument,
  SettlementDisputeDetailDocument,
  SettlementDisputeTableDocument,
  StartSettlementDisputeReviewDocument,
  WithdrawSettlementDisputeDocument,
  WorkerPortalStatusDocument,
  type CreateMyLoadCommentInput,
  type DriverExpenseDetailQuery,
  type DriverExpenseTableQuery,
  type DriverExpenseTableQueryVariables,
  type MyComplianceProfileQuery,
  type MyExpensesQuery,
  type DashControlQuery,
  type MyLoadPayEstimateQuery,
  type MyPortalFeaturesQuery,
  type MyPtoQuery,
  type MyYtdPayQuery,
  type RequestMyPtoInput,
  type RespondToMyAssignmentInput,
  type ReviewDriverExpenseInput,
  type SubmitMyExpenseInput,
  type UpdateDashControlInput,
  type UpdateMyContactInfoInput,
  type CreateSettlementDisputeInput,
  type InviteWorkerToPortalInput,
  type MyAdvancesQuery,
  type MyDisputesQuery,
  type MyEscrowQuery,
  type MyHosDailyLogsQuery,
  type MyHosStateQuery,
  type MyHosViolationsQuery,
  type MyLoadCommentsQuery,
  type MyLoadsQuery,
  type MyPeriodSummaryQuery,
  type MyPortalProfileQuery,
  type MyRecentPayEventsQuery,
  type MySettlementQuery,
  type MySettlementsQuery,
  type PortalLoadScope,
  type RecordMyStopActionInput,
  type ResolveSettlementDisputeInput,
  type SettlementDisputeDetailQuery,
  type SettlementDisputeTableQuery,
  type SettlementDisputeTableQueryVariables,
  type WorkerPortalStatusQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type WorkerPortalStatus = WorkerPortalStatusQuery["workerPortalStatus"];
export type PortalInvitationRow = WorkerPortalStatus["invitations"][number];
export type SettlementDisputeRow = NonNullable<
  SettlementDisputeTableQuery["settlementDisputes"]["edges"]
>[number]["node"];
export type SettlementDisputeDetail = NonNullable<
  SettlementDisputeDetailQuery["settlementDispute"]
>;
export type PortalProfile = MyPortalProfileQuery["myPortalProfile"];
export type PortalLoad = MyLoadsQuery["myLoads"][number];
export type PortalStop = PortalLoad["stops"][number];
export type PortalPeriodSummary = MyPeriodSummaryQuery["myPeriodSummary"];
export type PortalLoadComment = MyLoadCommentsQuery["myLoadComments"][number];
export type PortalPayEvent = MyRecentPayEventsQuery["myRecentPayEvents"][number];
export type PortalSettlementSummary = MySettlementsQuery["mySettlements"]["items"][number];
export type PortalSettlementDetail = NonNullable<MySettlementQuery["mySettlement"]>;
export type PortalSettlementLine = NonNullable<PortalSettlementDetail["lines"]>[number];
export type PortalEscrow = MyEscrowQuery["myEscrow"];
export type PortalAdvance = MyAdvancesQuery["myAdvances"][number];
export type PortalDispute = MyDisputesQuery["myDisputes"][number];
export type PortalComplianceProfile = MyComplianceProfileQuery["myComplianceProfile"];
export type PortalPtoRow = MyPtoQuery["myPto"][number];
export type PortalExpense = MyExpensesQuery["myExpenses"][number];
export type PortalPayEstimate = MyLoadPayEstimateQuery["myLoadPayEstimate"];
export type PortalYtdPay = MyYtdPayQuery["myYtdPay"];
export type PortalFeatures = MyPortalFeaturesQuery["myPortalFeatures"];
export type DashControl = DashControlQuery["dashControl"];
export type DriverExpenseRow = NonNullable<
  DriverExpenseTableQuery["driverExpenses"]["edges"]
>[number]["node"];
export type DriverExpenseDetail = NonNullable<DriverExpenseDetailQuery["driverExpense"]>;

export const settlementDisputeTableGraphQLConfig = defineDataTableGraphQLConfig<
  SettlementDisputeRow,
  SettlementDisputeTableQueryVariables
>({
  document: SettlementDisputeTableDocument,
  operationName: "SettlementDisputeTable",
  connectionKey: "settlementDisputes",
});

export async function fetchWorkerPortalStatus(workerId: string) {
  const data = await requestGraphQL({
    document: WorkerPortalStatusDocument,
    operationName: "WorkerPortalStatus",
    variables: { workerId },
  });
  return data.workerPortalStatus;
}

export async function inviteWorkerToPortal(input: InviteWorkerToPortalInput) {
  const data = await requestGraphQL({
    document: InviteWorkerToPortalDocument,
    operationName: "InviteWorkerToPortal",
    variables: { input },
  });
  return data.inviteWorkerToPortal;
}

export async function revokeWorkerPortalAccess(workerId: string) {
  const data = await requestGraphQL({
    document: RevokeWorkerPortalAccessDocument,
    operationName: "RevokeWorkerPortalAccess",
    variables: { workerId },
  });
  return data.revokeWorkerPortalAccess;
}

export async function fetchSettlementDisputeDetail(id: string) {
  const data = await requestGraphQL({
    document: SettlementDisputeDetailDocument,
    operationName: "SettlementDisputeDetail",
    variables: { id },
  });
  return data.settlementDispute;
}

export async function fetchOpenSettlementDisputeCount() {
  const data = await requestGraphQL({
    document: OpenSettlementDisputeCountDocument,
    operationName: "OpenSettlementDisputeCount",
  });
  return data.openSettlementDisputeCount;
}

export async function startSettlementDisputeReview(id: string) {
  const data = await requestGraphQL({
    document: StartSettlementDisputeReviewDocument,
    operationName: "StartSettlementDisputeReview",
    variables: { id },
  });
  return data.startSettlementDisputeReview;
}

export async function resolveSettlementDispute(input: ResolveSettlementDisputeInput) {
  const data = await requestGraphQL({
    document: ResolveSettlementDisputeDocument,
    operationName: "ResolveSettlementDispute",
    variables: { input },
  });
  return data.resolveSettlementDispute;
}

export async function fetchMyPortalProfile() {
  const data = await requestGraphQL({
    document: MyPortalProfileDocument,
    operationName: "MyPortalProfile",
  });
  return data.myPortalProfile;
}

export async function fetchMyLoads(scope: PortalLoadScope, limit?: number) {
  const data = await requestGraphQL({
    document: MyLoadsDocument,
    operationName: "MyLoads",
    variables: { scope, limit },
  });
  return data.myLoads;
}

export async function recordMyStopAction(input: RecordMyStopActionInput) {
  const data = await requestGraphQL({
    document: RecordMyStopActionDocument,
    operationName: "RecordMyStopAction",
    variables: { input },
  });
  return data.recordMyStopAction;
}

export async function createMyLoadComment(input: CreateMyLoadCommentInput) {
  const data = await requestGraphQL({
    document: CreateMyLoadCommentDocument,
    operationName: "CreateMyLoadComment",
    variables: { input },
  });
  return data.createMyLoadComment;
}

export async function fetchMyLoadComments(shipmentId: string) {
  const data = await requestGraphQL({
    document: MyLoadCommentsDocument,
    operationName: "MyLoadComments",
    variables: { shipmentId },
  });
  return data.myLoadComments;
}

export async function fetchMyPeriodSummary() {
  const data = await requestGraphQL({
    document: MyPeriodSummaryDocument,
    operationName: "MyPeriodSummary",
  });
  return data.myPeriodSummary;
}

export async function fetchMyRecentPayEvents(limit?: number) {
  const data = await requestGraphQL({
    document: MyRecentPayEventsDocument,
    operationName: "MyRecentPayEvents",
    variables: { limit },
  });
  return data.myRecentPayEvents;
}

export async function fetchMySettlements(limit?: number, offset?: number) {
  const data = await requestGraphQL({
    document: MySettlementsDocument,
    operationName: "MySettlements",
    variables: { limit, offset },
  });
  return data.mySettlements;
}

export async function fetchMySettlement(id: string) {
  const data = await requestGraphQL({
    document: MySettlementDocument,
    operationName: "MySettlement",
    variables: { id },
  });
  return data.mySettlement;
}

export async function fetchMyEscrow() {
  const data = await requestGraphQL({
    document: MyEscrowDocument,
    operationName: "MyEscrow",
  });
  return data.myEscrow;
}

export async function fetchMyAdvances() {
  const data = await requestGraphQL({
    document: MyAdvancesDocument,
    operationName: "MyAdvances",
  });
  return data.myAdvances;
}

export async function fetchMyDisputes() {
  const data = await requestGraphQL({
    document: MyDisputesDocument,
    operationName: "MyDisputes",
  });
  return data.myDisputes;
}

export async function createSettlementDispute(input: CreateSettlementDisputeInput) {
  const data = await requestGraphQL({
    document: CreateSettlementDisputeDocument,
    operationName: "CreateSettlementDispute",
    variables: { input },
  });
  return data.createSettlementDispute;
}

export async function withdrawSettlementDispute(id: string) {
  const data = await requestGraphQL({
    document: WithdrawSettlementDisputeDocument,
    operationName: "WithdrawSettlementDispute",
    variables: { id },
  });
  return data.withdrawSettlementDispute;
}

export const driverExpenseTableGraphQLConfig = defineDataTableGraphQLConfig<
  DriverExpenseRow,
  DriverExpenseTableQueryVariables
>({
  document: DriverExpenseTableDocument,
  operationName: "DriverExpenseTable",
  connectionKey: "driverExpenses",
});

export async function fetchMyComplianceProfile() {
  const data = await requestGraphQL({
    document: MyComplianceProfileDocument,
    operationName: "MyComplianceProfile",
  });
  return data.myComplianceProfile;
}

export async function updateMyContactInfo(input: UpdateMyContactInfoInput) {
  const data = await requestGraphQL({
    document: UpdateMyContactInfoDocument,
    operationName: "UpdateMyContactInfo",
    variables: { input },
  });
  return data.updateMyContactInfo;
}

export async function fetchMyPto() {
  const data = await requestGraphQL({
    document: MyPtoDocument,
    operationName: "MyPto",
  });
  return data.myPto;
}

export async function requestMyPto(input: RequestMyPtoInput) {
  const data = await requestGraphQL({
    document: RequestMyPtoDocument,
    operationName: "RequestMyPto",
    variables: { input },
  });
  return data.requestMyPto;
}

export async function cancelMyPto(id: string) {
  const data = await requestGraphQL({
    document: CancelMyPtoDocument,
    operationName: "CancelMyPto",
    variables: { id },
  });
  return data.cancelMyPto;
}

export async function fetchMyExpenses() {
  const data = await requestGraphQL({
    document: MyExpensesDocument,
    operationName: "MyExpenses",
  });
  return data.myExpenses;
}

export async function submitMyExpense(input: SubmitMyExpenseInput) {
  const data = await requestGraphQL({
    document: SubmitMyExpenseDocument,
    operationName: "SubmitMyExpense",
    variables: { input },
  });
  return data.submitMyExpense;
}

export async function cancelMyExpense(id: string) {
  const data = await requestGraphQL({
    document: CancelMyExpenseDocument,
    operationName: "CancelMyExpense",
    variables: { id },
  });
  return data.cancelMyExpense;
}

export async function respondToMyAssignment(input: RespondToMyAssignmentInput) {
  const data = await requestGraphQL({
    document: RespondToMyAssignmentDocument,
    operationName: "RespondToMyAssignment",
    variables: { input },
  });
  return data.respondToMyAssignment;
}

export async function fetchMyLoadPayEstimate(shipmentId: string, moveId: string) {
  const data = await requestGraphQL({
    document: MyLoadPayEstimateDocument,
    operationName: "MyLoadPayEstimate",
    variables: { shipmentId, moveId },
  });
  return data.myLoadPayEstimate;
}

export async function fetchMyYtdPay(year: number) {
  const data = await requestGraphQL({
    document: MyYtdPayDocument,
    operationName: "MyYtdPay",
    variables: { year },
  });
  return data.myYtdPay;
}

export async function fetchDriverExpenseDetail(id: string) {
  const data = await requestGraphQL({
    document: DriverExpenseDetailDocument,
    operationName: "DriverExpenseDetail",
    variables: { id },
  });
  return data.driverExpense;
}

export async function fetchPendingDriverExpenseCount() {
  const data = await requestGraphQL({
    document: PendingDriverExpenseCountDocument,
    operationName: "PendingDriverExpenseCount",
  });
  return data.pendingDriverExpenseCount;
}

export async function reviewDriverExpense(input: ReviewDriverExpenseInput) {
  const data = await requestGraphQL({
    document: ReviewDriverExpenseDocument,
    operationName: "ReviewDriverExpense",
    variables: { input },
  });
  return data.reviewDriverExpense;
}

export async function fetchMyPortalFeatures() {
  const data = await requestGraphQL({
    document: MyPortalFeaturesDocument,
    operationName: "MyPortalFeatures",
  });
  return data.myPortalFeatures;
}

export async function fetchDashControl() {
  const data = await requestGraphQL({
    document: DashControlDocument,
    operationName: "DashControl",
  });
  return data.dashControl;
}

export async function updateDashControl(input: UpdateDashControlInput) {
  const data = await requestGraphQL({
    document: UpdateDashControlDocument,
    operationName: "UpdateDashControl",
    variables: { input },
  });
  return data.updateDashControl;
}

export type MyHosState = NonNullable<MyHosStateQuery["myHosState"]>;
export type MyHosDailyLog = MyHosDailyLogsQuery["myHosDailyLogs"][number];
export type MyHosViolation = MyHosViolationsQuery["myHosViolations"][number];

export async function fetchMyHosState() {
  const data = await requestGraphQL({
    document: MyHosStateDocument,
    operationName: "MyHosState",
  });
  return data.myHosState;
}

export async function fetchMyHosDailyLogs(startDate: string, endDate: string) {
  const data = await requestGraphQL({
    document: MyHosDailyLogsDocument,
    operationName: "MyHosDailyLogs",
    variables: { startDate, endDate },
  });
  return data.myHosDailyLogs;
}

export async function fetchMyHosViolations(since?: number) {
  const data = await requestGraphQL({
    document: MyHosViolationsDocument,
    operationName: "MyHosViolations",
    variables: { since },
  });
  return data.myHosViolations;
}
