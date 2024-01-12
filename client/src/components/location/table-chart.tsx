/*
 * COPYRIGHT(c) 2024 MONTA
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
import { Location, LocationComment } from "@/types/location";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getLocationPickupData } from "@/services/LocationRequestService";
import { Loader2 } from "lucide-react";
import { formatDateToHumanReadable } from "@/lib/date";
import { upperFirst } from "@/lib/utils";
import { MinimalUser } from "@/types/accounts";
import { AvatarImage } from "@radix-ui/react-avatar";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";

function SkeletonLoader() {
  return (
    <div className="mt-20 flex flex-col items-center justify-center">
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

function classNames(...classes: (string | boolean)[]) {
  return classes.filter(Boolean).join(" ");
}

function UserAvatar({ user }: { user: MinimalUser }) {
  // Determine the initials for the fallback avatar
  const initials = user.profile
    ? user.profile.firstName.charAt(0) + user.profile.lastName.charAt(0)
    : "";

  // Determine the avatar image source
  const avatarSrc = user.profile?.thumbnail
    ? user.profile.thumbnail
    : `https://avatar.vercel.sh/${user.email}`;

  return (
    <Avatar className="relative mt-3 h-6 w-6 flex-none rounded-full">
      <AvatarImage
        src={avatarSrc}
        alt={user.username}
        className="h-6 w-6 rounded-full"
      />
      <AvatarFallback delayMs={600}>{initials}</AvatarFallback>
    </Avatar>
  );
}

export function CommentList({ comments }: { comments: LocationComment[] }) {
  const userFullName = (comment: LocationComment) => {
    return `${comment.enteredBy.profile?.firstName} ${comment.enteredBy.profile?.lastName}`;
  };

  return comments.length > 0 ? (
    <ul role="list" className="space-y-6">
      {comments.map((comment, commentIdx) => (
        <li key={comment.id} className="relative mr-5 flex gap-x-4">
          <div
            className={classNames(
              commentIdx === comments.length - 1 ? "h-6" : "-bottom-6",
              "absolute left-0 top-0 flex w-6 justify-center",
            )}
          >
            <div className="w-px bg-gray-200" />
          </div>
          <>
            <UserAvatar user={comment.enteredBy} />
            <div className="flex-auto rounded-md p-3 ring-1 ring-inset ring-gray-200">
              <div className="flex justify-between gap-x-4">
                <div className="py-0.5 text-xs leading-5 text-gray-500">
                  <span className="font-medium text-gray-900">
                    {upperFirst(userFullName(comment))}
                  </span>
                  {" posted a "}
                  <span className="font-medium">
                    {upperFirst(comment.commentTypeName)}
                  </span>
                </div>
                <time
                  dateTime={comment.created}
                  className="flex-none py-0.5 text-xs leading-5 text-gray-500"
                >
                  {formatDateToHumanReadable(comment.created)}
                </time>
              </div>
              <p className="text-sm leading-6 text-gray-500">
                {comment.comment}
              </p>
            </div>
          </>
        </li>
      ))}
    </ul>
  ) : (
    <div className="my-4 flex flex-col items-center justify-center overflow-hidden rounded-lg">
      <div className="px-6 py-4">
        <h4 className="mt-20 text-xl font-semibold text-foreground">
          No Location Comments Available
        </h4>
      </div>
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
      <div className="col-xs-push-3 flex-1">
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
        <h2 className="scroll-m-20 pl-5 text-2xl font-semibold tracking-tight first:mt-0">
          Recent Comments
        </h2>
        <ScrollArea className="h-82 overflow-auto">
          <CommentList comments={row.original.locationComments} />
        </ScrollArea>
      </div>
    </div>
  );
}
