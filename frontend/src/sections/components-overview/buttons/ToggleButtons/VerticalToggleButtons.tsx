import { useState, MouseEvent } from 'react';

// material-ui
import { ToggleButton, ToggleButtonGroup } from '@mui/material';

// assets
import { ApartmentOutlined, AppstoreOutlined, TableOutlined } from '@ant-design/icons';

// ==============================|| TOGGLE BUTTON - VERTICAL ||============================== //

export default function VerticalToggleButtons() {
  const [view, setView] = useState('tree');

  const handleChange = (event: MouseEvent<HTMLElement>, nextView: string) => {
    setView(nextView);
  };

  return (
    <ToggleButtonGroup orientation="vertical" value={view} exclusive onChange={handleChange}>
      <ToggleButton value="tree" aria-label="tree">
        <ApartmentOutlined />
      </ToggleButton>
      <ToggleButton value="grid" aria-label="grid">
        <AppstoreOutlined />
      </ToggleButton>
      <ToggleButton value="table" aria-label="table">
        <TableOutlined />
      </ToggleButton>
    </ToggleButtonGroup>
  );
}
