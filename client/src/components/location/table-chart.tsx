/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React from "react";
import { Row } from "@tanstack/react-table";
import {
  Bar,
  BarChart,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Location } from "@/types/location";
import { truncateText, upperFirst } from "@/lib/utils";
import { formatDateToHumanReadable } from "@/lib/date";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getLocationPickupData } from "@/services/LocationRequestService";
import { Loader2 } from "lucide-react";

function SkeletonLoader() {
  return (
    <div className="flex flex-col items-center justify-center mt-20">
      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      <p className="mt-2 font-semibold text-accent-foreground">
        Loading Chart...
      </p>
      <p className="mt-2 text-muted-foreground">
        If this takes longer than 10 seconds, please refresh the page.
      </p>
    </div>
  );
}

export function LocationChart({ row }: { row: Row<Location> }) {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["locationPickupData", row.original.id],
    queryFn: async () => getLocationPickupData(row.original.id),
    enabled: row.original.id !== undefined,
    initialData: queryClient.getQueryData([
      "locationPickupData",
      row.original.id,
    ]),
    retry: false,
    refetchOnWindowFocus: false,
  });

  return (
    <div className="mt-7 flex border-b">
      <div className="flex-1 col-xs-push-3">
        <h2 className="scroll-m-20 pb-2 pl-5 text-2xl font-semibold tracking-tight first:mt-0">
          Monthly Pickups
        </h2>
        {isLoading ? (
          <SkeletonLoader />
        ) : (
          <ResponsiveContainer width="100%" height={350} className="mt-5">
            <BarChart data={data}>
              <XAxis
                dataKey="name"
                stroke="#888888"
                fontSize={12}
                tickLine={false}
                axisLine={false}
              />
              <YAxis
                stroke="#888888"
                fontSize={12}
                tickLine={false}
                axisLine={false}
              />
              <Tooltip />
              <Legend />
              <Bar dataKey="total" fill="#ad1dfa" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>

      {/* Container for Recent Comments */}
      <div className="flex-1">
        <h2 className="scroll-m-20 pb-2 pl-5 text-2xl font-semibold tracking-tight first:mt-0">
          Recent Comments
        </h2>
        {row.original.locationComments.length > 0 ? (
          row.original.locationComments
            .sort(
              (a, b) =>
                new Date(b.created).getTime() - new Date(a.created).getTime(),
            )
            .slice(0, 3)
            .map((comment) => (
              <div key={comment.id} className="flex flex-col overflow-hidden">
                <div className="px-6 py-4">
                  <h4 className="text-xl font-semibold text-gray-800 dark:text-white">
                    {comment.commentTypeName}
                  </h4>
                  <p className="mt-1">"{truncateText(comment.comment, 150)}"</p>
                </div>
                <div className="flex items-center px-6">
                  <p className="ml-2 text-sm text-gray-400">
                    by&nbsp;{upperFirst(comment.enteredByUsername)}&nbsp;
                    {formatDateToHumanReadable(comment.created)}
                  </p>
                </div>
              </div>
            ))
        ) : (
          <div className="flex flex-col text-center rounded-lg overflow-hidden my-4">
            <div className="px-6 py-4">
              <h4 className="text-xl font-semibold">
                No Location Comments Available
              </h4>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
