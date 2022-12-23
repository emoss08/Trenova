import { useState } from 'react';

// material-ui
import { FormControl, FormHelperText, InputLabel, MenuItem, Select, SelectChangeEvent } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// ==============================|| COMPONENTS - HELPER TEXT ||============================== //

export default function HelperTextSelect() {
  const [age, setAge] = useState('');

  const handleChange = (event: SelectChangeEvent) => {
    setAge(event.target.value);
  };

  const helperSelectCodeString = `<FormControl fullWidth>
  <InputLabel id="demo-simple-select-helper-label">Number</InputLabel>
  <Select labelId="demo-simple-select-helper-label" id="demo-simple-select-helper" value={age} label="Age" onChange={handleChange}>
    <MenuItem value="">
      <em>Select Number</em>
    </MenuItem>
    <MenuItem value={10}>Ten</MenuItem>
    <MenuItem value={20}>Twenty</MenuItem>
    <MenuItem value={30}>Thirty</MenuItem>
  </Select>
  <FormHelperText>helper text</FormHelperText>
</FormControl>`;

  return (
    <MainCard title="With Helper Text" codeString={helperSelectCodeString}>
      <FormControl fullWidth>
        <InputLabel id="demo-simple-select-helper-label">Number</InputLabel>
        <Select labelId="demo-simple-select-helper-label" id="demo-simple-select-helper" value={age} label="Age" onChange={handleChange}>
          <MenuItem value="">
            <em>Select Number</em>
          </MenuItem>
          <MenuItem value={10}>Ten</MenuItem>
          <MenuItem value={20}>Twenty</MenuItem>
          <MenuItem value={30}>Thirty</MenuItem>
        </Select>
        <FormHelperText>helper text</FormHelperText>
      </FormControl>
    </MainCard>
  );
}
