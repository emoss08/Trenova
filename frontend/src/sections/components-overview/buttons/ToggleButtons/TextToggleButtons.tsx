import { useState } from 'react';

// material-ui
import { ToggleButton, ToggleButtonGroup } from '@mui/material';

// ==============================|| TOGGLE BUTTON - TEXT ||============================== //

export default function TextToggleButtons() {
  const [alignment, setAlignment] = useState<string | null>('three');

  const handleAlignment = (event: React.MouseEvent<HTMLElement>, newAlignment: string | null) => {
    setAlignment(newAlignment);
  };

  return (
    <ToggleButtonGroup value={alignment} exclusive onChange={handleAlignment} aria-label="text alignment">
      <ToggleButton value="one" aria-label="first">
        One
      </ToggleButton>
      <ToggleButton value="two" aria-label="second">
        Two
      </ToggleButton>
      <ToggleButton value="three" aria-label="third">
        Three
      </ToggleButton>
      <ToggleButton value="four" aria-label="fourth">
        Four
      </ToggleButton>
    </ToggleButtonGroup>
  );
}
