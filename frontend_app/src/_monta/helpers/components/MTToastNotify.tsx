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

import {toast, cssTransition, Theme} from 'react-toastify'

export const MTToastNotify: ({
  message,
  theme,
  icon,
  autoClose,
}: {
  message: string
  theme: Theme
  icon?: string
  autoClose?: number
}) => void = ({message, theme, icon, autoClose}) => {
  const Bounce = cssTransition({
    enter: 'animate__animated animate__bounceIn',
    exit: 'animate__animated animate__bounceOut',
  })

  toast(message, {
    theme: theme,
    transition: Bounce,
    pauseOnFocusLoss: false,
    pauseOnHover: true,
    icon: icon ? icon : undefined,
    autoClose: autoClose ? autoClose : 5000,
  })
}
