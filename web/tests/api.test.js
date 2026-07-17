import test from 'node:test'
import assert from 'node:assert/strict'

import { createApiClient } from '../src/api.js'

function response(data, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return { code: 0, message: 'ok', data }
    },
  }
}

test('defaults to /api/v1 and adds an idempotency key to writes', async () => {
  const requests = []
  const client = createApiClient({
    fetchImpl: async (url, init) => {
      requests.push({ url, init })
      return response({ id: 'AP-1', status: '已确认' })
    },
  })

  const appointment = await client.checkinAppointment('AP-1')

  assert.equal(appointment.id, 'AP-1')
  assert.equal(requests[0].url, '/api/v1/appointments/AP-1/checkin')
  assert.equal(requests[0].init.method, 'POST')
  assert.match(requests[0].init.headers['Idempotency-Key'], /^cf-/)
})

test('uses a configured API origin without duplicating the API path', async () => {
  const requests = []
  const client = createApiClient({
    baseUrl: 'http://localhost:8080/api/v1/',
    fetchImpl: async (url) => {
      requests.push(url)
      return response({ list: [], total: 0 })
    },
  })

  await client.listAppointments({ page: 1, pageSize: 20 })

  assert.equal(requests[0], 'http://localhost:8080/api/v1/appointments?page=1&pageSize=20')
})

test('rejects non-zero API envelopes so callers can keep demo data', async () => {
  const client = createApiClient({
    fetchImpl: async () => ({
      ok: false,
      status: 409,
      async json() {
        return { code: 409, message: '状态不可推进', data: null }
      },
    }),
  })

  await assert.rejects(() => client.updateAppointmentStatus('AP-1', '候诊中'), /状态不可推进/)
})

test('exposes mobile lifecycle and follow-up operations through the same client', async () => {
  const paths = []
  const client = createApiClient({
    fetchImpl: async (url) => {
      paths.push(url)
      return response({ id: 'ok' })
    },
  })

  await client.createAppointment({ patient: '演示客户', department: '全科门诊' })
  await client.checkinAppointment('AP-1')
  await client.updateAppointmentStatus('AP-1', '候诊中')
  await client.updateAppointmentStatus('AP-1', '处理中')
  await client.updateAppointmentStatus('AP-1', '已完成')
  await client.completeFollowup('FW-1')

  assert.deepEqual(paths, [
    '/api/v1/appointments',
    '/api/v1/appointments/AP-1/checkin',
    '/api/v1/appointments/AP-1/status',
    '/api/v1/appointments/AP-1/status',
    '/api/v1/appointments/AP-1/status',
    '/api/v1/followups/FW-1/complete',
  ])
})

test('exposes invoice billing lifecycle with idempotent writes', async () => {
  const requests = []
  const client = createApiClient({
    fetchImpl: async (url, init) => {
      requests.push({ url, init })
      return response({ id: 'INV-1', status: '已开具', paidCents: 0 })
    },
  })
  await client.listInvoices({ page: 1, pageSize: 20, status: '已开具' })
  await client.getInvoice('INV-1')
  await client.createInvoice({ customerName: '星河科技', amountCents: 1000 })
  await client.updateInvoiceStatus('INV-1', '已开具')
  await client.addInvoicePayment('INV-1', { amountCents: 1000, method: '银行转账' })
  await client.reconcileInvoice('INV-1')
  assert.deepEqual(requests.map(({ url }) => url), [
    '/api/v1/invoices?page=1&pageSize=20&status=%E5%B7%B2%E5%BC%80%E5%85%B7',
    '/api/v1/invoices/INV-1',
    '/api/v1/invoices',
    '/api/v1/invoices/INV-1/status',
    '/api/v1/invoices/INV-1/payments',
    '/api/v1/invoices/INV-1/reconcile',
  ])
  assert.equal(requests[4].init.headers['Idempotency-Key'].startsWith('cf-'), true)
})
