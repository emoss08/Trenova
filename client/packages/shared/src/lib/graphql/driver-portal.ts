import {
  CancelMyExpenseDocument,
  DashControlDocument,
  CancelMyPtoDocument,
  CreateMyLoadCommentDocument,
  DriverExpenseDetailDocument,
  DriverExpenseTableDocument,
  MyComplianceProfileDocument,
  MyCredentialsDocument,
  MyTrainingDocument,
  MySafetyScorecardDocument,
  MyRecognitionsDocument,
  MyDisciplinaryActionsDocument,
  AcknowledgeMyDisciplinaryActionDocument,
  MyReviewsDocument,
  AcknowledgeMyReviewDocument,
  StartMyTrainingDocument,
  AcknowledgeMyTrainingDocument,
  MyExpensesDocument,
  MyLoadPayEstimateDocument,
  MyPortalFeaturesDocument,
  MyPtoBalancesDocument,
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
  type MyComplianceProfileQuery,
  type MyCredentialsQuery,
  type MyTrainingQuery,
  type MySafetyScorecardQuery,
  type MyRecognitionsQuery,
  type MyDisciplinaryActionsQuery,
  type AcknowledgeMyDisciplinaryActionMutation,
  type AcknowledgeMyDisciplinaryActionMutationVariables,
  type MyReviewsQuery,
  type AcknowledgeMyReviewMutation,
  type AcknowledgeMyReviewMutationVariables,
  type PortalTrainingFieldsFragment,
  type StartMyTrainingMutation,
  type StartMyTrainingMutationVariables,
  type AcknowledgeMyTrainingMutation,
  type AcknowledgeMyTrainingMutationVariables,
  type MyExpensesQuery,
  type DashControlQuery,
  type MyLoadPayEstimateQuery,
  type MyPortalFeaturesQuery,
  type MyPtoBalancesQuery,
  AcknowledgeMyPolicyDocument,
  MyAvailabilityDocument,
  MyPoliciesDocument,
  MyPolicyDocumentUrlDocument,
  MyProfileChangeRequestsDocument,
  WithdrawMyProfileChangeDocument,
  MyLeaveDocument,
  MyScheduleDocument,
  MyShiftSwapsDocument,
  ProposeMyShiftSwapDocument,
  RespondToMyShiftSwapDocument,
  SetMyAvailabilityDocument,
  type AcknowledgeMyPolicyInput,
  type AcknowledgeMyPolicyMutation,
  type AcknowledgeMyPolicyMutationVariables,
  type MyAvailabilityQuery,
  type MyPoliciesQuery,
  type MyPolicyDocumentUrlQuery,
  type MyPolicyDocumentUrlQueryVariables,
  type MyProfileChangeRequestsQuery,
  type WithdrawMyProfileChangeMutation,
  type WithdrawMyProfileChangeMutationVariables,
  type MyLeaveQuery,
  type MyScheduleQuery,
  type MyScheduleQueryVariables,
  type MyShiftSwapsQuery,
  type ProposeMyShiftSwapInput,
  type ProposeMyShiftSwapMutation,
  type ProposeMyShiftSwapMutationVariables,
  type RespondToMyShiftSwapInput,
  type RespondToMyShiftSwapMutation,
  type RespondToMyShiftSwapMutationVariables,
  type SetMyAvailabilityInput,
  type SetMyAvailabilityMutation,
  type SetMyAvailabilityMutationVariables,
  MyTotalCompensationDocument,
  type MyTotalCompensationQuery,
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
export type PortalCredential = MyCredentialsQuery["myCredentials"][number];
export type PortalTraining = PortalTrainingFieldsFragment;
export type PortalSafetyScorecard = MySafetyScorecardQuery["mySafetyScorecard"];
export type PortalRecognition = MyRecognitionsQuery["myRecognitions"][number];
export type PortalDisciplinaryAction = MyDisciplinaryActionsQuery["myDisciplinaryActions"][number];
export type PortalReview = MyReviewsQuery["myReviews"][number];
export type PortalPtoBalance = MyPtoBalancesQuery["myPtoBalances"][number];
export type PortalLeave = MyLeaveQuery["myLeave"];
export type PortalTotalCompensation = MyTotalCompensationQuery["myTotalCompensation"];
export type PortalLeaveCase = PortalLeave["cases"][number];
export type PortalExpense = MyExpensesQuery["myExpenses"][number];
export type PortalPayEstimate = MyLoadPayEstimateQuery["myLoadPayEstimate"];
export type PortalYtdPay = MyYtdPayQuery["myYtdPay"];
export type PortalFeatures = MyPortalFeaturesQuery["myPortalFeatures"];
export type DashControl = DashControlQuery["dashControl"];
export type DriverExpenseRow = NonNullable<
  DriverExpenseTableQuery["driverExpenses"]["edges"]
>[number]["node"];
export type DriverExpenseDetail = NonNullable<DriverExpenseDetailQuery["driverExpense"]>;

export const settlementDisputeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: SettlementDisputeTableDocument,
  operationName: "SettlementDisputeTable",
  connectionKey: "settlementDisputes",
});

