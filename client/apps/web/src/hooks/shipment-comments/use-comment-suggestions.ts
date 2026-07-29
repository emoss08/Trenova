import type {
  SuggestionFetcher,
  SuggestionPage,
} from "@trenova/shared/components/comment-editor/comment-editor";
import { fetchOptions } from "@/components/fields/autocomplete/autocomplete-content";
import {
  fetchGraphQLSelectOptions,
  type GraphQLSelectOptionsConfig,
  type SelectOption,
} from "@/lib/graphql/select-options";

const MENTION_PAGE_SIZE = 10;
const ENTITY_REF_GROUP_PAGE_SIZE = 5;

const shipmentSelectOptionsGraphQL = {
  resource: "SHIPMENT",
} satisfies GraphQLSelectOptionsConfig;

const workerSelectOptionsGraphQL = {
  resource: "WORKER",
} satisfies GraphQLSelectOptionsConfig;

const customerSelectOptionsGraphQL = {
  resource: "CUSTOMER",
} satisfies GraphQLSelectOptionsConfig;

interface UserSelectOption {
  id: string;
  name?: string;
  emailAddress?: string;
}

function selectOptionMetaString(option: SelectOption, key: string): string {
  const value = option.meta?.[key];
  return typeof value === "string" ? value : "";
}

export const fetchMentionCandidates: SuggestionFetcher = async (query, page) => {
  const response = await fetchOptions<UserSelectOption>(
    "/users/select-options/",
    query,
    page,
    MENTION_PAGE_SIZE,
  );

  return {
    items: (response.results ?? []).map((user) => ({
      id: user.id,
      label: user.name || user.emailAddress || user.id,
      description: user.name ? (user.emailAddress ?? null) : null,
    })),
    hasMore: response.next != null,
  };
};

export const fetchAllEntityRefCandidates: SuggestionFetcher = async (query, page) => {
  const [shipments, workers, customers] = await Promise.all([
    fetchGraphQLSelectOptions({
      ...shipmentSelectOptionsGraphQL,
      query,
      page,
      initialLimit: ENTITY_REF_GROUP_PAGE_SIZE,
    }).catch(() => null),
    fetchGraphQLSelectOptions({
      ...workerSelectOptionsGraphQL,
      query,
      page,
      initialLimit: ENTITY_REF_GROUP_PAGE_SIZE,
    }).catch(() => null),
    fetchGraphQLSelectOptions({
      ...customerSelectOptionsGraphQL,
      query,
      page,
      initialLimit: ENTITY_REF_GROUP_PAGE_SIZE,
    }).catch(() => null),
  ]);

  const result: SuggestionPage = { items: [], hasMore: false };

  for (const option of shipments?.results ?? []) {
    result.items.push({
      id: option.id,
      label: option.label,
      description:
        selectOptionMetaString(option, "bol") || selectOptionMetaString(option, "status") || null,
      kind: "shipment",
    });
  }
  for (const option of workers?.results ?? []) {
    const fleetCode = selectOptionMetaString(option, "fleetCode");
    result.items.push({
      id: option.id,
      label: option.label,
      description: fleetCode ? `Fleet: ${fleetCode}` : null,
      kind: "worker",
    });
  }
  for (const option of customers?.results ?? []) {
    const code = selectOptionMetaString(option, "code");
    result.items.push({
      id: option.id,
      label: code ? `${code} - ${option.label}` : option.label,
      description: null,
      kind: "customer",
    });
  }

  result.hasMore = shipments?.next != null || workers?.next != null || customers?.next != null;

  return result;
};
