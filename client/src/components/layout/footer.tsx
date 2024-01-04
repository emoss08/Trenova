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

export function Footer() {
  return (
    <footer className="mt-5 flex items-center justify-between px-10 py-2 font-semibold">
      {/* Copyright */}
      <div className="text-xs text-foreground">
        <span className="me-1">2023©</span>
        <a href="#" target="_blank" rel="noopener noreferrer">
          Monta Technologies
        </a>
      </div>

      {/* Menu */}
      <ul className="flex space-x-4 text-xs text-foreground">
        <li>
          <a href="#" target="_blank" rel="noopener noreferrer">
            Terms & Conditions
          </a>
        </li>
        <li>
          <a href="#" target="_blank" rel="noopener noreferrer">
            Support
          </a>
        </li>
        <li>
          <a href="#" target="_blank" rel="noopener noreferrer">
            License
          </a>
        </li>
      </ul>
    </footer>
  );
}
