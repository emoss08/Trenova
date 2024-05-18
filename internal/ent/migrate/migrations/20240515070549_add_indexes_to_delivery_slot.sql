-- Create index "deliveryslot_customer_id_location_id_day_of_week_start_time_end" to table: "delivery_slots"
CREATE UNIQUE INDEX "deliveryslot_customer_id_location_id_day_of_week_start_time_end" ON "delivery_slots" ("customer_id", "location_id", "day_of_week", "start_time", "end_time");
