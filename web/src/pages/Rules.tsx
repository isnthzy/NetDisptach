import { useState, useRef } from 'react'
import { Card, Button, Table, Modal, Form, Input, Select, InputNumber, Switch, Space, message, Tag, Divider, Radio, Progress } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, ExclamationCircleOutlined, UploadOutlined, ImportOutlined, LinkOutlined, FileOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { rulesApi, egressApi } from '../services/api'

interface Rule {
  id: string
  name: string
  priority: number
  enabled: boolean
  list_type: string
  domains: string[]
  cidrs: string[]
  ports: number[]
  action: string
  egress_id: string
  description: string
  source?: string
  domain_count?: number
}

const listTypeOptions = [
  { value: 'none', label: '普通规则' },
  { value: 'whitelist', label: '白名单' },
  { value: 'blacklist', label: '黑名单' },
]

function Rules() {
  const [modalOpen, setModalOpen] = useState(false)
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<Rule | null>(null)
  const [importSource, setImportSource] = useState<'url' | 'file'>('url')
  const [importProgress, setImportProgress] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [importForm] = Form.useForm()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const queryClient = useQueryClient()

  const { data: rules } = useQuery({
    queryKey: ['rules'],
    queryFn: rulesApi.list,
  })

  const { data: egressPolicies } = useQuery({
    queryKey: ['egress'],
    queryFn: egressApi.list,
  })

  const createMutation = useMutation({
    mutationFn: rulesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      message.success('规则已创建')
      setModalOpen(false)
    },
    onError: (error: any) => {
      message.error('创建规则失败: ' + (error.response?.data?.error || error.message || '未知错误'))
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Rule }) => rulesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      message.success('规则已更新')
      setModalOpen(false)
    },
    onError: (error: any) => {
      message.error('更新规则失败: ' + (error.response?.data?.error || error.message || '未知错误'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: rulesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      message.success('规则已删除')
    },
  })

  const importUrlMutation = useMutation({
    mutationFn: rulesApi.importFromUrl,
    onMutate: () => {
      setImportProgress(0)
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      message.success(`导入成功，共 ${data.domain_count} 个域名`)
      setImportModalOpen(false)
      setImportProgress(null)
      importForm.resetFields()
    },
    onError: (error: any) => {
      message.error('导入失败: ' + (error.response?.data?.error || error.message || '未知错误'))
      setImportProgress(null)
    },
  })

  const importFileMutation = useMutation({
    mutationFn: (formData: FormData) => rulesApi.importFromFile(formData),
    onMutate: () => {
      setImportProgress(0)
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      message.success(`导入成功，共 ${data.domain_count} 个域名`)
      setImportModalOpen(false)
      setImportProgress(null)
      importForm.resetFields()
      setSelectedFile(null)
    },
    onError: (error: any) => {
      message.error('导入失败: ' + (error.response?.data?.error || error.message || '未知错误'))
      setImportProgress(null)
    },
  })

  const handleAdd = () => {
    setEditingRule(null)
    form.resetFields()
    form.setFieldsValue({ action: 'forward', enabled: true, priority: 50, list_type: 'none' })
    setModalOpen(true)
  }

  const handleEdit = (record: Rule) => {
    setEditingRule(record)
    form.setFieldsValue({
      name: record.name,
      priority: record.priority,
      enabled: record.enabled,
      list_type: record.list_type || 'none',
      domains: record.domains?.join('\n'),
      cidrs: record.cidrs?.join('\n'),
      ports: record.ports?.join(', '),
      action: record.action,
      egress_id: record.egress_id,
      description: record.description,
    })
    setModalOpen(true)
  }

  const handleSubmit = () => {
    form.validateFields().then(values => {
      const rule: Rule = {
        id: editingRule?.id || `rule-${Date.now()}`,
        name: values.name,
        priority: values.priority,
        enabled: values.enabled,
        list_type: values.list_type,
        domains: values.domains ? values.domains.split('\n').filter((s: string) => s.trim()) : [],
        cidrs: values.cidrs ? values.cidrs.split('\n').filter((s: string) => s.trim()) : [],
        ports: values.ports ? values.ports.split(',').map((p: string) => parseInt(p.trim())).filter((n: number) => !isNaN(n)) : [],
        action: values.action,
        egress_id: values.egress_id,
        description: values.description,
      }

      if (editingRule) {
        updateMutation.mutate({ id: editingRule.id, data: rule })
      } else {
        createMutation.mutate(rule)
      }
    })
  }

  const handleDelete = (record: Rule) => {
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除规则 "${record.name}" 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk() {
        deleteMutation.mutate(record.id)
      },
    })
  }

  const handleImport = () => {
    setImportSource('url')
    importForm.resetFields()
    importForm.setFieldsValue({ priority: 50, enabled: true, source_type: 'url' })
    setSelectedFile(null)
    setImportModalOpen(true)
  }

  const handleImportSubmit = () => {
    importForm.validateFields().then(values => {
      if (importSource === 'url') {
        importUrlMutation.mutate({
          name: values.name || '导入的域名列表',
          url: values.url,
          egress_id: values.egress_id,
          priority: values.priority || 100,
          enabled: values.enabled !== false,
        })
      } else {
        if (!selectedFile) {
          message.error('请选择文件')
          return
        }
        const formData = new FormData()
        formData.append('file', selectedFile)
        formData.append('name', values.name || '导入的域名列表')
        formData.append('egress_id', values.egress_id || '')
        formData.append('priority', String(values.priority || 100))
        formData.append('enabled', values.enabled !== false ? 'true' : 'false')
        importFileMutation.mutate(formData)
      }
    })
  }

  const getListTypeTag = (listType: string) => {
    switch (listType) {
      case 'whitelist':
        return <Tag color="green">白名单</Tag>
      case 'blacklist':
        return <Tag color="red">黑名单</Tag>
      default:
        return <Tag>普通</Tag>
    }
  }

  const columns = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 70,
    },
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      width: 120,
    },
    {
      title: '类型',
      dataIndex: 'list_type',
      key: 'list_type',
      width: 80,
      render: (listType: string) => getListTypeTag(listType),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 60,
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'red'}>{enabled ? '开' : '关'}</Tag>
      ),
    },
    {
      title: '域名',
      dataIndex: 'domains',
      key: 'domains',
      render: (domains: string[], record: Rule) => {
        if (record.domain_count && record.domain_count > 0) {
          return <Tag color="blue">{record.domain_count.toLocaleString()} 个域名</Tag>
        }
        return domains?.slice(0, 2).join(', ') + (domains?.length > 2 ? '...' : '') || '-'
      },
    },
    {
      title: 'IP/CIDR',
      dataIndex: 'cidrs',
      key: 'cidrs',
      render: (cidrs: string[]) => cidrs?.slice(0, 2).join(', ') + (cidrs?.length > 2 ? '...' : '') || '-',
    },
    {
      title: '动作',
      dataIndex: 'action',
      key: 'action',
      width: 80,
      render: (action: string) => (
        <Tag color={action === 'reject' ? 'red' : 'blue'}>
          {action === 'reject' ? '拒绝' : '转发'}
        </Tag>
      ),
    },
    {
      title: '出口策略',
      dataIndex: 'egress_id',
      key: 'egress_id',
      width: 100,
      render: (egressId: string) => {
        const policy = (egressPolicies || []).find((p: any) => p.id === egressId)
        return policy?.name || egressId || '-'
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render: (_: any, record: Rule) => (
        <Space>
          <Button icon={<EditOutlined />} size="small" onClick={() => handleEdit(record)} />
          <Button
            icon={<DeleteOutlined />}
            size="small"
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
        title="路由规则管理"
        extra={
          <Space>
            <Button icon={<ImportOutlined />} onClick={handleImport}>导入列表</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>添加规则</Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={rules || []}
          rowKey="id"
          size="small"
          pagination={false}
        />
      </Card>

      <Modal
        title={editingRule ? '编辑规则' : '添加规则'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={650}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例如：Google服务走代理" />
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            rules={[
              { required: true, message: '请输入优先级' },
              {
                validator: (_, value) => {
                  if (value === undefined || value === null || value === '') {
                    return Promise.resolve()
                  }
                  if (value < 0 || value > 100) {
                    return Promise.reject('优先级必须在 0-100 之间')
                  }
                  // 检查优先级重复
                  const isDuplicate = (rules || []).some((r: Rule) =>
                    r.priority === value && r.id !== editingRule?.id
                  )
                  if (isDuplicate) {
                    return Promise.reject('优先级已被其他规则使用')
                  }
                  return Promise.resolve()
                }
              }
            ]}
          >
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Divider orientation="left">名单类型</Divider>

          <Form.Item name="list_type" label="类型" extra={
            <div style={{ marginTop: 4 }}>
              <div>• <b>普通规则</b>：按优先级匹配，匹配后执行动作</div>
              <div>• <b>白名单</b>：仅允许匹配的地址通过</div>
              <div>• <b>黑名单</b>：阻止匹配的地址访问</div>
            </div>
          }>
            <Select options={listTypeOptions} />
          </Form.Item>

          <Divider orientation="left">匹配条件</Divider>

          <Form.Item name="domains" label="域名 (每行一个，支持 *.example.com)">
            <Input.TextArea rows={3} placeholder="*.google.com&#10;*.youtube.com" />
          </Form.Item>

          <Form.Item name="cidrs" label="IP/CIDR (每行一个)">
            <Input.TextArea rows={3} placeholder="192.168.0.0/16&#10;10.0.0.0/8" />
          </Form.Item>

          <Form.Item name="ports" label="端口 (逗号分隔)">
            <Input placeholder="80, 443, 8080" />
          </Form.Item>

          <Divider orientation="left">动作设置</Divider>

          <Form.Item name="action" label="动作" rules={[{ required: true, message: '请选择动作' }]}>
            <Select>
              <Select.Option value="forward">转发到出口策略</Select.Option>
              <Select.Option value="reject">拒绝连接</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item noStyle shouldUpdate>
            {({ getFieldValue }) =>
              getFieldValue('action') === 'forward' && (
                <Form.Item name="egress_id" label="出口策略" rules={[{ required: true, message: '请选择出口策略' }]}>
                  <Select placeholder="选择出口策略">
                    {(egressPolicies || []).map((policy: any) => (
                      <Select.Option key={policy.id} value={policy.id}>
                        {policy.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              )
            }
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="导入域名列表"
        open={importModalOpen}
        onOk={handleImportSubmit}
        onCancel={() => {
          setImportModalOpen(false)
          setImportProgress(null)
        }}
        confirmLoading={importUrlMutation.isPending || importFileMutation.isPending}
        width={550}
      >
        <Form form={importForm} layout="vertical">
          <Form.Item name="name" label="规则名称">
            <Input placeholder="例如：代理域名列表" />
          </Form.Item>

          <Form.Item label="导入来源">
            <Radio.Group value={importSource} onChange={(e) => setImportSource(e.target.value)}>
              <Radio.Button value="url"><LinkOutlined /> 远程URL</Radio.Button>
              <Radio.Button value="file"><FileOutlined /> 本地文件</Radio.Button>
            </Radio.Group>
          </Form.Item>

          {importSource === 'url' ? (
            <Form.Item name="url" label="URL" rules={[{ required: true, message: '请输入URL' }]}>
              <Input placeholder="https://example.com/domain-list.txt" />
            </Form.Item>
          ) : (
            <Form.Item label="文件">
              <input
                type="file"
                ref={fileInputRef}
                style={{ display: 'none' }}
                onChange={(e) => {
                  if (e.target.files?.[0]) {
                    setSelectedFile(e.target.files[0])
                  }
                }}
                accept=".txt"
              />
              <Button
                icon={<UploadOutlined />}
                onClick={() => fileInputRef.current?.click()}
              >
                选择文件
              </Button>
              {selectedFile && (
                <span style={{ marginLeft: 8 }}>{selectedFile.name}</span>
              )}
            </Form.Item>
          )}

          <Form.Item name="egress_id" label="出口策略">
            <Select placeholder="选择出口策略" allowClear>
              {(egressPolicies || []).map((policy: any) => (
                <Select.Option key={policy.id} value={policy.id}>
                  {policy.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="priority"
            label="优先级"
            rules={[
              { required: true, message: '请输入优先级' },
              {
                validator: (_, value) => {
                  if (value === undefined || value === null || value === '') {
                    return Promise.resolve()
                  }
                  if (value < 0 || value > 100) {
                    return Promise.reject('优先级必须在 0-100 之间')
                  }
                  // 检查优先级重复
                  const isDuplicate = (rules || []).some((r: Rule) =>
                    r.priority === value
                  )
                  if (isDuplicate) {
                    return Promise.reject('优先级已被其他规则使用')
                  }
                  return Promise.resolve()
                }
              }
            ]}
          >
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>

        {importProgress !== null && (
          <Progress percent={importProgress} status="active" />
        )}
      </Modal>
    </div>
  )
}

export default Rules