export async function fetchWorkerPortalStatus(
  workerId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: WorkerPortalStatusDocument,
    operationName: "WorkerPortalStatus",
    variables: { workerId },
    signal: options?.signal,
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

export async function fetchSettlementDisputeDetail(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: SettlementDisputeDetailDocument,
    operationName: "SettlementDisputeDetail",
    variables: { id },
    signal: options?.signal,
  });
  return data.settlementDispute;
}

export async function fetchOpenSettlementDisputeCount(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: OpenSettlementDisputeCountDocument,
    operationName: "OpenSettlementDisputeCount",
    signal: options?.signal,
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

export async function fetchMyPortalProfile(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyPortalProfileDocument,
    operationName: "MyPortalProfile",
    signal: options?.signal,
  });
  return data.myPortalProfile;
}

export async function fetchMyLoads(
  scope: PortalLoadScope,
  limit?: number,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: MyLoadsDocument,
    operationName: "MyLoads",
    variables: { scope, limit },
    signal: options?.signal,
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

export async function fetchMyLoadComments(shipmentId: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyLoadCommentsDocument,
    operationName: "MyLoadComments",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.myLoadComments;
}

export async function fetchMyPeriodSummary(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyPeriodSummaryDocument,
    operationName: "MyPeriodSummary",
    signal: options?.signal,
  });
  return data.myPeriodSummary;
}

export async function fetchMyRecentPayEvents(limit?: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyRecentPayEventsDocument,
    operationName: "MyRecentPayEvents",
    variables: { limit },
    signal: options?.signal,
  });
  return data.myRecentPayEvents;
}

export async function fetchMySettlements(
  limit?: number,
  offset?: number,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: MySettlementsDocument,
    operationName: "MySettlements",
    variables: { limit, offset },
    signal: options?.signal,
  });
  return data.mySettlements;
}

export async function fetchMySettlement(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MySettlementDocument,
    operationName: "MySettlement",
    variables: { id },
    signal: options?.signal,
  });
  return data.mySettlement;
}

export async function fetchMyEscrow(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyEscrowDocument,
    operationName: "MyEscrow",
    signal: options?.signal,
  });
  return data.myEscrow;
}

export async function fetchMyAdvances(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyAdvancesDocument,
    operationName: "MyAdvances",
    signal: options?.signal,
  });
  return data.myAdvances;
}

export async function fetchMyDisputes(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyDisputesDocument,
    operationName: "MyDisputes",
    signal: options?.signal,
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

export const driverExpenseTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DriverExpenseTableDocument,
  operationName: "DriverExpenseTable",
  connectionKey: "driverExpenses",
});

export async function fetchMyComplianceProfile(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyComplianceProfileDocument,
    operationName: "MyComplianceProfile",
    signal: options?.signal,
  });
  return data.myComplianceProfile;
}

export async function fetchMyTraining(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyTrainingQuery>({
    document: MyTrainingDocument,
    operationName: "MyTraining",
    signal: options?.signal,
  });
  return data.myTraining as PortalTraining[];
}

export async function startMyTraining(id: string) {
  const data = await requestGraphQL<StartMyTrainingMutation, StartMyTrainingMutationVariables>({
    document: StartMyTrainingDocument,
    operationName: "StartMyTraining",
    variables: { id },
  });
  return data.startMyTraining as PortalTraining;
}

export async function acknowledgeMyTraining(id: string) {
  const data = await requestGraphQL<
    AcknowledgeMyTrainingMutation,
    AcknowledgeMyTrainingMutationVariables
  >({
    document: AcknowledgeMyTrainingDocument,
    operationName: "AcknowledgeMyTraining",
    variables: { id },
  });
  return data.acknowledgeMyTraining as PortalTraining;
}

export async function fetchMySafetyScorecard(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MySafetyScorecardQuery>({
    document: MySafetyScorecardDocument,
    operationName: "MySafetyScorecard",
    signal: options?.signal,
  });
  return data.mySafetyScorecard;
}

