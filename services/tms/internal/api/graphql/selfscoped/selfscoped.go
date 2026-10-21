// Package selfscoped names the root resolvers that act only on the caller's
// own records. authzlint lets them through without a permission check, and
// the field guard leaves what they return ungated by nested kind, because
// every record under them is the caller's own.
package selfscoped

import "strings"

// Resolvers lists, by the reason each holds, the resolvers that act only on the
// caller's own records and so need no permission beyond being signed in. Every
// name is listed: a resolver called My<Thing> earns no exemption by its name.
var Resolvers = map[string][]string{
	"the driver portal resolves the worker linked to the calling user " +
		"(driverportalservice.ResolveWorker) and reads or acts only on that worker": {
		"MutationResolver.AcknowledgeMyDisciplinaryAction",
		"MutationResolver.AcknowledgeMyPolicy",
		"MutationResolver.AcknowledgeMyReview",
		"MutationResolver.AcknowledgeMyTraining",
		"MutationResolver.CancelMyExpense",
		"MutationResolver.CancelMyPTO",
		"MutationResolver.CreateMyLoadComment",
		"MutationResolver.DismissMyNotifications",
		"MutationResolver.MarkAllMyNotificationsRead",
		"MutationResolver.MarkMyNotificationsRead",
		"MutationResolver.MarkMyNotificationsUnread",
		"MutationResolver.ProposeMyShiftSwap",
		"MutationResolver.RecordMyStopAction",
		"MutationResolver.RequestMyPTO",
		"MutationResolver.RespondToMyAssignment",
		"MutationResolver.RespondToMyShiftSwap",
		"MutationResolver.RestoreMyNotifications",
		"MutationResolver.SetMyAvailability",
		"MutationResolver.StartMyTraining",
		"MutationResolver.SubmitMyExpense",
		"MutationResolver.UpdateMyContactInfo",
		"MutationResolver.WithdrawMyProfileChange",
		"QueryResolver.MyAdvances",
		"QueryResolver.MyAvailability",
		"QueryResolver.MyComplianceProfile",
		"QueryResolver.MyCredentials",
		"QueryResolver.MyDisciplinaryActions",
		"QueryResolver.MyDisputes",
		"QueryResolver.MyEscrow",
		"QueryResolver.MyExpenses",
		"QueryResolver.MyHosDailyLogs",
		"QueryResolver.MyHosState",
		"QueryResolver.MyHosViolations",
		"QueryResolver.MyLeave",
		"QueryResolver.MyLoadComments",
		"QueryResolver.MyLoadPayEstimate",
		"QueryResolver.MyLoads",
		"QueryResolver.MyNotificationUnreadCount",
		"QueryResolver.MyNotifications",
		"QueryResolver.MyPTO",
		"QueryResolver.MyPTOBalances",
		"QueryResolver.MyPeriodSummary",
		"QueryResolver.MyPolicies",
		"QueryResolver.MyPolicyDocumentURL",
		"QueryResolver.MyPortalFeatures",
		"QueryResolver.MyPortalProfile",
		"QueryResolver.MyProfileChangeRequests",
		"QueryResolver.MyRecentPayEvents",
		"QueryResolver.MyRecognitions",
		"QueryResolver.MyReviews",
		"QueryResolver.MySafetyScorecard",
		"QueryResolver.MySchedule",
		"QueryResolver.MySettlement",
		"QueryResolver.MySettlements",
		"QueryResolver.MyShiftSwaps",
		"QueryResolver.MyTotalCompensation",
		"QueryResolver.MyTraining",
		"QueryResolver.MyYtdPay",
	},
	"capture devices are listed and revoked as the caller's own: the service is " +
		"asked with Mine set and refuses a device another user registered": {
		"MutationResolver.RevokeMyCaptureDevice",
		"QueryResolver.MyCaptureDevices",
	},
	"reports what capture the calling user may use, from their own session": {
		"QueryResolver.MyCaptureAccess",
	},
}

var roots = func() map[string]struct{} {
	out := make(map[string]struct{}, 96)
	for _, keys := range Resolvers {
		for _, key := range keys {
			out[strings.ToLower(key)] = struct{}{}
		}
	}
	return out
}()

// Root reports whether a root field (Query.myLoads, Mutation.cancelMyPTO) is
// one of the self-scoped resolvers. gqlgen names a resolver for its field with
// the initialisms it knows, so the two are compared without case.
func Root(object, field string) bool {
	_, ok := roots[strings.ToLower(object+"Resolver."+field)]
	return ok
}
