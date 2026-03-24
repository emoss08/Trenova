package webhooks

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Version = samsaraspec.WebhooksPostWebhooksRequestBodyVersion

type EventType = samsaraspec.WebhooksPostWebhooksRequestBodyEventTypes

const (
	Version20180101 Version = "2018-01-01"
	Version20210609 Version = "2021-06-09"
	Version20220913 Version = "2022-09-13"
	Version20240227 Version = "2024-02-27"
)

type CustomHeader = samsaraspec.CustomHeadersObjectRequestBody

type Webhook = samsaraspec.WebhooksGetWebhookResponseBody

type ListItem = samsaraspec.WebhookResponseResponseBody

type CreateRequest = samsaraspec.WebhooksPostWebhooksRequestBody

type UpdateRequest = samsaraspec.WebhooksPatchWebhookRequestBody

type ListResponse = samsaraspec.WebhooksListWebhooksResponseBody
