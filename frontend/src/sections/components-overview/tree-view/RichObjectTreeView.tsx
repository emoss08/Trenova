// material-ui
import { TreeView, TreeItem } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// assets
import { DownOutlined, RightOutlined } from '@ant-design/icons';

interface RenderTree {
  id: string;
  name: string;
  children?: readonly RenderTree[];
}

const data: RenderTree = {
  id: 'root',
  name: 'Parent',
  children: [
    {
      id: '1',
      name: 'Child - 1'
    },
    {
      id: '3',
      name: 'Child - 3',
      children: [
        {
          id: '4',
          name: 'Child - 4'
        }
      ]
    }
  ]
};

// ==============================|| TREE VIEW - RICH OBJECT ||============================== //

export default function RichObjectTreeView() {
  const renderTree = (nodes: RenderTree) => (
    <TreeItem key={nodes.id} nodeId={nodes.id} label={nodes.name}>
      {Array.isArray(nodes.children) ? nodes.children.map((node) => renderTree(node)) : null}
    </TreeItem>
  );

  const richTreeviewCodeString = `// RichObjectTreeView.tsx
<TreeView
  aria-label="rich object"
  defaultCollapseIcon={<DownOutlined />}
  defaultExpandIcon={<RightOutlined />}
  defaultExpanded={['root']}
  sx={{ height: 180, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
>
  {renderTree(data)}
</TreeView>`;

  return (
    <MainCard title="Rich Object" codeString={richTreeviewCodeString}>
      <TreeView
        aria-label="rich object"
        defaultCollapseIcon={<DownOutlined />}
        defaultExpandIcon={<RightOutlined />}
        defaultExpanded={['root']}
        sx={{ height: 180, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
      >
        {renderTree(data)}
      </TreeView>
    </MainCard>
  );
}
