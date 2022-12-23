import { useState } from 'react';

// material-ui
import { ToggleButton, ToggleButtonGroup } from '@mui/material';

// assets
import { BoldOutlined, ItalicOutlined, UnderlineOutlined, BgColorsOutlined, DownOutlined } from '@ant-design/icons';

// ==============================|| TOGGLE BUTTON - MULTIPLE ||============================== //

export default function MultipleToggleButtons() {
  const [formats, setFormats] = useState(() => ['bold', 'italic']);

  const handleFormat = (event: React.MouseEvent<HTMLElement>, newFormats: string[]) => {
    setFormats(newFormats);
  };

  return (
    <ToggleButtonGroup value={formats} onChange={handleFormat} aria-label="text formatting">
      <ToggleButton value="bold" aria-label="bold">
        <BoldOutlined />
      </ToggleButton>
      <ToggleButton value="italic" aria-label="italic">
        <ItalicOutlined />
      </ToggleButton>
      <ToggleButton value="underlined" aria-label="underlined">
        <UnderlineOutlined />
      </ToggleButton>
      <ToggleButton value="color" aria-label="color" disabled>
        <BgColorsOutlined />
        <DownOutlined style={{ fontSize: '0.625rem', marginLeft: 6 }} />
      </ToggleButton>
    </ToggleButtonGroup>
  );
}
