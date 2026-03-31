import { useState } from 'react'
import { Card, Button, Table, Modal, Form, Input, Select, Switch, Space, message } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { egressApi, nicsApi } from '../services/api'

interface EgressPolicy {
  id: string
  name: string
  nic: string
  proxy?: {
    host: string
    port: number
    protocol: string
    username?: string
    password?: string
  }
  description?: string
}

function Egress() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editingPolicy, setEditingPolicy] = useState<EgressPolicy | null>(null)
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const { data: policies } = useQuery({
    queryKey: ['egress'],
    queryFn: egressApi.list,
  })

  const { data: nics } = useQuery({
    queryKey: ['nics'],
    queryFn: nicsApi.list,
  })

  const createMutation = useMutation({
    mutationFn: egressApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['egress'] })
      message.success('策略已创建')
      setModalOpen(false)
    },
    onError: (error: any) => {
      console.error('Create egress error:', error)
      message.error('创建策略失败: ' + (error.response?.data?.error || error.message || '未知错误'))
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: EgressPolicy }) => egressApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['egress'] })
      message.success('策略已更新')
      setModalOpen(false)
    },
    onError: (error: any) => {
      console.error('Update egress error:', error)
      message.error('更新策略失败: ' + (error.response?.data?.error || error.message || '未知错误'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: egressApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['egress'] })
      message.success('策略已删除')
    },
  })

  const handleAdd = () => {
    setEditingPolicy(null)
    form.resetFields()
    form.setFieldsValue({ useProxy: false })
    setModalOpen(true)
  }

  const handleEdit = (record: EgressPolicy) => {
    setEditingPolicy(record)
    form.setFieldsValue({
      name: record.name,
      nic: record.nic,
      useProxy: !!record.proxy,
      proxyHost: record.proxy?.host,
      proxyPort: record.proxy?.port,
      proxyProtocol: record.proxy?.protocol || 'socks5',
      proxyUsername: record.proxy?.username,
      proxyPassword: record.proxy?.password,
      description: record.description,
    })
    setModalOpen(true)
  }

  const handleSubmit = () => {
    form.validateFields().then(values => {
      const policy: EgressPolicy = {
        id: editingPolicy?.id || `policy-${Date.now()}`,
        name: values.name,
        nic: values.nic,
        description: values.description,
      }

      if (values.useProxy) {
        policy.proxy = {
          host: values.proxyHost,
          port: parseInt(values.proxyPort) || 0,
          protocol: values.proxyProtocol,
          username: values.proxyUsername || '',
          password: values.proxyPassword || '',
        }
      }

      if (editingPolicy) {
        updateMutation.mutate({ id: editingPolicy.id, data: policy })
      } else {
        createMutation.mutate(policy)
      }
    }).catch((error) => {
      console.error('Form validation error:', error)
      message.error('请填写必填字段')
    })
  }

  const handleDelete = (record: EgressPolicy) => {
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除策略 "${record.name}" 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk() {
        deleteMutation.mutate(record.id)
      },
    })
  }

  const columns = [
    { title: '策略名称', dataIndex: 'name', key: 'name' },
    { title: '网卡', dataIndex: 'nic', key: 'nic' },
    {
      title: '代理服务器',
      key: 'proxy',
      render: (_: any, record: EgressPolicy) =>
        record.proxy
          ? `${record.proxy.protocol}://${record.proxy.host}:${record.proxy.port}`
          : '直连',
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: EgressPolicy) => (
        <Space>
          <Button icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Button
            icon={<DeleteOutlined />}
            danger
            onClick={() => handleDelete(record)}
          />
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Card
        title="出口策略管理"
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>添加策略</Button>}
      >
        <Table
          columns={columns}
          dataSource={policies || []}
          rowKey="id"
          pagination={false}
        />
      </Card>

      <Modal
        title={editingPolicy ? '编辑策略' : '添加策略'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="策略名称" rules={[{ required: true, message: '请输入策略名称' }]}>
            <Input placeholder="例如：WiFi走代理" />
          </Form.Item>

          <Form.Item name="nic" label="选择网卡" rules={[{ required: true, message: '请选择网卡' }]}>
            <Select placeholder="选择网卡">
              {(nics || []).map((nic: any) => (
                <Select.Option key={nic.name} value={nic.name}>
                  {nic.display_name || nic.name} ({nic.ip})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="useProxy" label="使用代理服务器" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Form.Item noStyle shouldUpdate>
            {({ getFieldValue }) =>
              getFieldValue('useProxy') && (
                <>
                  <Form.Item name="proxyProtocol" label="代理协议">
                    <Select>
                      <Select.Option value="socks5">SOCKS5</Select.Option>
                      <Select.Option value="http">HTTP</Select.Option>
                    </Select>
                  </Form.Item>

                  <Form.Item name="proxyHost" label="代理地址" rules={[{ required: true, message: '请输入代理地址' }]}>
                    <Input placeholder="192.168.1.100" />
                  </Form.Item>

                  <Form.Item name="proxyPort" label="代理端口" rules={[{ required: true, message: '请输入代理端口' }]}>
                    <Input type="number" placeholder="1080" />
                  </Form.Item>

                  <Form.Item name="proxyUsername" label="用户名">
                    <Input />
                  </Form.Item>

                  <Form.Item name="proxyPassword" label="密码">
                    <Input.Password />
                  </Form.Item>
                </>
              )
            }
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Egress
