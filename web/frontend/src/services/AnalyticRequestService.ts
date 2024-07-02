import axios from "@/lib/axiosConfig";

type DailyShipmentCount = {
  day: string;
  value: number;
};

type DailyShipmentCountResponse = {
  count: number;
  results: DailyShipmentCount[];
};

export async function getDailyShipmentCounts(
  startDate: string,
  endDate: string,
): Promise<DailyShipmentCountResponse> {
  const response = await axios.get("/analytics/daily-shipment-count/", {
    params: {
      start_date: startDate,
      end_date: endDate,
    },
  });
  return response.data;
}
