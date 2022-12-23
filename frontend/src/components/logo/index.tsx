import { Link } from 'react-router-dom';
import { To } from 'history';

// material-ui
import { ButtonBase } from '@mui/material';
import { SxProps } from '@mui/system';

// project import
import Logo from './LogoMain';
import LogoIcon from './LogoIcon';
import config from 'config';

// ==============================|| MAIN LOGO ||============================== //

interface Props {
  reverse?: boolean;
  isIcon?: boolean;
  sx?: SxProps;
  to?: To;
}

const LogoSection = ({ reverse, isIcon, sx, to }: Props) => (
  <ButtonBase disableRipple component={Link} to={!to ? config.defaultPath : to} sx={sx}>
    {isIcon ? <LogoIcon /> : <Logo reverse={reverse} />}
  </ButtonBase>
);

export default LogoSection;
