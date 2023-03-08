/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import Head from "next/head";
import Image from "next/image";
import { Inter } from "next/font/google";
import styles from "@/styles/Home.module.css";
import { authStore } from "@/utils/providers/AuthGuard";
import { ThemeModeSwitcher } from "@/components/layout/ThemeModeSwitcher";
import { MasterLayout } from "@/utils/MasterLayout";

const inter = Inter({ subsets: ["latin"] });

export default function Home() {
  const [user] = authStore.use("user");

  return (
    <>
      <MasterLayout>
      <div>
        <h1>Home</h1>
      </div>
      </MasterLayout>
    </>
  );
}
