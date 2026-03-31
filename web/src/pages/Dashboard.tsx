import { Row, Col, Card, Statistic, Table } from 'antd'
import {
  ApiOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { useQuery } from '@tanstack/react-query'
import { statsApi } from '../services/api'

interface TrafficPoint {
  timestamp: number
  bytes_in_rate: number
  bytes_out_rate: number
}

function Dashboard() {
  const { data: stats } = useQuery({
    queryKey: ['stats'],
    queryFn: statsApi.getOverview,
    refetchInterval: 5000,
  })

  const { data: trafficHistory } = useQuery({
    queryKey: ['trafficHistory'],
    queryFn: statsApi.getHistory,
    refetchInterval: 5000,
  })

  const { data: recentConnections } = useQuery({
    queryKey: ['recentConnections'],
    queryFn: statsApi.getRecentConnections,
    refetchInterval: 5000,
  })

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
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="活跃连接"
              value={stats?.active_connections || 0}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="入站流量"
              value={stats?.bytes_in || 0}
              prefix={<CloudDownloadOutlined />}
              suffix="字节"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="出站流量"
              value={stats?.bytes_out || 0}
              prefix={<CloudUploadOutlined />}
              suffix="字节"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总连接数"
              value={stats?.total_connections || 0}
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
        />
      </Card>
    </div>
  )
}

export default Dashboard