export async function fetchMyRecognitions(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyRecognitionsQuery>({
    document: MyRecognitionsDocument,
    operationName: "MyRecognitions",
    signal: options?.signal,
  });
  return data.myRecognitions;
}

export async function fetchMyDisciplinaryActions(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyDisciplinaryActionsQuery>({
    document: MyDisciplinaryActionsDocument,
    operationName: "MyDisciplinaryActions",
    signal: options?.signal,
  });
  return data.myDisciplinaryActions;
}

export async function acknowledgeMyDisciplinaryAction(id: string, comment?: string) {
  const data = await requestGraphQL<
    AcknowledgeMyDisciplinaryActionMutation,
    AcknowledgeMyDisciplinaryActionMutationVariables
  >({
    document: AcknowledgeMyDisciplinaryActionDocument,
    operationName: "AcknowledgeMyDisciplinaryAction",
    variables: { id, comment },
  });
  return data.acknowledgeMyDisciplinaryAction;
}

export async function fetchMyReviews(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyReviewsQuery>({
    document: MyReviewsDocument,
    operationName: "MyReviews",
    signal: options?.signal,
  });
  return data.myReviews;
}

export async function acknowledgeMyReview(id: string, comment?: string) {
  const data = await requestGraphQL<
    AcknowledgeMyReviewMutation,
    AcknowledgeMyReviewMutationVariables
  >({
    document: AcknowledgeMyReviewDocument,
    operationName: "AcknowledgeMyReview",
    variables: { id, comment },
  });
  return data.acknowledgeMyReview;
}

export async function fetchMyCredentials(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyCredentialsQuery>({
    document: MyCredentialsDocument,
    operationName: "MyCredentials",
    signal: options?.signal,
  });
  return data.myCredentials;
}

export async function updateMyContactInfo(input: UpdateMyContactInfoInput) {
  const data = await requestGraphQL({
    document: UpdateMyContactInfoDocument,
    operationName: "UpdateMyContactInfo",
    variables: { input },
  });
  return data.updateMyContactInfo;
}

