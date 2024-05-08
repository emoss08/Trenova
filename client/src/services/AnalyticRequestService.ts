import axios from "@/lib/axiosConfig";

export async function getDailyShipmentCounts(
  startDate: string,
  endDate: string,
): Promise<{ count: number }> {
  const response = await axios.get("/analytics/daily-shipment-count/", {
    params: {
      start_date: startDate,
      end_date: endDate,
    },
  });
  return response.data;
}
