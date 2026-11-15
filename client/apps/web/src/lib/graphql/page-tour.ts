import type {
  PageTourViewFieldsFragment,
  RecordPageTourInput,
} from "@trenova/graphql/generated/graphql";
import {
  MyPageTourViewsDocument,
  PageTourViewFieldsFragmentDoc,
  RecordMyPageTourDocument,
} from "@trenova/graphql/generated/documents/page-tour/page-tour";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type PageTourView = PageTourViewFieldsFragment;
export type PageTourOutcome = PageTourView["outcome"];

/** The page tours the signed-in person has already been shown. */
export async function fetchMyPageTourViews(options?: {
  signal?: AbortSignal;
}): Promise<PageTourView[]> {
  const data = await requestGraphQL({
    document: MyPageTourViewsDocument,
    operationName: "MyPageTourViews",
    signal: options?.signal,
  });

  return data.myPageTourViews.map((view) => getFragmentData(PageTourViewFieldsFragmentDoc, view));
}

/** Remembers that the signed-in person has been shown a tour, so it does not open again. */
export async function recordMyPageTour(input: RecordPageTourInput): Promise<PageTourView> {
  const data = await requestGraphQL({
    document: RecordMyPageTourDocument,
    operationName: "RecordMyPageTour",
    variables: { input },
  });

  return getFragmentData(PageTourViewFieldsFragmentDoc, data.recordMyPageTour);
}
