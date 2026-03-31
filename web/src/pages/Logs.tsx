import { useEffect, useRef, useState, useCallback } from 'react'
import { Card, Select, Button, Tag, Space, Switch, Tooltip, message, Badge } from 'antd'
import { ClearOutlined, PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined, DownloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { egressApi, nicsApi } from '../services/api'

interface LogEntry {
  id: number
  timestamp: string
  level: string
  message: string
  nic?: string
  egress?: string
  protocol?: string
  target?: string
  client?: string
  fields?: Record<string, any>
  raw?: string
}

interface EgressPolicy {
  id: string
  name: string
  nic: string
}

interface NIC {
  name: string
  ip: string
  display_name: string
}

interface ConnectionEvent {
  id: string
  action: string
  client: string
  target: string
  protocol: string
  egress_id: string
  nic: string
  proxy_used: boolean
}

const levelColors: Record<string, string> = {
  info: 'blue',
  warn: 'orange',
  error: 'red',
  debug: 'cyan',
  fatal: 'magenta',
  trace: 'gray',
}

const levelNames: Record<string, string> = {
  info: '信息',
  warn: '警告',
  error: '错误',
  debug: '调试',
  fatal: '致命',
  trace: '跟踪',
}

const LOG_MAX_COUNT = 1000
const RECONNECT_DELAY = 3000

function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [, setConnections] = useState<ConnectionEvent[]>([])
  const [filter, setFilter] = useState<string>('all')
  const [nicFilter, setNicFilter] = useState<string>('all')
  const [logTypeFilter, setLogTypeFilter] = useState<'all' | 'logs' | 'connections'>('all')
  const [paused, setPaused] = useState(false)
  const [connected, setConnected] = useState(false)
  const [showDebug, setShowDebug] = useState(false)
  const [autoScroll, setAutoScroll] = useState(true)
  const logRef = useRef<HTMLPreElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const logIdRef = useRef(0)
  const pausedRef = useRef(false) // Use ref to avoid re-creating WebSocket on pause toggle
  const manualReconnectRef = useRef(false) // Flag to prevent auto-reconnect when manually reconnecting

  const { data: egressPolicies } = useQuery({
    queryKey: ['egress'],
    queryFn: egressApi.list,
  })

  const { data: nics } = useQuery({
    queryKey: ['nics'],
    queryFn: nicsApi.list,
  })

  // Connect to WebSocket
  const connectWebSocket = useCallback(() => {
    // Use current page's port for WebSocket connection (GUI is served by API server)
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsHost = window.location.hostname || '127.0.0.1'
    const wsPort = window.location.port || (window.location.protocol === 'https:' ? '443' : '80')
    const wsUrl = `${wsProtocol}//${wsHost}:${wsPort}/ws`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
        // Subscribe to log channel
        ws.send(JSON.stringify({ type: 'subscribe', channels: ['log', 'connection', 'traffic'] }))
      }

      ws.onmessage = (event) => {
        if (pausedRef.current) return

        try {
          const data = JSON.parse(event.data)

          if (data.type === 'log') {
            const entry: LogEntry = {
              id: ++logIdRef.current,
              timestamp: data.timestamp || new Date().toISOString(),
              level: data.data?.level || 'info',
              message: data.data?.message || '',
              nic: data.data?.nic || data.data?.fields?.nic,
              egress: data.data?.egress_id || data.data?.fields?.egress_id,
              fields: data.data?.fields,
              raw: event.data,
            }
            setLogs(prev => [...prev.slice(-(LOG_MAX_COUNT - 1)), entry])
          } else if (data.type === 'connection') {
            const conn: ConnectionEvent = {
              id: data.data?.id || `conn-${Date.now()}`,
              action: data.data?.action || 'created',
              client: data.data?.client || '',
              target: data.data?.target || '',
              protocol: data.data?.protocol || '',
              egress_id: data.data?.egress_id || '',
              nic: data.data?.nic || '',
              proxy_used: data.data?.proxy_used || false,
            }
            setConnections(prev => [...prev.slice(-(LOG_MAX_COUNT - 1)), conn])

            // Also add as log entry for visibility
            const logEntry: LogEntry = {
              id: ++logIdRef.current,
              timestamp: data.timestamp || new Date().toISOString(),
              level: 'info',
              message: conn.action === 'created'
                ? `连接建立: ${conn.client} → ${conn.target} (${conn.protocol})`
                : `连接关闭: ${conn.target}`,
              nic: conn.nic,
              egress: conn.egress_id,
              protocol: conn.protocol,
              target: conn.target,
              client: conn.client,
            }
            setLogs(prev => [...prev.slice(-(LOG_MAX_COUNT - 1)), logEntry])
          }
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      ws.onclose = () => {
        setConnected(false)
        // Skip auto-reconnect if this was a manual reconnect
        if (manualReconnectRef.current) {
          manualReconnectRef.current = false
          return
        }
        // Attempt to reconnect after delay
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current)
        }
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('Attempting to reconnect WebSocket...')
          connectWebSocket()
        }, RECONNECT_DELAY)
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        setConnected(false)
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      setConnected(false)
    }
  }, []) // Remove paused from dependency array to avoid reconnection loop

  // Sync pausedRef with paused state
  useEffect(() => {
    pausedRef.current = paused
  }, [paused])

  useEffect(() => {
    connectWebSocket()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      wsRef.current?.close()
    }
  }, [connectWebSocket])

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  // Filter logs
  const filteredLogs = logs.filter(log => {
    // Filter by level
    if (filter !== 'all' && log.level !== filter) return false

    // Hide debug logs if disabled
    if (!showDebug && log.level === 'debug') return false

    // Filter by type
    if (logTypeFilter === 'connections' && !log.protocol) return false
    if (logTypeFilter === 'logs' && log.protocol) return false

    // Filter by NIC/egress
    if (nicFilter !== 'all') {
      if (nicFilter === 'wifi' && !log.nic?.toLowerCase().includes('wlan') && !log.nic?.toLowerCase().includes('wifi')) return false
      if (nicFilter === 'ethernet' && !log.nic?.toLowerCase().includes('eth') && !log.nic?.toLowerCase().includes('以太')) return false
      if (nicFilter.startsWith('egress:') && log.egress !== nicFilter.slice(7)) return false
      if (!nicFilter.startsWith('egress:') && nicFilter !== 'wifi' && nicFilter !== 'ethernet' && log.nic !== nicFilter) return false
    }

    return true
  })

  const handleClear = () => {
    setLogs([])
    setConnections([])
    message.success('日志已清空')
  }

  const handleExport = () => {
    const content = filteredLogs.map(log =>
      `[${log.timestamp}] [${log.level.toUpperCase()}] ${log.message}`
    ).join('\n')

    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `netdispatch-logs-${new Date().toISOString().slice(0, 10)}.txt`
    a.click()
    URL.revokeObjectURL(url)
    message.success('日志已导出')
  }

  // Build filter options
  const filterOptions = [
    { value: 'all', label: '全部来源' },
  ]

  ;(nics || []).forEach((nic: NIC) => {
    const nameLower = nic.name.toLowerCase()
    let filterValue = nic.name
    let label = nic.display_name || nic.name

    if (nameLower.includes('wlan') || nameLower.includes('wifi')) {
      filterValue = 'wifi'
      label = 'WiFi'
    } else if (nameLower.includes('eth') || nameLower.includes('以太')) {
      filterValue = 'ethernet'
      label = '以太网'
    }

    if (!filterOptions.find(o => o.value === filterValue)) {
      filterOptions.push({ value: filterValue, label })
    }
  })

  ;(egressPolicies || []).forEach((policy: EgressPolicy) => {
    filterOptions.push({ value: `egress:${policy.id}`, label: `策略: ${policy.name}` })
  })

  // Count by level
  const levelCounts = logs.reduce((acc, log) => {
    acc[log.level] = (acc[log.level] || 0) + 1
    return acc
  }, {} as Record<string, number>)

  return (
    <div>
      <Card
        title={
          <Space>
            <span>实时日志</span>
            <Badge
              status={connected ? 'success' : 'error'}
              text={connected ? '已连接' : '未连接'}
            />
          </Space>
        }
        extra={
          <Space wrap>
            <Select
              value={logTypeFilter}
              onChange={setLogTypeFilter}
              style={{ width: 120 }}
              options={[
                { value: 'all', label: '全部类型' },
                { value: 'logs', label: '系统日志' },
                { value: 'connections', label: '连接事件' },
              ]}
            />
            <Select
              value={nicFilter}
              onChange={setNicFilter}
              style={{ width: 150 }}
              placeholder="筛选来源"
              options={filterOptions}
            />
            <Select
              value={filter}
              onChange={setFilter}
              style={{ width: 120 }}
              options={[
                { value: 'all', label: '全部级别' },
                { value: 'error', label: `错误 (${levelCounts.error || 0})` },
                { value: 'warn', label: `警告 (${levelCounts.warn || 0})` },
                { value: 'info', label: `信息 (${levelCounts.info || 0})` },
                { value: 'debug', label: `调试 (${levelCounts.debug || 0})` },
              ]}
            />
            <Tooltip title={showDebug ? '隐藏调试日志' : '显示调试日志'}>
              <Switch
                checked={showDebug}
                onChange={setShowDebug}
                checkedChildren="调试"
                unCheckedChildren="调试"
              />
            </Tooltip>
            <Tooltip title={autoScroll ? '关闭自动滚动' : '开启自动滚动'}>
              <Switch
                checked={autoScroll}
                onChange={setAutoScroll}
                checkedChildren="滚动"
                unCheckedChildren="滚动"
              />
            </Tooltip>
            <Tooltip title={paused ? '继续接收日志' : '暂停接收日志'}>
              <Button
                type={paused ? 'primary' : 'default'}
                icon={paused ? <PlayCircleOutlined /> : <PauseCircleOutlined />}
                onClick={() => setPaused(!paused)}
              />
            </Tooltip>
            <Tooltip title="重新连接">
              <Button
                icon={<ReloadOutlined />}
                onClick={() => {
                  // Set flag to prevent auto-reconnect from onclose
                  manualReconnectRef.current = true
                  // Clear any pending reconnect timeout
                  if (reconnectTimeoutRef.current) {
                    clearTimeout(reconnectTimeoutRef.current)
                  }
                  wsRef.current?.close()
                  // Small delay to ensure old connection is cleaned up
                  setTimeout(() => {
                    connectWebSocket()
                  }, 100)
                }}
              />
            </Tooltip>
            <Button icon={<DownloadOutlined />} onClick={handleExport}>
              导出
            </Button>
            <Button icon={<ClearOutlined />} onClick={handleClear}>
              清空
            </Button>
          </Space>
        }
      >
        <pre
          ref={logRef}
          style={{
            height: 'calc(100vh - 320px)',
            overflow: 'auto',
            backgroundColor: '#1e1e1e',
            color: '#d4d4d4',
            padding: 12,
            margin: 0,
            borderRadius: 4,
            fontSize: 12,
            fontFamily: 'Consolas, Monaco, "Courier New", monospace',
          }}
        >
          {filteredLogs.length === 0 ? (
            <div style={{ color: '#666', textAlign: 'center', padding: 40 }}>
              {paused ? '日志已暂停' : '等待日志...'}
            </div>
          ) : (
            filteredLogs.map((log) => (
              <div key={log.id} style={{ marginBottom: 2, wordBreak: 'break-all' }}>
                <span style={{ color: '#6a9955' }}>
                  [{log.timestamp.slice(11, 19)}]
                </span>
                {' '}
                <Tag
                  color={levelColors[log.level] || 'default'}
                  style={{ marginRight: 4, fontSize: 11 }}
                >
                  {levelNames[log.level] || log.level.toUpperCase()}
                </Tag>
                {log.egress && (
                  <Tag color="purple" style={{ marginRight: 4, fontSize: 11 }}>
                    {(egressPolicies || []).find((p: EgressPolicy) => p.id === log.egress)?.name || log.egress}
                  </Tag>
                )}
                {log.nic && !log.egress && (
                  <Tag
                    color={log.nic.toLowerCase().includes('wlan') || log.nic.toLowerCase().includes('wifi') ? 'blue' : 'green'}
                    style={{ marginRight: 4, fontSize: 11 }}
                  >
                    {log.nic}
                  </Tag>
                )}
                {log.protocol && (
                  <Tag color="geekblue" style={{ marginRight: 4, fontSize: 11 }}>
                    {log.protocol.toUpperCase()}
                  </Tag>
                )}
                <span style={{
                  color: log.level === 'error' ? '#f14c4c' :
                         log.level === 'warn' ? '#cca700' :
                         log.level === 'debug' ? '#75beff' : '#d4d4d4'
                }}>
                  {log.message}
                </span>
              </div>
            ))
          )}
        </pre>
        <div style={{ marginTop: 8, color: '#666', fontSize: 12 }}>
          共 {logs.length} 条日志，显示 {filteredLogs.length} 条
          {paused && ' (已暂停)'}
        </div>
      </Card>
    </div>
  )
}

export default Logs