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

import React, {useRef, useEffect} from 'react'
import { useLayout } from "@/utils/layout/LayoutProvider";

const BG_COLORS = ['bg-white', 'bg-info']

export function Sidebar() {
  const {classes} = useLayout()
  const sidebarCSSClass = classes.sidebar
  const sideBarRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!sidebarCSSClass) {
      return
    }

    BG_COLORS.forEach((cssClass) => {
      sideBarRef.current?.classList.remove(cssClass)
    })

    sidebarCSSClass.forEach((cssClass) => {
      sideBarRef.current?.classList.add(cssClass)
    })
  }, [sidebarCSSClass])

  return <>Sidebar</>
}
