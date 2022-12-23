import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Avatar, Button, Dialog, DialogTitle, Grid, List, ListItem, ListItemAvatar, ListItemText } from '@mui/material';

// project import
import IconButton from 'components/@extended/IconButton';

// assets
import { CloseOutlined, PlusOutlined } from '@ant-design/icons';

const emails = ['username@gmail.com', 'user02@gmail.com'];
const avatarImage = require.context('assets/images/users', true);

// ==============================|| DIALOG - SIMPLE ||============================== //

export interface Props {
  open: boolean;
  selectedValue: string;
  onClose: (value: string) => void;
}

function SimpleDialog({ onClose, selectedValue, open }: Props) {
  const theme = useTheme();

  const handleClose = () => {
    onClose(selectedValue);
  };

  const handleListItemClick = (value: string) => {
    onClose(value);
  };

  return (
    <Dialog onClose={handleClose} open={open}>
      <Grid
        container
        spacing={2}
        justifyContent="space-between"
        alignItems="center"
        sx={{ borderBottom: `1px solid ${theme.palette.divider}` }}
      >
        <Grid item>
          <DialogTitle>Set backup account</DialogTitle>
        </Grid>
        <Grid item sx={{ mr: 1.5 }}>
          <IconButton color="secondary" onClick={handleClose}>
            <CloseOutlined />
          </IconButton>
        </Grid>
      </Grid>

      <List sx={{ p: 2.5 }}>
        {emails.map((email, index) => (
          <ListItem button onClick={() => handleListItemClick(email)} key={email} selected={selectedValue === email} sx={{ p: 1.25 }}>
            <ListItemAvatar>
              <Avatar src={avatarImage(`./avatar-${index + 1}.png`)} />
            </ListItemAvatar>
            <ListItemText primary={email} />
          </ListItem>
        ))}
        <ListItem autoFocus button onClick={() => handleListItemClick('addAccount')} sx={{ p: 1.25 }}>
          <ListItemAvatar>
            <Avatar sx={{ bgcolor: 'primary.lighter', color: 'primary.main', width: 32, height: 32 }}>
              <PlusOutlined style={{ fontSize: '0.625rem' }} />
            </Avatar>
          </ListItemAvatar>
          <ListItemText primary="Add Account" />
        </ListItem>
      </List>
    </Dialog>
  );
}

export default function SimpleDialogDemo() {
  const [open, setOpen] = useState(false);
  const [selectedValue, setSelectedValue] = useState(emails[1]);

  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = (value: string) => {
    setOpen(false);
    setSelectedValue(value);
  };

  return (
    <>
      <Button variant="contained" onClick={handleClickOpen}>
        Open simple dialog
      </Button>
      <SimpleDialog selectedValue={selectedValue} open={open} onClose={handleClose} />
    </>
  );
}
