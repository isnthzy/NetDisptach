import { Menu } from 'antd'
import {
  DashboardOutlined,
  CloudServerOutlined,
  OrderedListOutlined,
  FileTextOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'

const items = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: '/egress',
    icon: <CloudServerOutlined />,
    label: '出口策略',
  },
  {
    key: '/rules',
    icon: <OrderedListOutlined />,
    label: '路由规则',
  },
  {
    key: '/logs',
    icon: <FileTextOutlined />,
    label: '日志',
  },
  {
    key: '/settings',
    icon: <SettingOutlined />,
    label: '设置',
  },
  {
    key: '/help',
    icon: <QuestionCircleOutlined />,
    label: '使用手册',
  },
]

function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{
        height: '64px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderBottom: '1px solid #f0f0f0',
        fontWeight: 'bold',
        fontSize: '18px',
      }}>
        NetDispatch
      </div>
      <Menu
        mode="inline"
        selectedKeys={[location.pathname]}
        items={items}
        onClick={({ key }) => navigate(key)}
        style={{ border: 'none', flex: 1 }}
      />
    </div>
  )
}

export default Sidebar
