/*
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

import React from "react";

import { Button } from "@/components/ui/button";

function ErrorPage() {
  return (
    <div className="space-y-3 p-10 text-center">
      <h1 className="text-2xl font-bold">Well, this is awkward....</h1>
      <p>You either don't have access to this page or it doesn't exist.</p>
      <p>We recommend contacting your administrator for more information.</p>
      <Button>Go back to the dashboard</Button>
    </div>
  );
}
export default ErrorPage;
