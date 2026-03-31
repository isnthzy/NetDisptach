import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
})

export const nicsApi = {
  list: () => api.get('/nics').then(r => r.data),
  get: (name: string) => api.get(`/nics/${name}`).then(r => r.data),
}

export const egressApi = {
  list: () => api.get('/egress').then(r => r.data),
  create: (data: any) => api.post('/egress', data).then(r => r.data),
  update: (id: string, data: any) => api.put(`/egress/${id}`, data).then(r => r.data),
  delete: (id: string) => api.delete(`/egress/${id}`),
  test: (id: string) => api.post(`/egress/${id}/test`).then(r => r.data),
}

export const rulesApi = {
  list: () => api.get('/rules').then(r => r.data),
  create: (data: any) => api.post('/rules', data).then(r => r.data),
  update: (id: string, data: any) => api.put(`/rules/${id}`, data).then(r => r.data),
  delete: (id: string) => api.delete(`/rules/${id}`),
}

export const connectionsApi = {
  list: () => api.get('/connections').then(r => r.data),
  get: (id: string) => api.get(`/connections/${id}`).then(r => r.data),
  close: (id: string) => api.delete(`/connections/${id}`),
}

export const statsApi = {
  getOverview: () => api.get('/stats/overview').then(r => r.data),
  getTraffic: () => api.get('/stats/traffic').then(r => r.data),
  getConnections: () => api.get('/connections').then(r => r.data),
  getHistory: () => api.get('/stats/history').then(r => r.data),
  getRecentConnections: () => api.get('/connections/recent').then(r => r.data),
}

export const configApi = {
  get: () => api.get('/config').then(r => r.data),
  update: (data: any) => api.put('/config', data).then(r => r.data),
}

export const statusApi = {
  get: () => api.get('/status').then(r => r.data),
}

export const systemApi = {
  getInfo: () => api.get('/system/info').then(r => r.data),
  health: () => api.get('/health').then(r => r.data),
}
