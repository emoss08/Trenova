import { useState } from 'react';
import { ToggleButton, ToggleButtonGroup } from '@mui/material';

// assets
import { AlignLeftOutlined, AlignCenterOutlined, AlignRightOutlined, UnorderedListOutlined } from '@ant-design/icons';

// ==============================|| TOGGLE BUTTON - COLOR ||============================== //

export default function ColorToggleButton() {
  const [alignment, setAlignment] = useState<string | null>('left');

  const handleAlignment = (event: React.MouseEvent<HTMLElement>, newAlignment: string | null) => {
    setAlignment(newAlignment);
  };

  return (
    <ToggleButtonGroup color="primary" value={alignment} exclusive onChange={handleAlignment} aria-label="text alignment">
      <ToggleButton value="left" aria-label="left aligned">
        <AlignLeftOutlined />
      </ToggleButton>
      <ToggleButton value="center" aria-label="centered">
        <AlignCenterOutlined />
      </ToggleButton>
      <ToggleButton value="right" aria-label="right aligned">
        <AlignRightOutlined />
      </ToggleButton>
      <ToggleButton value="list" aria-label="list">
        <UnorderedListOutlined />
      </ToggleButton>
    </ToggleButtonGroup>
  );
}
