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
import {
  faFilePdf,
  faFileCsv,
  faFileExcel,
} from "@fortawesome/free-solid-svg-icons";
import { useQuery, useQueryClient } from "react-query";
import { getUserReports } from "@/requests/UserRequestFactory";
import { getUserId } from "@/lib/utils";
import { Menu, Skeleton, Text } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faDownload } from "@fortawesome/pro-duotone-svg-icons";
import { headerStore } from "@/stores/HeaderStore";

export const UserReports: React.FC = () => {
  const userId = getUserId() || "";
  const queryClient = useQueryClient();
  const [downloadMenuOpen] = headerStore.use("downloadMenuOpen");

  // No stale time on this we want it to always be up-to-date
  const { data: userReportData, isLoading: isUserReportDataLoading } = useQuery(
    {
      queryKey: ["userReport", userId],
      queryFn: () => getUserReports(),
      initialData: () => {
        return queryClient.getQueryData(["userReport", userId]);
      },
      enabled: downloadMenuOpen,
    }
  );

  if (isUserReportDataLoading) {
    return <Skeleton width={200} height={250} />;
  }

  if (!userReportData || userReportData.length === 0) {
    return (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          alignItems: "center",
          height: "100%",
          width: "100%",
          marginTop: "30%",
        }}
      >
        <FontAwesomeIcon icon={faDownload} size="3x" />
        <Text>No downloads available</Text>
      </div>
    );
  }

  const menuItems = userReportData.map((item) => {
    let icon;

    if (item.file_name) {
      const fileExtension = (
        item.file_name.split(".").pop() || ""
      ).toLowerCase();

      switch (fileExtension) {
        case "pdf":
          icon = faFilePdf;
          break;
        case "csv":
          icon = faFileCsv;
          break;
        case "xls":
        case "xlsx":
          icon = faFileExcel;
          break;
        default:
          icon = faDownload; // default download icon if the file type is not PDF, CSV, or Excel
      }
    } else {
      icon = faDownload; // use the default download icon if `file_name` is not defined
    }

    return (
      <Menu.Item
        key={item.id}
        icon={<FontAwesomeIcon icon={icon} />}
        component="a"
        href={item.report}
      >
        {item.file_name}
      </Menu.Item>
    );
  });

  return <>{menuItems}</>; // return the list of Menu.Item components
};
