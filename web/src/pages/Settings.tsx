import { Form, Card, Input, InputNumber, Switch, Select, message, Divider, Row, Col, Tag, Alert, Button } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { configApi, nicsApi, statusApi } from '../services/api'
import { useEffect, useState } from 'react'

interface SOCKSUser {
  username: string
  password: string
}

function Settings() {
  const [form] = Form.useForm()
  const queryClient = useQueryClient()
  const [serverEnabled, setServerEnabled] = useState(true)

  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: configApi.get,
  })

  const { data: nics } = useQuery({
    queryKey: ['nics'],
    queryFn: nicsApi.list,
  })

  const { data: status } = useQuery({
    queryKey: ['status'],
    queryFn: statusApi.get,
    refetchInterval: 5000,
  })

  const updateMutation = useMutation({
    mutationFn: configApi.update,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      queryClient.invalidateQueries({ queryKey: ['status'] })
      message.success('设置已保存')
    },
    onError: () => {
      message.error('保存设置失败')
    },
  })

  // Set form values when config loads
  useEffect(() => {
    if (config) {
      form.setFieldsValue(config)
      setServerEnabled(config.server?.enabled ?? true)
    }
  }, [config, form])

  // Real-time config update function
  // Uses the original config from API instead of form values to preserve all fields
  const updateConfig = (path: string, value: any) => {
    if (!config) return

    // Deep clone the original config to preserve all fields (egress, routing, etc.)
    const updated = JSON.parse(JSON.stringify(config))

    // Update only the specific path
    const pathParts = path.split('.')
    let current: any = updated
    for (let i = 0; i < pathParts.length - 1; i++) {
      current = current[pathParts[i]]
    }
    current[pathParts[pathParts.length - 1]] = value

    updateMutation.mutate(updated)
  }

  // Update SOCKS5 auth users
  const updateSOCKS5Users = (users: SOCKSUser[]) => {
    if (!config) return

    const updated = JSON.parse(JSON.stringify(config))
    updated.server.socks5.auth.users = users
    updateMutation.mutate(updated)
  }

  if (isLoading) return null

  return (
    <div>
      <Card title="代理服务设置">
        <Form
          form={form}
          layout="vertical"
          initialValues={config}
        >
          <Divider orientation="left">代理开关</Divider>

          <Form.Item name={['server', 'enabled']} valuePropName="checked" label="启用代理转发">
            <Switch
              onChange={(checked) => {
                setServerEnabled(checked)
                if (!checked) {
                  // 关闭总开关时，同步关闭两个端口的开关
                  if (config) {
                    const updated = JSON.parse(JSON.stringify(config))
                    updated.server.enabled = false
                    updated.server.http.enabled = false
                    updated.server.socks5.enabled = false
                    updateMutation.mutate(updated)
                    form.setFieldsValue({
                      server: {
                        ...updated.server,
                        http: { ...updated.server.http },
                        socks5: { ...updated.server.socks5 },
                      },
                    })
                  }
                } else {
                  updateConfig('server.enabled', true)
                }
              }}
            />
          </Form.Item>

          {!serverEnabled && (
            <Alert
              message="代理服务已停止"
              description="开启总开关后，需要手动启用 HTTP 或 SOCKS5 代理端口。"
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}

          <Divider orientation="left">绑定地址</Divider>

          <Form.Item name={['server', 'bind']} label="绑定地址" rules={[{ required: true, message: '请选择绑定地址' }]}>
            <Select
              placeholder="选择绑定地址"
              onChange={(value) => updateConfig('server.bind', value)}
              disabled={!serverEnabled}
            >
              {(nics || []).map((nic: any) => (
                <Select.Option key={nic.ip} value={nic.ip}>
                  {nic.ip} ({nic.display_name || nic.name})
                </Select.Option>
              ))}
              <Select.Option value="0.0.0.0">0.0.0.0 (所有网卡)</Select.Option>
              <Select.Option value="127.0.0.1">127.0.0.1 (仅本机)</Select.Option>
            </Select>
          </Form.Item>

          <Divider orientation="left">代理端口</Divider>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name={['server', 'http', 'port']} label="HTTP/HTTPS 代理端口">
                <InputNumber
                  min={1}
                  max={65535}
                  style={{ width: '100%' }}
                  disabled={!serverEnabled}
                  onBlur={(e) => {
                    const value = parseInt(e.target.value)
                    if (!isNaN(value)) {
                      updateConfig('server.http.port', value)
                    }
                  }}
                />
              </Form.Item>
              <Form.Item label="启用">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Form.Item name={['server', 'http', 'enabled']} valuePropName="checked" noStyle>
                    <Switch
                      onChange={(checked) => updateConfig('server.http.enabled', checked)}
                      disabled={!serverEnabled}
                    />
                  </Form.Item>
                  {status && (
                    <Tag color={status.http_running ? 'green' : 'default'}>
                      {status.http_running ? '运行中' : '已停止'}
                    </Tag>
                  )}
                </div>
              </Form.Item>
            </Col>

            <Col span={12}>
              <Form.Item name={['server', 'socks5', 'port']} label="SOCKS5 端口">
                <InputNumber
                  min={1}
                  max={65535}
                  style={{ width: '100%' }}
                  disabled={!serverEnabled}
                  onBlur={(e) => {
                    const value = parseInt(e.target.value)
                    if (!isNaN(value)) {
                      updateConfig('server.socks5.port', value)
                    }
                  }}
                />
              </Form.Item>
              <Form.Item label="启用">
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Form.Item name={['server', 'socks5', 'enabled']} valuePropName="checked" noStyle>
                    <Switch
                      onChange={(checked) => updateConfig('server.socks5.enabled', checked)}
                      disabled={!serverEnabled}
                    />
                  </Form.Item>
                  {status && (
                    <Tag color={status.socks5_running ? 'green' : 'default'}>
                      {status.socks5_running ? '运行中' : '已停止'}
                    </Tag>
                  )}
                </div>
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left">SOCKS5 认证</Divider>

          <Form.Item name={['server', 'socks5', 'auth', 'enabled']} valuePropName="checked" label="启用 SOCKS5 认证">
            <Switch
              onChange={(checked) => updateConfig('server.socks5.auth.enabled', checked)}
              disabled={!serverEnabled}
            />
          </Form.Item>

          <Form.Item noStyle shouldUpdate>
            {({ getFieldValue }) =>
              getFieldValue(['server', 'socks5', 'auth', 'enabled']) && (
                <Card size="small" style={{ marginBottom: 16 }}>
                  <Form.List name={['server', 'socks5', 'auth', 'users']}>
                    {(fields, { add, remove }) => (
                      <>
                        {fields.map(({ key, name, ...restField }) => (
                          <Row key={key} gutter={16} style={{ marginBottom: 8 }}>
                            <Col span={10}>
                              <Form.Item
                                {...restField}
                                name={[name, 'username']}
                                rules={[{ required: true, message: '请输入用户名' }]}
                              >
                                <Input
                                  placeholder="用户名"
                                  onBlur={() => {
                                    const users = form.getFieldValue(['server', 'socks5', 'auth', 'users'])
                                    updateSOCKS5Users(users || [])
                                  }}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={10}>
                              <Form.Item
                                {...restField}
                                name={[name, 'password']}
                                rules={[{ required: true, message: '请输入密码' }]}
                              >
                                <Input.Password
                                  placeholder="密码"
                                  onBlur={() => {
                                    const users = form.getFieldValue(['server', 'socks5', 'auth', 'users'])
                                    updateSOCKS5Users(users || [])
                                  }}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={4}>
                              <Button
                                type="text"
                                danger
                                icon={<DeleteOutlined />}
                                onClick={() => {
                                  remove(name)
                                  // Get current users after removal
                                  setTimeout(() => {
                                    const users = form.getFieldValue(['server', 'socks5', 'auth', 'users']) || []
                                    updateSOCKS5Users(users)
                                  }, 0)
                                }}
                              />
                            </Col>
                          </Row>
                        ))}
                        <Form.Item>
                          <Button
                            type="dashed"
                            onClick={() => add()}
                            block
                            icon={<PlusOutlined />}
                          >
                            添加用户
                          </Button>
                        </Form.Item>
                      </>
                    )}
                  </Form.List>
                </Card>
              )
            }
          </Form.Item>

          <Divider orientation="left">认证设置</Divider>

          <Form.Item name={['api', 'auth', 'enabled']} valuePropName="checked" label="启用 Web 控制台认证">
            <Switch onChange={(checked) => updateConfig('api.auth.enabled', checked)} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name={['api', 'auth', 'username']} label="用户名">
                <Input
                  onBlur={(e) => updateConfig('api.auth.username', e.target.value)}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name={['api', 'auth', 'password']} label="密码">
                <Input.Password
                  onBlur={(e) => updateConfig('api.auth.password', e.target.value)}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Card>
    </div>
  )
}

export default Settings
