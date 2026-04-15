import type { RainViewerData } from "@/types/shipment-map";

const RAINVIEWER_API = "https://api.rainviewer.com/public/weather-maps.json";

export class WeatherRadarService {
  public async getWeatherMaps(): Promise<RainViewerData> {
    const response = await fetch(RAINVIEWER_API);

    if (!response.ok) {
      throw new Error(`RainViewer API error: ${response.status}`);
    }

    return response.json() as Promise<RainViewerData>;
  }
}
