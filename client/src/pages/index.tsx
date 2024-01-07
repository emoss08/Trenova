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

import { Card, CardContent } from "@/components/ui/card";

export default function Index() {
  return (
    <Card>
      <CardContent>
        <p className="text-lg mb-2 border-b border-dashed font-semibold">
          Development in progress...
        </p>
        <p className="mt-1 text-sm text-gray-400">
          Monta is currently undergoing comprehensive development. As a result,
          certain features might not yet be accessible, or they may not perform
          to their intended specifications.
        </p>
      </CardContent>
    </Card>
  );
}
