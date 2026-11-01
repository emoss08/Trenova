import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  WeatherAlertDocument,
  WeatherAlertFeatureFieldsFragmentDoc,
  WeatherAlertsDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  WeatherAlertDetail,
  WeatherAlertFeature,
  WeatherAlertFeatureCollection,
} from "@/types/weather-alert";

export class WeatherAlertService {
  public async getAlerts(options?: {
    signal?: AbortSignal;
  }): Promise<WeatherAlertFeatureCollection> {
    const response = await requestGraphQL({
      document: WeatherAlertsDocument,
      operationName: "WeatherAlerts",
      signal: options?.signal,
    });

    return {
      type: "FeatureCollection",
      features: response.weatherAlerts.features.map(
        (feature) =>
          getFragmentData(WeatherAlertFeatureFieldsFragmentDoc, feature) as WeatherAlertFeature,
      ),
    };
  }

  public async getAlertDetail(
    alertId: string,
    options?: { signal?: AbortSignal },
  ): Promise<WeatherAlertDetail> {
    const response = await requestGraphQL({
      document: WeatherAlertDocument,
      operationName: "WeatherAlert",
      variables: { id: alertId },
      signal: options?.signal,
    });
    const detail = response.weatherAlert;

    return {
      feature: getFragmentData(
        WeatherAlertFeatureFieldsFragmentDoc,
        detail.feature,
      ) as WeatherAlertFeature,
      activities: detail.activities.map((activity) => ({
        ...activity,
        details: (activity.details as Record<string, unknown> | null) ?? null,
      })),
    };
  }
}
