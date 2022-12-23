// material-ui
import { Autocomplete, TextField } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// ==============================|| AUTOCOMPLETE - DISABLED ||============================== //

export default function DisabledAutocomplete() {
  const disabledAutocompleteCodeString = `<Autocomplete
  fullWidth
  id="disabled-options-demo"
  options={timeSlots}
  getOptionDisabled={(option) => option === timeSlots[0] || option === timeSlots[2]}
  renderInput={(params) => <TextField {...params} label="Disabled options" />}
/>`;

  return (
    <MainCard title="Disabled Options" codeString={disabledAutocompleteCodeString}>
      <Autocomplete
        fullWidth
        id="disabled-options-demo"
        options={timeSlots}
        getOptionDisabled={(option) => option === timeSlots[0] || option === timeSlots[2]}
        renderInput={(params) => <TextField {...params} label="Disabled options" />}
      />
    </MainCard>
  );
}

// One time slot every 30 minutes.
const timeSlots = Array.from(new Array(24 * 2)).map(
  (_, index) => `${index < 20 ? '0' : ''}${Math.floor(index / 2)}:${index % 2 === 0 ? '00' : '30'}`
);
