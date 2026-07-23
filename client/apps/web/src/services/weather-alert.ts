import { api } from "@/lib/api";
import type { WeatherAlertDetail, WeatherAlertFeatureCollection } from "@/types/weather-alert";

export class WeatherAlertService {
  public async getAlerts(): Promise<WeatherAlertFeatureCollection> {
    return api.get<WeatherAlertFeatureCollection>("/weather-alerts/");
  }

  public async getAlertDetail(alertId: string): Promise<WeatherAlertDetail> {
    return api.get<WeatherAlertDetail>(`/weather-alerts/${alertId}/`);
  }
}
