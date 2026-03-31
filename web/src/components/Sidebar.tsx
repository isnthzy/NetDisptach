import { Menu, Drawer, Button } from 'antd'
import {
  DashboardOutlined,
  CloudServerOutlined,
  OrderedListOutlined,
  FileTextOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
  MenuOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'
import { useState, useEffect } from 'react'

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

interface SidebarProps {
  isMobile?: boolean
}

function Sidebar({ isMobile }: SidebarProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const [drawerOpen, setDrawerOpen] = useState(false)

  useEffect(() => {
    // Close drawer on route change
    setDrawerOpen(false)
  }, [location.pathname])

  const handleNavigate = (key: string) => {
    navigate(key)
    if (isMobile) {
      setDrawerOpen(false)
    }
  }

  const menuContent = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{
        height: '64px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        borderBottom: '1px solid #f0f0f0',
        fontWeight: 'bold',
        fontSize: '18px',
        background: 'linear-gradient(135deg, #1890ff, #40a9ff)',
        color: '#fff',
      }}>
        NetDispatch
      </div>
      <Menu
        mode="inline"
        selectedKeys={[location.pathname]}
        items={items}
        onClick={({ key }) => handleNavigate(key)}
        style={{ border: 'none', flex: 1 }}
      />
    </div>
  )

  // Mobile: Show hamburger menu button
  if (isMobile) {
    return (
      <>
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          height: '56px',
          background: '#fff',
          borderBottom: '1px solid #e8e8e8',
          display: 'flex',
          alignItems: 'center',
          padding: '0 16px',
          zIndex: 100,
          boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
        }}>
          <Button
            type="text"
            icon={<MenuOutlined style={{ fontSize: '20px' }} />}
            onClick={() => setDrawerOpen(true)}
          />
          <span style={{
            marginLeft: '12px',
            fontWeight: 'bold',
            fontSize: '18px',
            background: 'linear-gradient(135deg, #1890ff, #40a9ff)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
          }}>
            NetDispatch
          </span>
        </div>
        <Drawer
          placement="left"
          open={drawerOpen}
          onClose={() => setDrawerOpen(false)}
          width={260}
          styles={{ body: { padding: 0 } }}
          closeIcon={null}
        >
          {menuContent}
        </Drawer>
        {/* Spacer for fixed header */}
        <div style={{ height: '56px' }} />
      </>
    )
  }

  // Desktop: Regular sidebar
  return menuContent
}

export default Sidebar
