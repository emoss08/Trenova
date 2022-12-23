import { CSSProperties, useState } from 'react';

// material-ui
import { useTheme, Theme } from '@mui/material/styles';
import { CardMedia, Link, Menu, MenuItem, Stack, Tooltip, Typography } from '@mui/material';

// third-party
import { Draggable, DraggingStyle, NotDraggingStyle } from 'react-beautiful-dnd';

// project imports
import EditStory from '../Backlogs/EditStory';
import AlertItemDelete from './AlertItemDelete';
import { openSnackbar } from 'store/reducers/snackbar';
import { useDispatch, useSelector } from 'store';
import { selectItem, deleteItem } from 'store/reducers/kanban';
import IconButton from 'components/@extended/IconButton';

// assets
import { ClusterOutlined, MoreOutlined } from '@ant-design/icons';

// types
import { KanbanItem } from 'types/kanban';

interface Props {
  item: KanbanItem;
  index: number;
}

const backImage = require.context('assets/images/profile', true);

// item drag wrapper
const getDragWrapper = (
  isDragging: boolean,
  draggableStyle: DraggingStyle | NotDraggingStyle | undefined,
  theme: Theme,
  radius: string
): CSSProperties | undefined => {
  const bgcolor = theme.palette.background.paper + 99;
  return {
    userSelect: 'none',
    margin: `0 0 ${8}px 0`,
    padding: 16,
    border: '1px solid',
    borderColor: theme.palette.divider,
    backgroundColor: isDragging ? bgcolor : theme.palette.background.paper,
    borderRadius: radius,
    ...draggableStyle
  };
};

// ==============================|| KANBAN BOARD - ITEMS ||============================== //

const Items = ({ item, index }: Props) => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const backProfile = item.image && backImage(`./${item.image}`);

  const kanban = useSelector((state) => state.kanban);
  const { userStory, items, columns } = kanban;

  const itemStory = userStory.filter((story) => story?.itemIds?.filter((itemId) => itemId === item.id)[0])[0];

  const handlerDetails = (id: string) => {
    dispatch(selectItem(id));
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

  const [openStoryDrawer, setOpenStoryDrawer] = useState<boolean>(false);
  const handleStoryDrawerOpen = () => {
    setOpenStoryDrawer((prevState) => !prevState);
  };

  const editStory = () => {
    setOpenStoryDrawer((prevState) => !prevState);
  };

  return (
    <Draggable key={item.id} draggableId={item.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          style={getDragWrapper(snapshot.isDragging, provided.draggableProps.style, theme, `4px`)}
        >
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: itemStory ? -0.75 : 0 }}>
            <Typography
              onClick={() => handlerDetails(item.id)}
              variant="subtitle1"
              sx={{
                display: 'inline-block',
                width: 'calc(100% - 34px)',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                verticalAlign: 'middle',
                cursor: 'pointer',
                '&:hover': {
                  textDecoration: 'underline'
                }
              }}
            >
              {item.title}
            </Typography>

            <IconButton size="small" color="secondary" onClick={handleClick} aria-controls="menu-comment" aria-haspopup="true">
              <MoreOutlined />
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
                  handlerDetails(item.id);
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
            {open && <AlertItemDelete title={item.title} open={open} handleClose={handleModalClose} />}
          </Stack>
          {itemStory && (
            <>
              <Stack direction="row" spacing={0.5} alignItems="center">
                <Tooltip title="User Story">
                  <ClusterOutlined style={{ color: theme.palette.primary.dark, fontSize: '0.75rem' }} />
                </Tooltip>
                <Tooltip title={itemStory.title}>
                  <Link variant="caption" color="primary.dark" underline="hover" onClick={editStory} sx={{ cursor: 'pointer', pt: 0.5 }}>
                    User Story #{itemStory.id}
                  </Link>
                </Tooltip>
              </Stack>
              {openStoryDrawer && <EditStory story={itemStory} open={openStoryDrawer} handleDrawerOpen={handleStoryDrawerOpen} />}
            </>
          )}
          {backProfile && (
            <CardMedia component="img" image={backProfile} sx={{ width: '100%', borderRadius: 1, mt: 1.5 }} title="Slider5 image" />
          )}
        </div>
      )}
    </Draggable>
  );
};

export default Items;
