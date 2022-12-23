import React, { CSSProperties, useState } from 'react';

// material-ui
import { Theme, useTheme } from '@mui/material/styles';
import { Link, Menu, MenuItem, Stack, TableCell, TableRow, Typography } from '@mui/material';

// third-party
import { format } from 'date-fns';
import { Draggable } from 'react-beautiful-dnd';

// project imports
import AlertItemDelete from '../Board/AlertItemDelete';
import { openSnackbar } from 'store/reducers/snackbar';
import { useDispatch, useSelector } from 'store';
import { selectItem, deleteItem } from 'store/reducers/kanban';
import IconButton from 'components/@extended/IconButton';

// assets
import { MoreOutlined, ProfileOutlined } from '@ant-design/icons';

// types
import { KanbanItem } from 'types/kanban';

interface Props {
  itemId: string;
  index: number;
}

// drag wrapper
const getDragWrapper = (isDragging: boolean, theme: Theme): CSSProperties | undefined => {
  const bgcolor = theme.palette.mode === 'dark' ? theme.palette.background.paper + 90 : theme.palette.primary.lighter + 99;
  return {
    backgroundColor: isDragging ? bgcolor : 'transparent',
    userSelect: 'none'
  };
};

// ==============================|| KANBAN BACKLOGS - ITEMS ||============================== //

const Items = ({ itemId, index }: Props) => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const { columns, items, profiles, userStory } = useSelector((state) => state.kanban);

  const item = items.filter((data: KanbanItem) => data.id === itemId)[0];
  const itemColumn = columns.filter((column) => column.itemIds.filter((id) => id === item.id)[0])[0];
  const itemProfile = profiles.filter((profile) => profile.id === item.assign)[0];

  const handlerDetails = () => {
    dispatch(selectItem(itemId));
  };

  const [anchorEl, setAnchorEl] = useState<Element | ((element: Element) => Element) | null | undefined>(null);
  const handleClick = (event: React.MouseEvent<HTMLButtonElement> | undefined) => {
    setAnchorEl(event?.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const [open, setOpen] = useState(false);
  const handleModalClose = (status: boolean) => {
    setOpen(false);
    if (status) {
      dispatch(deleteItem(item.id, items, columns, userStory));
      dispatch(
        openSnackbar({
          open: true,
          message: 'Task Deleted successfully',
          anchorOrigin: { vertical: 'top', horizontal: 'right' },
          variant: 'alert',
          alert: {
            color: 'success'
          },
          close: false
        })
      );
    }
  };

  return (
    <Draggable draggableId={item.id} index={index}>
      {(provided, snapshot) => (
        <TableRow
          hover
          key={item.id}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          ref={provided.innerRef}
          sx={{
            '& th,& td': {
              whiteSpace: 'nowrap'
            },
            '& .more-button': {
              opacity: 0
            },
            ':hover': {
              '& .more-button': {
                opacity: 1
              }
            },
            ...(Boolean(anchorEl) && {
              '& .more-button': {
                opacity: 1
              }
            }),
            ...getDragWrapper(snapshot.isDragging, theme)
          }}
        >
          <TableCell sx={{ pl: 3, minWidth: 120, width: 120, height: 46 }} />
          <TableCell sx={{ width: 110, minWidth: 110 }}>
            <Stack direction="row" spacing={0.75} alignItems="center">
              <ProfileOutlined style={{ color: theme.palette.info.main, marginTop: -2 }} />
              <Typography variant="subtitle2">{item.id}</Typography>
            </Stack>
          </TableCell>
          <TableCell sx={{ maxWidth: 'calc(100vw - 850px)', minWidth: 140 }} component="th" scope="row">
            <Link
              underline="hover"
              color="default"
              onClick={handlerDetails}
              sx={{
                overflow: 'hidden',
                display: 'block',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                ':hover': { color: 'info.main' },
                cursor: 'pointer'
              }}
            >
              {item.title}
            </Link>
          </TableCell>
          <TableCell sx={{ width: 60, minWidth: 60 }}>
            <IconButton
              className="more-button"
              size="small"
              onClick={handleClick}
              aria-controls="menu-comment"
              aria-haspopup="true"
              color="secondary"
            >
              <MoreOutlined style={{ color: theme.palette.text.primary }} />
            </IconButton>
            <Menu
              id="menu-comment"
              anchorEl={anchorEl}
              keepMounted
              open={Boolean(anchorEl)}
              onClose={handleClose}
              variant="selectedMenu"
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'right'
              }}
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right'
              }}
            >
              <MenuItem
                onClick={() => {
                  handleClose();
                  handlerDetails();
                }}
              >
                Edit
              </MenuItem>
              <MenuItem
                onClick={() => {
                  handleClose();
                  setOpen(true);
                }}
              >
                Delete
              </MenuItem>
            </Menu>
            <AlertItemDelete title={item.title} open={open} handleClose={handleModalClose} />
          </TableCell>
          <TableCell sx={{ width: 90, minWidth: 90 }}>{itemColumn ? itemColumn.title : 'New'}</TableCell>
          <TableCell sx={{ width: 140, minWidth: 140 }}>{itemProfile ? itemProfile.name : ''}</TableCell>
          <TableCell sx={{ width: 85, minWidth: 85, textTransform: 'capitalize' }}>{item.priority}</TableCell>
          <TableCell sx={{ width: 120, minWidth: 120 }}>{item.dueDate ? format(new Date(item.dueDate), 'd MMM yyyy') : ''}</TableCell>
        </TableRow>
      )}
    </Draggable>
  );
};

export default Items;
