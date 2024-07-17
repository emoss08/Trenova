/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { formatDateRelativeToNow } from "@/lib/date";
import { upperFirst } from "@/lib/utils";
import { User } from "@/types/accounts";
import { Location, LocationComment } from "@/types/location";
import { AvatarImage } from "@radix-ui/react-avatar";
import { Row } from "@tanstack/react-table";

// const TableBarChart = lazy(() => import("../chart/bar-chart"));

function classNames(...classes: (string | boolean)[]) {
  return classes.filter(Boolean).join(" ");
}

function UserAvatar({ user }: { user: User }) {
  // Determine the initials for the fallback avatar
  const initials = user.name
    .split(" ")
    .map((n) => n[0])
    .join("");

  // Determine the avatar image source
  const avatarSrc =
    user.profilePicUrl ?? `https://avatar.vercel.sh/${user.email}`;

  return (
    <Avatar className="relative mt-3 size-6 flex-none rounded-full">
      <AvatarImage
        src={avatarSrc}
        alt={user.username}
        className="size-6 rounded-full"
      />
      <AvatarFallback delayMs={600}>{initials}</AvatarFallback>
    </Avatar>
  );
}

export function CommentList({ comments }: { comments: LocationComment[] }) {
  return comments.length > 0 ? (
    <ul role="list" className="gap-y-4">
      {comments.map((comment, commentIdx) => (
        <li key={comment.id} className="relative mr-5 flex gap-4">
          <div
            className={classNames(
              commentIdx === comments.length - 1 ? "h-6" : "-bottom-6",
              "absolute left-0 top-0 flex w-6 justify-center",
            )}
          >
            <div className="w-px bg-border" />
          </div>
          <>
            <UserAvatar user={comment.edges?.user as User} />
            <div className="flex-auto rounded-md border border-border p-3">
              <div className="flex justify-between gap-x-4">
                <div className="py-0.5 text-xs leading-5 text-foreground">
                  <span className="font-medium text-accent-foreground">
                    {upperFirst(comment.edges?.user.name ?? "")}
                  </span>
                  {" posted a "}
                  <span className="font-medium">
                    {upperFirst(comment.edges?.commentType.name ?? "")}
                  </span>
                </div>
                <time
                  dateTime={comment.createdAt}
                  className="flex-none py-0.5 text-xs leading-5 text-muted-foreground"
                >
                  {formatDateRelativeToNow(comment.createdAt)}
                </time>
              </div>
              <p className="text-sm leading-6 text-muted-foreground">
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

export default function LocationChart({ row }: { row: Row<Location> }) {
  // const queryClient = useQueryClient();

  // const { data, isError } = useQuery({
  //   queryKey: ["locationPickupData", row.original.id],
  //   queryFn: async () => getLocationPickupData(row.original.id),
  //   enabled: row.original.id !== undefined,
  //   initialData: queryClient.getQueryData([
  //     "locationPickupData",
  //     row.original.id,
  //   ]),
  //   retry: false,
  //   staleTime: Infinity,
  //   refetchOnWindowFocus: false,
  // });

  // if (isError) {
  //   return (
  //     <div className="m-4">
  //       <ErrorLoadingData message="Failed to loading the proper information to display the chart." />
  //     </div>
  //   );
  // }

  return (
    <div className="mt-7 flex border-b">
      {/* <Suspense fallback={<ComponentLoader />}>
        {data && <TableBarChart data={data} />}
      </Suspense> */}
      <div className="flex-1 space-x-4">
        <h2 className="scroll-m-20 pl-5 text-2xl font-semibold tracking-tight first:mt-0">
          Recent Comments
        </h2>
        <ScrollArea className="h-82 overflow-auto">
          <CommentList comments={row.original.edges?.comments || []} />
        </ScrollArea>
      </div>
    </div>
  );
}
