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

import React, { useState } from "react";
import CreatableSelect from "react-select/creatable";

const createOption = (label: string) => ({
  label,
  value: label.toLowerCase().replace(/\W/g, ""),
});

type Option = {
  label: string;
  value: string;
};
type Props = {
  options: Option[];
};

const MontaSelect = ({ options }: Props) => {
  const [currentOptions, setCurrentOptions] = useState<OptionsType<Option>>(options);

  const handleChange = (newValue: ValueType<Option>) => {
    if (newValue) {
      const newOption = newValue as Option;
      const newOptions = [...currentOptions, newOption];
      setCurrentOptions(newOptions);

      // You can also store the new option in a database for future use here
    }
  };

  return (
    <CreatableSelect
      options={currentOptions}
      onChange={handleChange}
      isSearchable
    />
  );
};

export default MontaSelect;
