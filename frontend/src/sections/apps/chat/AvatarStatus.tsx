// material-ui
import { useTheme } from '@mui/material/styles';

// assets
import { CheckCircleFilled, ClockCircleFilled, MinusCircleFilled } from '@ant-design/icons';

// ==============================|| AVATAR STATUS ICONS ||============================== //

type Props = {
  status: string;
};

const AvatarStatus = ({ status }: Props) => {
  const theme = useTheme();

  switch (status) {
    case 'available':
      return <CheckCircleFilled style={{ color: theme.palette.success.main }} />;

    case 'do_not_disturb':
      return <MinusCircleFilled style={{ color: theme.palette.secondary.main }} />;

    case 'offline':
      return <ClockCircleFilled style={{ color: theme.palette.warning.main }} />;

    default:
      return null;
  }
};

export default AvatarStatus;
