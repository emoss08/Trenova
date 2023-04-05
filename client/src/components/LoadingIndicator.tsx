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

import React, { useEffect } from "react";
import topbar from "topbar";

interface LoadingIndicatorProps {
  loading: boolean;
}

const LoadingIndicator: React.FC<LoadingIndicatorProps> = ({ loading }) => {
  topbar.config({
    barThickness: 3,
    barColors: {
      "0": "rgba(26,  188, 156, .9)",
      ".25": "rgba(52,  152, 219, .9)",
      ".50": "rgba(241, 196, 15,  .9)",
      ".75": "rgba(230, 126, 34,  .9)",
      "1.0": "rgba(211, 84,  0,   .9)"
    },
    shadowBlur: 10,
    shadowColor: "rgba(0,   0,   0,   .6)",
    className: "topbar"
  });

  useEffect(() => {
    if (loading) {
      topbar.show();
    } else {
      topbar.hide();
    }
  }, [loading]);

  return null;
};
export default LoadingIndicator;