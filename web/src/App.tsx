import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import { useState, useEffect } from 'react'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import Egress from './pages/Egress'
import Rules from './pages/Rules'
import Logs from './pages/Logs'
import Settings from './pages/Settings'
import Help from './pages/Help'

const { Content, Sider } = Layout

function App() {
  const [isMobile, setIsMobile] = useState(false)

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768)
    }

    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {!isMobile && (
        <Sider width={220} theme="light" style={{ position: 'fixed', left: 0, top: 0, bottom: 0 }}>
          <Sidebar isMobile={false} />
        </Sider>
      )}
      <Layout style={{ marginLeft: isMobile ? 0 : 220 }}>
        {isMobile && <Sidebar isMobile={true} />}
        <Content style={{ margin: isMobile ? '0' : '16px', marginTop: isMobile ? 0 : undefined }}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/egress" element={<Egress />} />
            <Route path="/rules" element={<Rules />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/help" element={<Help />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
