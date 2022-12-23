import { useState } from 'react';

// material-ui
import { Menu, MenuItem, Typography } from '@mui/material';

// project imports
import IconButton from 'components/@extended/IconButton';

// assets
import { BackwardOutlined, CopyOutlined, DeleteOutlined, ForwardOutlined, MoreOutlined } from '@ant-design/icons';

const ChatMessageAction = ({ index }: { index: number }) => {
  const [anchorEl, setAnchorEl] = useState<Element | ((element: Element) => Element) | null | undefined>(null);

  const handleClickSort = (event: React.MouseEvent<HTMLButtonElement> | undefined) => {
    setAnchorEl(event?.currentTarget);
  };

  const handleCloseSort = () => {
    setAnchorEl(null);
  };

  const open = Boolean(anchorEl);

  return (
    <>
      <IconButton
        id={`chat-action-button-${index}`}
        aria-controls={open ? `chat-action-menu-${index}` : undefined}
        aria-haspopup="true"
        aria-expanded={open ? 'true' : undefined}
        onClick={handleClickSort}
        size="small"
        color="secondary"
      >
        <MoreOutlined />
      </IconButton>
      <Menu
        id={`chat-action-menu-${index}`}
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleCloseSort}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'right'
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'right'
        }}
        MenuListProps={{
          'aria-labelledby': `chat-action-button-${index}`
        }}
        sx={{
          p: 0,
          '& .MuiMenu-list': {
            p: 0
          }
        }}
      >
        <MenuItem>
          <BackwardOutlined style={{ paddingRight: 8 }} />
          <Typography>Reply</Typography>
        </MenuItem>
        <MenuItem>
          <ForwardOutlined style={{ paddingRight: 8 }} />
          <Typography>Forward</Typography>
        </MenuItem>
        <MenuItem>
          <CopyOutlined style={{ paddingRight: 8 }} />
          <Typography>Copy</Typography>
        </MenuItem>
        <MenuItem>
          <DeleteOutlined style={{ paddingRight: 8, paddingLeft: 0 }} />
          <Typography>Delete</Typography>
        </MenuItem>
      </Menu>
    </>
  );
};

export default ChatMessageAction;
