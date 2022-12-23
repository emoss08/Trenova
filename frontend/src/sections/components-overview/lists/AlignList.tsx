// material-ui
import { Divider, List, ListItem, ListItemAvatar, ListItemText, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import AntAvatar from 'components/@extended/Avatar';

const avatarImage = require.context('assets/images/users', true);

// ==============================|| LIST - ALIGN ||============================== //

export default function AlignList() {
  const alignListCodeString = `<List sx={{ width: '100%', bgcolor: 'background.paper' }}>
  <ListItem alignItems="flex-start">
    <ListItemAvatar>
      <AntAvatar alt="Remy Sharp" src={avatarImage('./avatar-1.png')} />
    </ListItemAvatar>
    <ListItemText
      primary="Brunch this weekend?"
      secondary={
        <>
          <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
            Ali Connors
          </Typography>
          {" — I'll be in your neighborhood doing errands this…"}
        </>
      }
    />
  </ListItem>
  <Divider variant="inset" component="li" />
  <ListItem alignItems="flex-start">
    <ListItemAvatar>
      <AntAvatar alt="Travis Howard" src={avatarImage('./avatar-2.png')} />
    </ListItemAvatar>
    <ListItemText
      primary="Summer BBQ"
      secondary={
        <>
          <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
            to Scott, Alex, Jennifer
          </Typography>
          {" — Wish I could come, but I'm out of town this…"}
        </>
      }
    />
  </ListItem>
  <Divider variant="inset" component="li" />
  <ListItem alignItems="flex-start">
    <ListItemAvatar>
      <AntAvatar alt="Cindy Baker" src={avatarImage('./avatar-3.png')} />
    </ListItemAvatar>
    <ListItemText
      primary="Oui Oui"
      secondary={
        <>
          <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
            Sandra Adams
          </Typography>
          {' — Do you have Paris recommendations? Have you ever…'}
        </>
      }
    />
  </ListItem>
</List>`;

  return (
    <MainCard content={false} codeString={alignListCodeString}>
      <List sx={{ width: '100%', bgcolor: 'background.paper' }}>
        <ListItem alignItems="flex-start">
          <ListItemAvatar>
            <AntAvatar alt="Remy Sharp" src={avatarImage(`./avatar-1.png`)} />
          </ListItemAvatar>
          <ListItemText
            primary="Brunch this weekend?"
            secondary={
              <>
                <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
                  Ali Connors
                </Typography>
                {" — I'll be in your neighborhood doing errands this…"}
              </>
            }
          />
        </ListItem>
        <Divider variant="inset" component="li" />
        <ListItem alignItems="flex-start">
          <ListItemAvatar>
            <AntAvatar alt="Travis Howard" src={avatarImage(`./avatar-2.png`)} />
          </ListItemAvatar>
          <ListItemText
            primary="Summer BBQ"
            secondary={
              <>
                <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
                  to Scott, Alex, Jennifer
                </Typography>
                {" — Wish I could come, but I'm out of town this…"}
              </>
            }
          />
        </ListItem>
        <Divider variant="inset" component="li" />
        <ListItem alignItems="flex-start">
          <ListItemAvatar>
            <AntAvatar alt="Cindy Baker" src={avatarImage(`./avatar-3.png`)} />
          </ListItemAvatar>
          <ListItemText
            primary="Oui Oui"
            secondary={
              <>
                <Typography sx={{ display: 'inline' }} component="span" variant="body2" color="text.primary">
                  Sandra Adams
                </Typography>
                {' — Do you have Paris recommendations? Have you ever…'}
              </>
            }
          />
        </ListItem>
      </List>
    </MainCard>
  );
}
