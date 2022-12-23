// material-ui
import { CSSObject, useTheme } from '@mui/material/styles';
import { Box } from '@mui/material';

// project import
import { ColorProps } from 'types/extended';
import getColors from 'utils/getColors';

interface Props {
  color?: ColorProps;
  size?: number;
  variant?: string;
  sx?: CSSObject;
}

const Dot = ({ color, size, variant, sx }: Props) => {
  const theme = useTheme();
  const colors = getColors(theme, color || 'primary');
  const { main } = colors;

  return (
    <Box
      component="span"
      sx={{
        width: size || 8,
        height: size || 8,
        borderRadius: '50%',
        bgcolor: variant === 'outlined' ? '' : main,
        ...(variant === 'outlined' && {
          border: `1px solid ${main}`
        }),
        ...sx
      }}
    />
  );
};

export default Dot;
