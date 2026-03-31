import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import Sidebar from './components/Sidebar'
import Dashboard from './pages/Dashboard'
import Egress from './pages/Egress'
import Rules from './pages/Rules'
import Logs from './pages/Logs'
import Settings from './pages/Settings'
import Help from './pages/Help'

const { Content, Sider } = Layout

function App() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} theme="light">
        <Sidebar />
      </Sider>
      <Layout>
        <Content style={{ margin: '16px' }}>
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
