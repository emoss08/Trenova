import React, { useState, CSSProperties } from 'react';

// material-ui
import { Theme, useTheme } from '@mui/material/styles';
import {
  Box,
  Collapse,
  Link,
  Menu,
  MenuItem,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Tooltip,
  Typography
} from '@mui/material';

// third-party
import { format } from 'date-fns';
import { Droppable, Draggable } from 'react-beautiful-dnd';

// project imports
import AddItem from './AddItem';
import EditStory from './EditStory';
import AlertStoryDelete from './AlertStoryDelete';
import Items from './Items';
import { openSnackbar } from 'store/reducers/snackbar';
import { useDispatch, useSelector } from 'store';
import { deleteStory } from 'store/reducers/kanban';
import IconButton from 'components/@extended/IconButton';

// assets
import { DownOutlined, MoreOutlined, RightOutlined, PlusSquareTwoTone, ClusterOutlined } from '@ant-design/icons';

// types
import { KanbanUserStory } from 'types/kanban';

interface Props {
  story: KanbanUserStory;
  index: number;
}

// drag wrapper
const getDragWrapper = (isDragging: boolean, theme: Theme, open: boolean): CSSProperties | undefined => {
  let bgcolor = 'transparent';
  if (open) {
    bgcolor = theme.palette.mode === 'dark' ? theme.palette.background.default : theme.palette.secondary.lighter;
  }

  if (isDragging) {
    bgcolor = theme.palette.mode === 'dark' ? theme.palette.background.default : theme.palette.primary.lighter;
  }

  return {
    backgroundColor: bgcolor,
    userSelect: 'none'
  };
};

// drop wrapper
const getDropWrapper = (isDraggingOver: boolean, theme: Theme) => {
  return {
    background: 'transparent'
  };
};

// ==============================|| KANBAN BACKLOGS - USER STORY ||============================== //

const UserStory = ({ story, index }: Props) => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const kanban = useSelector((state) => state.kanban);
  const { columns, profiles, userStory, userStoryOrder } = kanban;
  const [open, setOpen] = React.useState(index === 0);

  const storyColumn = columns.filter((column) => column.id === story.columnId)[0];
  const storyProfile = profiles.filter((profile) => profile.id === story.assign)[0];

  // drawer
  const [openDrawer, setOpenDrawer] = useState<boolean>(false);
  const handleDrawerOpen = () => {
    setOpenDrawer((prevState) => !prevState);
  };

  const addItem = () => {
    setOpenDrawer((prevState) => !prevState);
  };

  const [openStoryDrawer, setOpenStoryDrawer] = useState<boolean>(false);
  const handleStoryDrawerOpen = () => {
    setOpenStoryDrawer((prevState) => !prevState);
  };

  const editStory = () => {
    setOpenStoryDrawer((prevState) => !prevState);
  };

  const [anchorEl, setAnchorEl] = useState<Element | ((element: Element) => Element) | null | undefined>(null);
  const handleClick = (event: React.MouseEvent<HTMLButtonElement> | undefined) => {
    setAnchorEl(event?.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const [openModal, setOpenModal] = useState(false);

  const handleModalClose = (status: boolean) => {
    setOpenModal(false);
    if (status) {
      dispatch(deleteStory(story.id, userStory, userStoryOrder));
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
    <>
      <Draggable draggableId={story.id} index={index}>
        {(provided, snapshot) => (
          <>
            <TableRow
              hover
              {...provided.draggableProps}
              {...provided.dragHandleProps}
              ref={provided.innerRef}
              sx={{
                ...getDragWrapper(snapshot.isDragging, theme, open),
                ...(!open && {
                  '& .MuiTableCell-root': {
                    border: 'none'
                  }
                })
              }}
            >
              <TableCell sx={{ pl: 3, minWidth: 120, width: 120, height: 46 }}>
                <Stack direction="row" spacing={0.5} alignItems="center">
                  <Tooltip title="Add Task">
                    <IconButton aria-label="expand row" onClick={addItem} size="small" sx={{ fontSize: '1.15rem' }}>
                      <PlusSquareTwoTone color={theme.palette.primary.main} />
                    </IconButton>
                  </Tooltip>
                  <IconButton aria-label="expand row" size="small" onClick={() => setOpen(!open)} color="secondary">
                    {open ? <DownOutlined /> : <RightOutlined />}
                  </IconButton>
                </Stack>
              </TableCell>
              <TableCell sx={{ width: 110, minWidth: 110 }}>
                <Stack direction="row" spacing={0.75} alignItems="center">
                  <ClusterOutlined style={{ color: theme.palette.primary.main, marginTop: -2 }} />
                  <Typography variant="subtitle2">{story.id}</Typography>
                </Stack>
              </TableCell>
              <TableCell sx={{ maxWidth: 'calc(100vw - 850px)', minWidth: 140 }} component="th" scope="row">
                <Link
                  underline="hover"
                  color="default"
                  onClick={editStory}
                  sx={{
                    overflow: 'hidden',
                    display: 'block',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    ':hover': { color: 'primary.main' },
                    cursor: 'pointer'
                  }}
                >
                  {story.title}
                </Link>
              </TableCell>
              <TableCell sx={{ width: 60, minWidth: 60 }}>
                <IconButton size="small" aria-controls="menu-comment" onClick={handleClick} aria-haspopup="true" color="secondary">
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
                      editStory();
                    }}
                  >
                    Edit
                  </MenuItem>
                  <MenuItem
                    onClick={() => {
                      handleClose();
                      setOpenModal(true);
                    }}
                  >
                    Delete
                  </MenuItem>
                </Menu>
                {openModal && <AlertStoryDelete title={story.title} open={openModal} handleClose={handleModalClose} />}
              </TableCell>
              <TableCell sx={{ width: 90, minWidth: 90 }}>{storyColumn ? storyColumn.title : ''}</TableCell>
              <TableCell sx={{ width: 140, minWidth: 140 }}>{storyProfile ? storyProfile.name : ''}</TableCell>
              <TableCell sx={{ width: 85, minWidth: 85, textTransform: 'capitalize' }}>{story.priority}</TableCell>
              <TableCell sx={{ width: 120, minWidth: 120 }}>{story.dueDate ? format(new Date(story.dueDate), 'd MMM yyyy') : ''}</TableCell>
            </TableRow>

            <Droppable droppableId={story.id} type="item">
              {(providedDrop, snapshotDrop) => (
                <TableRow
                  ref={providedDrop.innerRef}
                  {...providedDrop.droppableProps}
                  sx={getDropWrapper(snapshotDrop.isDraggingOver, theme)}
                >
                  <TableCell style={{ padding: 0 }} colSpan={8}>
                    <Collapse in={open} timeout="auto" unmountOnExit>
                      {open && (
                        <Box sx={{ margin: 0 }}>
                          <TableContainer>
                            <Table size="small" aria-label="purchases">
                              <TableBody>
                                {story.itemIds?.map((itemId: string, i: number) => (
                                  <Items key={itemId} itemId={itemId} index={i} />
                                ))}
                                {providedDrop.placeholder}
                              </TableBody>
                            </Table>
                          </TableContainer>
                        </Box>
                      )}
                    </Collapse>
                  </TableCell>
                </TableRow>
              )}
            </Droppable>
          </>
        )}
      </Draggable>
      <EditStory story={story} open={openStoryDrawer} handleDrawerOpen={handleStoryDrawerOpen} />
      <AddItem open={openDrawer} handleDrawerOpen={handleDrawerOpen} storyId={story.id} />
    </>
  );
};

export default UserStory;
