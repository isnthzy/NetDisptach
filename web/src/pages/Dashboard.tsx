import { Row, Col, Card, Statistic, Table } from 'antd'
import {
  ApiOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  WifiOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { useQuery } from '@tanstack/react-query'
import { useState, useEffect, useRef } from 'react'
import { statsApi } from '../services/api'

interface TrafficPoint {
  timestamp: number
  bytes_in_rate: number
  bytes_out_rate: number
}

interface TrafficStats {
  bytes_in: number
  bytes_out: number
  active_connections: number
  total_connections: number
}

// Custom hook for WebSocket connection
function useWebSocketStats() {
  const [stats, setStats] = useState<TrafficStats | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)

  useEffect(() => {
    const connect = () => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${window.location.host}/ws`

      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
        console.log('WebSocket connected')
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'traffic') {
            setStats({
              bytes_in: data.data.bytes_in,
              bytes_out: data.data.bytes_out,
              active_connections: data.data.active_connections,
              total_connections: stats?.total_connections || 0,
            })
          }
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      ws.onclose = () => {
        setConnected(false)
        console.log('WebSocket disconnected, reconnecting...')
        // Reconnect after 3 seconds
        reconnectTimeoutRef.current = window.setTimeout(connect, 3000)
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    }

    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  return { stats, connected }
}

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) {
    return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB'
  }
  if (bytes >= 1024 * 1024) {
    return (bytes / 1024 / 1024).toFixed(2) + ' MB'
  }
  if (bytes >= 1024) {
    return (bytes / 1024).toFixed(2) + ' KB'
  }
  return bytes + ' B'
}

function Dashboard() {
  // Use WebSocket for real-time stats
  const { stats: wsStats, connected } = useWebSocketStats()

  // Fall back to polling for total_connections and traffic history
  const { data: apiStats } = useQuery({
    queryKey: ['stats'],
    queryFn: statsApi.getOverview,
    refetchInterval: 10000, // Less frequent as backup
  })

  const { data: trafficHistory } = useQuery({
    queryKey: ['trafficHistory'],
    queryFn: statsApi.getHistory,
    refetchInterval: 2000, // Keep polling for history chart
  })

  const { data: recentConnections } = useQuery({
    queryKey: ['recentConnections'],
    queryFn: statsApi.getRecentConnections,
    refetchInterval: 5000,
  })

  // Merge WebSocket stats with API stats
  const stats = {
    active_connections: wsStats?.active_connections ?? apiStats?.active_connections ?? 0,
    bytes_in: wsStats?.bytes_in ?? apiStats?.bytes_in ?? 0,
    bytes_out: wsStats?.bytes_out ?? apiStats?.bytes_out ?? 0,
    total_connections: apiStats?.total_connections ?? 0,
  }

  const trafficOption = {
    title: { text: '实时流量', left: 'center' },
    tooltip: { trigger: 'axis' },
    legend: { data: ['入站', '出站'], bottom: 0 },
    xAxis: {
      type: 'category',
      data: (trafficHistory || []).map((p: TrafficPoint) => {
        const date = new Date(p.timestamp * 1000)
        return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`
      }),
      axisLabel: { rotate: 45, fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      name: '速率 (B/s)',
      axisLabel: {
        formatter: (value: number) => {
          if (value >= 1024 * 1024) return (value / 1024 / 1024).toFixed(1) + ' MB/s'
          if (value >= 1024) return (value / 1024).toFixed(1) + ' KB/s'
          return value + ' B/s'
        }
      }
    },
    series: [
      {
        name: '入站',
        type: 'line',
        smooth: true,
        data: (trafficHistory || []).map((p: TrafficPoint) => p.bytes_in_rate),
        areaStyle: { opacity: 0.3 },
        itemStyle: { color: '#52c41a' },
      },
      {
        name: '出站',
        type: 'line',
        smooth: true,
        data: (trafficHistory || []).map((p: TrafficPoint) => p.bytes_out_rate),
        areaStyle: { opacity: 0.3 },
        itemStyle: { color: '#1890ff' },
      },
    ],
    grid: { bottom: 80 },
  }

  const columns = [
    { title: '客户端', dataIndex: 'client_addr', key: 'client', width: 180 },
    { title: '目标地址', dataIndex: 'target_addr', key: 'target', ellipsis: true },
    { title: '协议', dataIndex: 'protocol', key: 'protocol', width: 80 },
    { title: '出口策略', dataIndex: 'egress_id', key: 'egress', width: 120 },
    { title: '网卡', dataIndex: 'nic', key: 'nic', width: 100 },
  ]

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title={
                <span>
                  活跃连接
                  {connected && <WifiOutlined style={{ marginLeft: 8, color: '#52c41a' }} />}
                </span>
              }
              value={stats.active_connections}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="入站流量"
              value={formatBytes(stats.bytes_in)}
              prefix={<CloudDownloadOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="出站流量"
              value={formatBytes(stats.bytes_out)}
              prefix={<CloudUploadOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="总连接数"
              value={stats.total_connections}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }}>
        <ReactECharts option={trafficOption} style={{ height: 350 }} />
      </Card>

      <Card title="最近连接" style={{ marginTop: 16 }}>
        <Table
          columns={columns}
          dataSource={recentConnections || []}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          size="small"
          scroll={{ x: 600 }}
        />
      </Card>
    </div>
  )
}

export default Dashboard
