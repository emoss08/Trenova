import { ReactNode } from 'react';

// material-ui
import { styled } from '@mui/material/styles';
import { Box, Typography } from '@mui/material';
import { treeItemClasses, TreeView, TreeItemProps, TreeItem } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import {
  MailFilled,
  DeleteFilled,
  TagFilled,
  ProfileFilled,
  InfoCircleFilled,
  SnippetsFilled,
  TagsFilled,
  CaretDownFilled,
  CaretRightFilled
} from '@ant-design/icons';

declare module 'react' {
  interface CSSProperties {
    '--tree-view-color'?: string;
    '--tree-view-bg-color'?: string;
  }
}

type StyledTreeItemProps = TreeItemProps & {
  bgColor?: string;
  color?: string;
  labelIcon: ReactNode;
  labelInfo?: string;
  labelText: string;
};

const StyledTreeItemRoot = styled(TreeItem)(({ theme }) => ({
  color: theme.palette.text.secondary,
  [`& .${treeItemClasses.content}`]: {
    color: theme.palette.text.secondary,
    borderTopRightRadius: theme.spacing(2),
    borderBottomRightRadius: theme.spacing(2),
    paddingRight: theme.spacing(1),
    fontWeight: theme.typography.fontWeightMedium,
    '&.Mui-expanded': {
      fontWeight: theme.typography.fontWeightRegular
    },
    '&:hover': {
      backgroundColor: theme.palette.action.hover
    },
    '&.Mui-focused, &.Mui-selected, &.Mui-selected.Mui-focused': {
      backgroundColor: `var(--tree-view-bg-color, ${theme.palette.action.selected})`,
      color: 'var(--tree-view-color)'
    },
    [`& .${treeItemClasses.label}`]: {
      fontWeight: 'inherit',
      color: 'inherit'
    }
  },
  [`& .${treeItemClasses.group}`]: {
    marginLeft: 0,
    [`& .${treeItemClasses.content}`]: {
      paddingLeft: theme.spacing(2)
    }
  }
}));

function StyledTreeItem(props: StyledTreeItemProps) {
  const { bgColor, color, labelIcon, labelInfo, labelText, ...other } = props;

  return (
    <StyledTreeItemRoot
      label={
        <Box sx={{ display: 'flex', alignItems: 'center', p: 0.5, pr: 0 }}>
          <Box sx={{ mr: 1, fontSize: '1rem' }}>{labelIcon}</Box>
          <Typography variant="body2" sx={{ fontWeight: 'inherit', flexGrow: 1 }}>
            {labelText}
          </Typography>
          <Typography variant="caption" color="inherit">
            {labelInfo}
          </Typography>
        </Box>
      }
      style={{
        '--tree-view-color': color,
        '--tree-view-bg-color': bgColor
      }}
      {...other}
    />
  );
}

// ==============================|| TREE VIEW - GMAIL ||============================== //

export default function GmailTreeView() {
  const gmailTreeviewCodeString = `// GmailTreeView.tsx
<TreeView
  aria-label="gmail"
  defaultExpanded={['3']}
  defaultCollapseIcon={<CaretDownFilled />}
  defaultExpandIcon={<CaretRightFilled />}
  defaultEndIcon={<div style={{ width: 24 }} />}
  sx={{ height: 400, flexGrow: 1, overflowY: 'auto' }}
>
  <StyledTreeItem nodeId="1" labelText="All Mail" labelIcon={<MailFilled />} />
  <StyledTreeItem nodeId="2" labelText="Trash" labelIcon={<DeleteFilled />} />
  <StyledTreeItem nodeId="3" labelText="Categories" labelIcon={<TagFilled />}>
    <StyledTreeItem nodeId="5" labelText="Social" labelIcon={<ProfileFilled />} labelInfo="90" color="#1a73e8" bgColor="#e8f0fe" />
    <StyledTreeItem
      nodeId="6"
      labelText="Updates"
      labelIcon={<InfoCircleFilled />}
      labelInfo="2,294"
      color="#e3742f"
      bgColor="#fcefe3"
    />
    <StyledTreeItem
      nodeId="7"
      labelText="Forums"
      labelIcon={<SnippetsFilled />}
      labelInfo="3,566"
      color="#a250f5"
      bgColor="#f3e8fd"
    />
    <StyledTreeItem nodeId="8" labelText="Promotions" labelIcon={<TagsFilled />} labelInfo="733" color="#3c8039" bgColor="#e6f4ea" />
  </StyledTreeItem>
  <StyledTreeItem nodeId="4" labelText="History" labelIcon={<TagFilled />} />
</TreeView>`;

  return (
    <MainCard title="Gmail Clone" codeString={gmailTreeviewCodeString}>
      <TreeView
        aria-label="gmail"
        defaultExpanded={['3']}
        defaultCollapseIcon={<CaretDownFilled />}
        defaultExpandIcon={<CaretRightFilled />}
        defaultEndIcon={<div style={{ width: 24 }} />}
        sx={{ height: 400, flexGrow: 1, overflowY: 'auto' }}
      >
        <StyledTreeItem nodeId="1" labelText="All Mail" labelIcon={<MailFilled />} />
        <StyledTreeItem nodeId="2" labelText="Trash" labelIcon={<DeleteFilled />} />
        <StyledTreeItem nodeId="3" labelText="Categories" labelIcon={<TagFilled />}>
          <StyledTreeItem nodeId="5" labelText="Social" labelIcon={<ProfileFilled />} labelInfo="90" color="#1a73e8" bgColor="#e8f0fe" />
          <StyledTreeItem
            nodeId="6"
            labelText="Updates"
            labelIcon={<InfoCircleFilled />}
            labelInfo="2,294"
            color="#e3742f"
            bgColor="#fcefe3"
          />
          <StyledTreeItem
            nodeId="7"
            labelText="Forums"
            labelIcon={<SnippetsFilled />}
            labelInfo="3,566"
            color="#a250f5"
            bgColor="#f3e8fd"
          />
          <StyledTreeItem nodeId="8" labelText="Promotions" labelIcon={<TagsFilled />} labelInfo="733" color="#3c8039" bgColor="#e6f4ea" />
        </StyledTreeItem>
        <StyledTreeItem nodeId="4" labelText="History" labelIcon={<TagFilled />} />
      </TreeView>
    </MainCard>
  );
}
