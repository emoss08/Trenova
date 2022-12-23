import { ReactNode } from 'react';

// material-ui
import { styled, useTheme, Theme } from '@mui/material/styles';
import MuiAvatar from '@mui/material/Avatar';
import { AvatarProps } from '@mui/material';

// project import
import getColors from 'utils/getColors';

// types
import { AvatarTypeProps, ColorProps, ExtendedStyleProps, SizeProps } from 'types/extended';

// ==============================|| AVATAR - COLOR STYLE ||============================== //

interface AvatarStyleProps extends ExtendedStyleProps {
  variant?: AvatarProps['variant'];
  type?: AvatarTypeProps;
}

function getColorStyle({ variant, theme, color, type }: AvatarStyleProps) {
  const colors = getColors(theme, color);
  const { lighter, light, main, contrastText } = colors;

  switch (type) {
    case 'filled':
      return {
        color: contrastText,
        backgroundColor: main
      };
    case 'outlined':
      return {
        color: main,
        border: '1px solid',
        borderColor: main,
        backgroundColor: 'transparent'
      };
    case 'combined':
      return {
        color: main,
        border: '1px solid',
        borderColor: light,
        backgroundColor: lighter
      };
    default:
      return {
        color: main,
        backgroundColor: lighter
      };
  }
}

// ==============================|| AVATAR - SIZE STYLE ||============================== //

function getSizeStyle(size?: SizeProps) {
  switch (size) {
    case 'badge':
      return {
        border: '2px solid',
        fontSize: '0.675rem',
        width: 20,
        height: 20
      };
    case 'xs':
      return {
        fontSize: '0.75rem',
        width: 24,
        height: 24
      };
    case 'sm':
      return {
        fontSize: '0.875rem',
        width: 32,
        height: 32
      };
    case 'lg':
      return {
        fontSize: '1.2rem',
        width: 52,
        height: 52
      };
    case 'xl':
      return {
        fontSize: '1.5rem',
        width: 64,
        height: 64
      };
    case 'md':
    default:
      return {
        fontSize: '1rem',
        width: 40,
        height: 40
      };
  }
}

// ==============================|| STYLED - AVATAR ||============================== //

interface StyleProps {
  color: ColorProps;
  variant?: AvatarProps['variant'];
  type?: AvatarTypeProps;
  theme: Theme;
  size?: SizeProps;
}

const AvatarStyle = styled(MuiAvatar, { shouldForwardProp: (prop) => prop !== 'color' && prop !== 'type' && prop !== 'size' })(
  ({ theme, variant, color, type, size }: StyleProps) => ({
    ...getSizeStyle(size),
    ...getColorStyle({ variant, theme, color, type }),
    ...(size === 'badge' && {
      borderColor: theme.palette.background.default
    })
  })
);

// ==============================|| EXTENDED - AVATAR ||============================== //

export interface Props extends AvatarProps {
  color?: ColorProps;
  children?: ReactNode | string;
  type?: AvatarTypeProps;
  size?: SizeProps;
}

export default function Avatar({ variant = 'circular', children, color = 'primary', type, size = 'md', ...others }: Props) {
  const theme = useTheme();

  return (
    <AvatarStyle variant={variant} theme={theme} color={color} type={type} size={size} {...others}>
      {children}
    </AvatarStyle>
  );
}