export async function fetchMyPto(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyPtoDocument,
    operationName: "MyPto",
    signal: options?.signal,
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

export async function fetchMyPtoBalances(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyPtoBalancesQuery>({
    document: MyPtoBalancesDocument,
    operationName: "MyPtoBalances",
    signal: options?.signal,
  });
  return data.myPtoBalances;
}

export async function fetchMyLeave(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyLeaveQuery>({
    document: MyLeaveDocument,
    operationName: "MyLeave",
    signal: options?.signal,
  });
  return data.myLeave;
}

export type MySchedule = MyScheduleQuery["mySchedule"];
export type MyScheduleDay = MySchedule["days"][number];
export type MyAvailabilityRow = MyAvailabilityQuery["myAvailability"][number];
export type MyShiftSwapRow = MyShiftSwapsQuery["myShiftSwaps"][number];

export async function fetchMySchedule(
  args: { at?: number; weeks?: number } = {},
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL<MyScheduleQuery, MyScheduleQueryVariables>({
    document: MyScheduleDocument,
    operationName: "MySchedule",
    variables: { at: args.at ?? null, weeks: args.weeks ?? null },
    signal: options?.signal,
  });
  return data.mySchedule;
}

export async function fetchMyAvailability(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyAvailabilityQuery>({
    document: MyAvailabilityDocument,
    operationName: "MyAvailability",
    signal: options?.signal,
  });
  return data.myAvailability;
}

export async function fetchMyShiftSwaps(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyShiftSwapsQuery>({
    document: MyShiftSwapsDocument,
    operationName: "MyShiftSwaps",
    signal: options?.signal,
  });
  return data.myShiftSwaps;
}

export async function setMyAvailability(input: SetMyAvailabilityInput) {
  const data = await requestGraphQL<SetMyAvailabilityMutation, SetMyAvailabilityMutationVariables>({
    document: SetMyAvailabilityDocument,
    operationName: "SetMyAvailability",
    variables: { input },
  });
  return data.setMyAvailability;
}

export async function proposeMyShiftSwap(input: ProposeMyShiftSwapInput) {
  const data = await requestGraphQL<
    ProposeMyShiftSwapMutation,
    ProposeMyShiftSwapMutationVariables
  >({
    document: ProposeMyShiftSwapDocument,
    operationName: "ProposeMyShiftSwap",
    variables: { input },
  });
  return data.proposeMyShiftSwap;
}

export async function respondToMyShiftSwap(input: RespondToMyShiftSwapInput) {
  const data = await requestGraphQL<
    RespondToMyShiftSwapMutation,
    RespondToMyShiftSwapMutationVariables
  >({
    document: RespondToMyShiftSwapDocument,
    operationName: "RespondToMyShiftSwap",
    variables: { input },
  });
  return data.respondToMyShiftSwap;
}

export type MyPolicy = MyPoliciesQuery["myPolicies"][number];
export type MyProfileChangeRequest =
  MyProfileChangeRequestsQuery["myProfileChangeRequests"][number];

export async function fetchMyPolicies(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyPoliciesQuery>({
    document: MyPoliciesDocument,
    operationName: "MyPolicies",
    signal: options?.signal,
  });
  return data.myPolicies;
}

export async function fetchMyPolicyDocumentUrl(
  policyId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL<MyPolicyDocumentUrlQuery, MyPolicyDocumentUrlQueryVariables>({
    document: MyPolicyDocumentUrlDocument,
    operationName: "MyPolicyDocumentUrl",
    variables: { policyId },
    signal: options?.signal,
  });
  return data.myPolicyDocumentUrl;
}

export async function fetchMyProfileChangeRequests(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyProfileChangeRequestsQuery>({
    document: MyProfileChangeRequestsDocument,
    operationName: "MyProfileChangeRequests",
    signal: options?.signal,
  });
  return data.myProfileChangeRequests;
}

export async function acknowledgeMyPolicy(input: AcknowledgeMyPolicyInput) {
  const data = await requestGraphQL<
    AcknowledgeMyPolicyMutation,
    AcknowledgeMyPolicyMutationVariables
  >({
    document: AcknowledgeMyPolicyDocument,
    operationName: "AcknowledgeMyPolicy",
    variables: { input },
  });
  return data.acknowledgeMyPolicy;
}

export async function withdrawMyProfileChange(id: string) {
  const data = await requestGraphQL<
    WithdrawMyProfileChangeMutation,
    WithdrawMyProfileChangeMutationVariables
  >({
    document: WithdrawMyProfileChangeDocument,
    operationName: "WithdrawMyProfileChange",
    variables: { id },
  });
  return data.withdrawMyProfileChange;
}

export async function fetchMyTotalCompensation(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL<MyTotalCompensationQuery>({
    document: MyTotalCompensationDocument,
    operationName: "MyTotalCompensation",
    signal: options?.signal,
  });
  return data.myTotalCompensation;
}

export async function cancelMyPto(id: string) {
  const data = await requestGraphQL({
    document: CancelMyPtoDocument,
    operationName: "CancelMyPto",
    variables: { id },
  });
  return data.cancelMyPto;
}

export async function fetchMyExpenses(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyExpensesDocument,
    operationName: "MyExpenses",
    signal: options?.signal,
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

export async function fetchMyLoadPayEstimate(
  shipmentId: string,
  moveId: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: MyLoadPayEstimateDocument,
    operationName: "MyLoadPayEstimate",
    variables: { shipmentId, moveId },
    signal: options?.signal,
  });
  return data.myLoadPayEstimate;
}

export async function fetchMyYtdPay(year: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyYtdPayDocument,
    operationName: "MyYtdPay",
    variables: { year },
    signal: options?.signal,
  });
  return data.myYtdPay;
}

export async function fetchDriverExpenseDetail(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: DriverExpenseDetailDocument,
    operationName: "DriverExpenseDetail",
    variables: { id },
    signal: options?.signal,
  });
  return data.driverExpense;
}

export async function fetchPendingDriverExpenseCount(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: PendingDriverExpenseCountDocument,
    operationName: "PendingDriverExpenseCount",
    signal: options?.signal,
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

export async function fetchMyPortalFeatures(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyPortalFeaturesDocument,
    operationName: "MyPortalFeatures",
    signal: options?.signal,
  });
  return data.myPortalFeatures;
}

export async function fetchDashControl(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: DashControlDocument,
    operationName: "DashControl",
    signal: options?.signal,
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

export async function fetchMyHosState(options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyHosStateDocument,
    operationName: "MyHosState",
    signal: options?.signal,
  });
  return data.myHosState;
}

export async function fetchMyHosDailyLogs(
  startDate: string,
  endDate: string,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: MyHosDailyLogsDocument,
    operationName: "MyHosDailyLogs",
    variables: { startDate, endDate },
    signal: options?.signal,
  });
  return data.myHosDailyLogs;
}

export async function fetchMyHosViolations(since?: number, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: MyHosViolationsDocument,
    operationName: "MyHosViolations",
    variables: { since },
    signal: options?.signal,
  });
  return data.myHosViolations;
}
