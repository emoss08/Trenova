// material-ui
import { Box, FormControlLabel, Radio, Tooltip } from '@mui/material';

// project import
import Avatar from 'components/@extended/Avatar';

// assets
import { CheckOutlined } from '@ant-design/icons';

// ==============================|| CALENDAR - COLOR PALETTE ||============================== //

interface Props {
  color: string;
  value: string;
}

const ColorPalette = ({ color, value }: Props) => (
  <Tooltip title={color}>
    <FormControlLabel
      value={value}
      label=""
      control={
        <Radio
          disableRipple
          icon={
            <Avatar variant="rounded" type="combined" size="xs" sx={{ backgroundColor: color, borderColor: 'divider' }}>
              <Box sx={{ display: 'none' }} />
            </Avatar>
          }
          checkedIcon={
            <Avatar variant="rounded" type="combined" size="xs" sx={{ backgroundColor: color, color: '#000', borderColor: 'divider' }}>
              <CheckOutlined />
            </Avatar>
          }
          sx={{
            '&:hover': {
              bgcolor: 'transparent'
            }
          }}
        />
      }
    />
  </Tooltip>
);

export default ColorPalette;
