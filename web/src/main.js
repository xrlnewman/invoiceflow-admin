import './styles.css'
import { createApiClient } from './api.js'

const api = createApiClient()

const demoAppointments = [
  { id: 'INV-0716-082', patient: '杭州星河科技', department: '增值税专票', doctor: '林然 · 财务专员', scheduledAt: '2026-07-16T09:30:00+08:00', status: '候诊中' },
  { id: 'INV-0716-081', patient: '苏州云杉供应链', department: '电子普票', doctor: '沈宁 · 结算专员', scheduledAt: '2026-07-16T09:45:00+08:00', status: '已确认' },
  { id: 'INV-0716-080', patient: '上海岸线设计', department: '服务费发票', doctor: '赵然 · 财务专员', scheduledAt: '2026-07-16T10:00:00+08:00', status: '已完成' },
  { id: 'INV-0716-079', patient: '南京微光传媒', department: '广告费发票', doctor: '林然 · 财务专员', scheduledAt: '2026-07-16T10:15:00+08:00', status: '待确认' },
  { id: 'INV-0716-078', patient: '成都山海咨询', department: '咨询费发票', doctor: '周宁 · 结算专员', scheduledAt: '2026-07-16T10:30:00+08:00', status: '待确认' },
]

const demoFollowups = [
  { id: 'TASK-0716-012', patient: '杭州星河科技', summary: '核对销项税额与开票信息', dueAt: '今天 16:00', status: '待完成' },
  { id: 'TASK-0716-011', patient: '南京微光传媒', summary: '跟进客户回款凭证', dueAt: '今天 17:30', status: '待完成' },
  { id: 'TASK-0716-010', patient: '苏州云杉供应链', summary: '补齐合同与发票附件', dueAt: '明天 09:30', status: '待完成' },
  { id: 'TASK-0715-009', patient: '上海岸线设计', summary: '完成月度发票归档', dueAt: '已完成', status: '已完成' },
]

const demoInvoices = [
  { id: 'INV-202607-082', customerName: '杭州星河科技', amountCents: 128000, paidCents: 128000, currency: 'CNY', status: '部分回款', dueDate: '2026-07-20', items: [{ description: '企业软件订阅', quantity: 1, amountCents: 128000 }], payments: [{ id: 'PAY-082-01', amountCents: 128000, method: '银行转账', reference: 'HZ20260716001', receivedAt: '2026-07-16T09:30:00Z' }], events: [{ id: 'EV-082-1', fromStatus: '草稿', toStatus: '待审核', type: 'status', actor: '林然', createdAt: '2026-07-15T08:00:00Z' }, { id: 'EV-082-2', fromStatus: '待审核', toStatus: '已开具', type: 'status', actor: '沈宁', createdAt: '2026-07-15T09:20:00Z' }] },
  { id: 'INV-202607-081', customerName: '苏州云杉供应链', amountCents: 86000, paidCents: 0, currency: 'CNY', status: '已开具', dueDate: '2026-07-25', items: [{ description: '供应链系统实施服务', quantity: 1, amountCents: 86000 }], payments: [], events: [{ id: 'EV-081-1', fromStatus: '草稿', toStatus: '待审核', type: 'status', actor: '林然', createdAt: '2026-07-15T08:10:00Z' }] },
  { id: 'INV-202607-080', customerName: '上海岸线设计', amountCents: 42000, paidCents: 42000, currency: 'CNY', status: '已核销', dueDate: '2026-07-18', items: [{ description: '品牌设计服务', quantity: 1, amountCents: 42000 }], payments: [{ id: 'PAY-080-01', amountCents: 42000, method: '支付宝', reference: 'ALIPAY-080', receivedAt: '2026-07-14T10:00:00Z' }], events: [{ id: 'EV-080-1', fromStatus: '已开具', toStatus: '部分回款', type: 'payment', actor: '周宁', createdAt: '2026-07-14T10:00:00Z' }, { id: 'EV-080-2', fromStatus: '部分回款', toStatus: '已核销', type: 'reconcile', actor: '周宁', createdAt: '2026-07-14T10:05:00Z' }] },
  { id: 'INV-202607-079', customerName: '南京微光传媒', amountCents: 196000, paidCents: 0, currency: 'CNY', status: '待审核', dueDate: '2026-07-30', items: [{ description: '广告投放服务', quantity: 2, amountCents: 196000 }], payments: [], events: [] },
]

const demoDashboard = { todayAppointments: 86, averageWaitMinutes: 12, completed: 58, checkedIn: 42, pendingFollowups: 12 }
const statusColors = { 待确认: 'coral', 已确认: 'indigo', 候诊中: 'amber', 处理中: 'green', 已完成: 'green', 已取消: 'gray', 草稿: 'gray', 待审核: 'amber', 已开具: 'indigo', 部分回款: 'coral', 已核销: 'green', 已归档: 'gray' }
const nav = [
  ['overview', '运营总览', '⌂'],
  ['queue', '发票队列', '▤'],
  ['billing', '收款工作台', '¥'],
  ['doctors', '财务专员排班', '◉'],
  ['patients', '客户档案', '♧'],
  ['followups', '跟进任务', '✓'],
  ['mobile', '移动端体验', '⌁'],
]

let appointments = demoAppointments.map((item) => ({ ...item }))
let followupTasks = demoFollowups.map((item) => ({ ...item }))
let invoices = demoInvoices.map((item) => ({ ...item }))
let selectedInvoice = null
let invoiceFilter = ''
let dashboard = { ...demoDashboard }
let page = 'overview'
let toast = ''
let toastTimer
let dataSource = '演示数据'
let isSyncing = false

function displayCopy(root) {
  const rules = [['候诊', '处理'], ['回访', '跟进'], ['健康', '账款'], ['临床', '财务'], ['科室', '发票类型'], ['人次', '笔'], ['位客户', '张发票'], ['诊断', '真实财务'], ['林负责人', '林然 · 财务专员'], ['沈负责人', '沈宁 · 结算专员'], ['赵负责人', '赵然 · 财务专员'], ['周负责人', '周宁 · 结算专员'], ['陈负责人', '陈敏 · 财务专员'], ['王负责人', '王可 · 结算专员'], ['全科门诊', '增值税专票'], ['皮肤科', '电子普票'], ['康复理疗', '服务费发票'], ['营养咨询', '咨询费发票'], ['就诊', '开票'], ['CF-', 'CUS-']]
  const walker = document.createTreeWalker(root, 4)
  while (walker.nextNode()) rules.forEach(([from, to]) => { walker.currentNode.nodeValue = walker.currentNode.nodeValue.replaceAll(from, to) })
}

function timeLabel(value) {
  const match = String(value ?? '').match(/T(\d{2}:\d{2})/)
  return match?.[1] || String(value ?? '').slice(0, 5) || '--:--'
}

function normalizeAppointment(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名客户',
    department: item.department || '待分诊',
    doctor: item.doctor || '待安排',
    scheduledAt: item.scheduledAt || '',
    status: item.status || '待确认',
  }
}

function normalizeFollowup(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名客户',
    summary: item.summary || '健康回访任务',
    dueAt: item.dueAt || '--',
    status: item.status || '待完成',
  }
}

function showToast(message) {
  toast = message
  render()
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast = ''
    render()
  }, 2200)
}

function appointmentAction(appointment) {
  if (appointment.status === '待确认') return `<button class="text-action" data-action="checkin" data-appointment-id="${appointment.id}">确认</button>`
  if (appointment.status === '已确认') return `<button class="text-action" data-action="status" data-next-status="候诊中" data-appointment-id="${appointment.id}">进入候诊</button>`
  if (appointment.status === '候诊中') return `<button class="text-action" data-action="status" data-next-status="处理中" data-appointment-id="${appointment.id}">开始处理</button>`
  if (appointment.status === '处理中') return `<button class="text-action" data-action="status" data-next-status="已完成" data-appointment-id="${appointment.id}">完成处理</button>`
  return '<button class="text-action" data-toast="该发票已完成，无需重复操作">查看详情</button>'
}

function header(title) {
  return `<header><span>工作台　/　<strong>${title}</strong></span><span class="header-tools"><span>2026 年 7 月 16 日</span><span class="data-source ${dataSource === 'API 数据' ? 'remote' : ''}">● ${isSyncing ? '同步中' : dataSource}</span><button class="refresh" data-refresh ${isSyncing ? 'disabled' : ''}>↻ 刷新</button></span></header>`
}

function render() {
  const title = nav.find((item) => item[0] === page)?.[1] || '运营总览'
  const content = page === 'overview' ? overview() : page === 'queue' ? queue() : page === 'billing' ? billing() : page === 'doctors' ? doctors() : page === 'patients' ? patients() : page === 'followups' ? followups() : mobileView()
  document.querySelector('#app').innerHTML = `<div class="shell"><aside><div class="brand"><span>¥</span><div><strong>InvoiceFlow</strong><small>发票收款运营中心</small></div></div><div class="clinic">● 上海静安联合财务中心　⌄</div><p class="caption">临床运营</p><nav>${nav.map((item) => `<button class="${page === item[0] ? 'active' : ''}" data-page="${item[0]}"><i>${item[2]}</i>${item[1]}${item[0] === 'queue' ? '<em>8</em>' : ''}</button>`).join('')}</nav><div class="user"><b>许</b><span><strong>许汝林</strong><small>运营管理员</small></span></div></aside><main>${header(title)}<section class="heading"><div><p>THURSDAY, JUL 16 · INVOICEFLOW</p><h1>${title} <i>✦</i></h1><label>让每一次发票，都有被照顾的下一步。</label></div><button class="primary" data-action="create-appointment">＋ 新建发票</button></section>${content}<footer>InvoiceFlow 发票收款运营 · 免费开源 · 演示数据不含诊断与真实客户信息</footer><div class="toast" ${toast ? '' : 'hidden'}>${toast}</div></main></div>`
  const root = document.querySelector('#app')
  displayCopy(root)
  const filter = document.querySelector('[data-invoice-filter]')
  if (filter) { filter.value = invoiceFilter; filter.addEventListener('change', () => { invoiceFilter = filter.value; render() }) }
  bind()
}

function overview() {
  return `<section class="metrics"><article class="metric dark"><span>今日发票</span><strong>${dashboard.todayAppointments}</strong><small>↗ 较昨日 +14.6%</small></article><article class="metric"><span>平均候诊</span><strong>${dashboard.averageWaitMinutes}<small> 分钟</small></strong><small class="good">较上周 -3 分钟</small></article><article class="metric"><span>今日完成</span><strong>${dashboard.completed}<small> 人次</small></strong><div class="progress"><i style="width:68%"></i></div></article><article class="metric warm"><span>待回访</span><strong>${dashboard.pendingFollowups}<small> 条</small></strong><small class="coral">今日需完成</small></article></section><section class="grid"><article class="panel calendar"><div class="panel-head"><div><h2>今日发票队列</h2><p>7 月 16 日 · 周四 · 共 ${dashboard.todayAppointments} 位客户</p></div><button class="link" data-page="queue">查看队列 →</button></div><div class="timeline">${appointments.slice(0, 4).map((appointment) => `<div class="time-row"><span>${timeLabel(appointment.scheduledAt)}</span><i class="time-dot ${statusColors[appointment.status] || 'indigo'}"></i><div><strong>${appointment.patient}</strong><small>${appointment.department} · ${appointment.status}</small></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b></div>`).join('')}</div></article><article class="panel"><div class="panel-head"><div><h2>科室处理负载</h2><p>当前时段排班利用率</p></div><button class="link" data-page="doctors">排班管理 →</button></div><div class="load-list">${[['全科门诊', '32 / 40', '80%', 'indigo'], ['皮肤科', '18 / 24', '75%', 'coral'], ['康复理疗', '12 / 18', '67%', 'green'], ['营养咨询', '8 / 12', '66%', 'amber']].map((item) => `<div class="load"><div><strong>${item[0]}</strong><span>${item[1]}</span></div><div class="load-bar"><i class="${item[3]}" style="width:${item[2]}"></i></div><b>${item[2]}</b></div>`).join('')}</div></article></section><section class="grid lower"><article class="panel"><div class="panel-head"><div><h2>回访完成趋势</h2><p>近 7 日任务完成率</p></div><span class="legend">本周平均 84%</span></div><div class="spark"><i style="height:38%"></i><i style="height:58%"></i><i style="height:46%"></i><i style="height:74%"></i><i style="height:66%"></i><i style="height:88%"></i><i class="today" style="height:80%"></i></div><div class="days"><span>周五</span><span>周六</span><span>周日</span><span>周一</span><span>周二</span><span>周三</span><span>今天</span></div></article><article class="panel tasks"><div class="panel-head"><div><h2>待办提醒</h2><p>需要运营人员跟进的事项</p></div></div><div class="task"><span class="task-icon coral">!</span><div><strong>3 位客户需要改约</strong><small>发票队列 · 10 分钟前</small></div><button data-page="queue">处理</button></div><div class="task"><span class="task-icon amber">✓</span><div><strong>${dashboard.pendingFollowups} 条回访今日到期</strong><small>健康回访 · 32 分钟前</small></div><button data-page="followups">查看</button></div></article></section>`
}

function queue() {
  return `<section class="panel full"><div class="panel-head"><div><h2>发票队列</h2><p>${dataSource === 'API 数据' ? 'API 实时发票' : '20 条演示发票'} · 支持确认、候诊、处理和完成</p></div><span class="chip">今天　⌄</span></div><div class="table"><div class="th"><span>发票编号 / 客户</span><span>科室</span><span>时间</span><span>状态</span><span>操作</span></div>${appointments.concat(dataSource === 'API 数据' ? [] : appointments.slice(0, 3)).map((appointment) => `<div class="tr"><span><strong>${appointment.id}</strong><small>${appointment.patient}</small></span><span>${appointment.department}</span><span>${timeLabel(appointment.scheduledAt)}</span><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b><span>${appointmentAction(appointment)}</span></div>`).join('')}</div></section>`
}

function money(cents) { return `¥${(Number(cents || 0) / 100).toLocaleString('zh-CN', { minimumFractionDigits: 2 })}` }

function billing() {
  const current = selectedInvoice
  const visible = invoiceFilter ? invoices.filter((item) => item.status === invoiceFilter) : invoices
  return `<section class="grid billing-grid"><article class="panel full"><div class="panel-head"><div><h2>收款工作台</h2><p>${dataSource === 'API 数据' ? 'API 实时发票' : '4 条演示发票'} · 开票、回款、核销一条线闭环</p></div><select class="chip" data-invoice-filter><option value="">全部状态</option><option value="待审核">待审核</option><option value="已开具">已开具</option><option value="部分回款">部分回款</option><option value="已核销">已核销</option></select></div><div class="table"><div class="th"><span>发票编号 / 客户</span><span>应收金额</span><span>已回款</span><span>状态</span><span>操作</span></div>${visible.map((invoice) => `<div class="tr"><span><strong>${invoice.id}</strong><small>${invoice.customerName}</small></span><span>${money(invoice.amountCents)}</span><span>${money(invoice.paidCents)}</span><b class="status ${statusColors[invoice.status] || 'indigo'}">${invoice.status}</b><button class="text-action" data-action="open-invoice" data-invoice-id="${invoice.id}">查看详情</button></div>`).join('') || '<div class="muted">暂无匹配发票</div>'}</div></article>${current ? `<article class="panel invoice-detail"><div class="panel-head"><div><h2>${current.customerName}</h2><p>${current.id} · 到期 ${current.dueDate || '--'}</p></div><b class="status ${statusColors[current.status] || 'indigo'}">${current.status}</b></div><div class="detail-amount"><span>应收 ${money(current.amountCents)}</span><strong>已回款 ${money(current.paidCents)}</strong></div><div class="detail-actions"><button class="primary small" data-action="invoice-payment" data-invoice-id="${current.id}">登记回款</button><button class="secondary small" data-action="invoice-reconcile" data-invoice-id="${current.id}">核销发票</button></div><h3>发票时间线</h3><div class="invoice-events">${(current.events || []).map((event) => `<div class="event"><i></i><div><strong>${event.toStatus || event.type}</strong><small>${event.actor || '系统'} · ${event.createdAt || '--'}</small></div></div>`).join('') || '<p class="muted">暂无状态事件</p>'}</div></article>` : '<article class="panel invoice-detail empty"><h2>选择一张发票</h2><p>查看明细、事件时间线并登记回款或完成核销。</p></article>'}</section>`
}

function doctors() {
  return `<section class="panel full"><div class="panel-head"><div><h2>负责人排班</h2><p>8 位负责人 · 今日 42 个可发票时段</p></div><button class="primary small" data-toast="排班编辑器已打开">编辑排班</button></div><div class="doctor-grid">${[['林负责人', '全科门诊', '32 号候诊', 'indigo'], ['沈负责人', '皮肤科', '18 号候诊', 'coral'], ['赵负责人', '康复理疗', '处理中', 'green'], ['周负责人', '营养咨询', '8 号候诊', 'amber'], ['陈负责人', '全科门诊', '午间休息', 'gray'], ['王负责人', '心理咨询', '6 号候诊', 'indigo']].map((doctor) => `<article><div class="doctor-avatar ${doctor[3]}">${doctor[0][0]}</div><div><strong>${doctor[0]}</strong><small>${doctor[1]}</small></div><span>${doctor[2]}</span><div class="schedule-line"><i style="width:78%"></i></div></article>`).join('')}</div></section>`
}

function patients() {
  return `<section class="panel full"><div class="panel-head"><div><h2>客户档案</h2><p>30 条虚构档案 · 仅用于界面演示</p></div><button class="link" data-toast="导出任务已创建">导出列表 ↓</button></div><div class="table"><div class="th"><span>客户 / 编号</span><span>最近科室</span><span>最近就诊</span><span>回访状态</span><span>操作</span></div>${[['林晓雨', 'CF-2038', '全科门诊', '07/16', '待回访'], ['沈明远', 'CF-2037', '皮肤科', '07/15', '进行中'], ['赵思涵', 'CF-2036', '康复理疗', '07/14', '已完成'], ['周子昂', 'CF-2035', '全科门诊', '07/13', '待回访'], ['许安然', 'CF-2034', '营养咨询', '07/12', '已完成']].map((patient) => `<div class="tr"><span><strong>${patient[0]}</strong><small>${patient[1]}</small></span><span>${patient[2]}</span><span>${patient[3]}</span><b class="status ${patient[4] === '已完成' ? 'green' : 'coral'}">${patient[4]}</b><button class="text-action" data-toast="${patient[0]} 档案已打开">查看档案</button></div>`).join('')}</div></section>`
}

function followups() {
  return `<section class="panel full"><div class="panel-head"><div><h2>回访任务</h2><p>${dataSource === 'API 数据' ? 'API 实时回访' : '12 条待跟进任务'} · 由负责人/护士确认后记录</p></div><span class="chip">全部任务　⌄</span></div><div class="follow-list">${followupTasks.map((item) => `<article><span class="task-icon ${item.status === '已完成' ? 'green' : 'coral'}">✓</span><div><strong>${item.id} · ${item.patient}</strong><p>${item.summary}</p><small>${item.dueAt} · ${dataSource === 'API 数据' ? 'API 数据' : '演示任务'}</small></div>${item.status === '已完成' ? '<button class="text-action" data-toast="该回访已经完成">查看</button>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成任务</button>`}</article>`).join('')}</div></section>`
}

function mobileView() {
  return `<section class="mobile-panel"><div class="mobile-panel__hero"><span>INVOICEFLOW MOBILE</span><h2>我的就诊与回访</h2><p>客户端可在同一套闭环 API 中完成确认、候诊、处理和回访确认。</p><button class="primary" data-action="create-appointment">＋ 创建演示发票</button></div><div class="mobile-list"><h3>今日发票</h3>${appointments.slice(0, 4).map((appointment) => `<article class="mobile-card"><div><small>${timeLabel(appointment.scheduledAt)} · ${appointment.department}</small><strong>${appointment.patient}</strong><span>${appointment.doctor} · ${appointment.status}</span></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b>${appointmentAction(appointment)}</article>`).join('')}</div><div class="mobile-list"><h3>我的回访</h3>${followupTasks.slice(0, 3).map((item) => `<article class="mobile-card"><div><small>${item.dueAt}</small><strong>${item.summary}</strong><span>${item.patient} · ${item.status}</span></div>${item.status === '已完成' ? '<b class="status green">已完成</b>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成回访</button>`}</article>`).join('')}</div></section>`
}

async function refreshFromApi({ quiet = false } = {}) {
  if (isSyncing) return
  isSyncing = true
  render()
  try {
    const [nextDashboard, nextAppointments, nextFollowups, nextInvoices] = await Promise.all([
      api.getDashboard(),
      api.listAppointments({ page: 1, pageSize: 20 }),
      api.listFollowups({ page: 1, pageSize: 20 }),
      api.listInvoices({ page: 1, pageSize: 20 }),
    ])
    dashboard = { ...demoDashboard, ...nextDashboard }
    appointments = (nextAppointments?.list || []).map(normalizeAppointment)
    followupTasks = (nextFollowups?.list || []).map(normalizeFollowup)
    invoices = (nextInvoices?.list || []).map((item) => ({ ...item, items: item.items || [], payments: item.payments || [], events: item.events || [] }))
    dataSource = 'API 数据'
    if (!quiet) toast = '已从 InvoiceFlow API 刷新数据'
  } catch (error) {
    dataSource = '演示数据'
    if (!quiet) toast = `API 暂不可用，继续使用演示数据：${error.message}`
  } finally {
    isSyncing = false
    render()
  }
}

async function openInvoice(button) {
  const id = button.dataset.invoiceId
  try {
    selectedInvoice = await api.getInvoice(id)
    dataSource = 'API 数据'
  } catch (error) {
    selectedInvoice = invoices.find((item) => item.id === id) || null
    showToast(`发票详情接口暂不可用：${error.message}`)
    return
  }
  render()
}

async function registerInvoicePayment(button) {
  const invoice = invoices.find((item) => item.id === button.dataset.invoiceId) || selectedInvoice
  if (!invoice) return
  const amount = Math.max(1, Number(invoice.amountCents || 0) - Number(invoice.paidCents || 0))
  try {
    await api.addInvoicePayment(invoice.id, { amountCents: amount, method: '银行转账', reference: `DEMO-${invoice.id}` })
    selectedInvoice = await api.getInvoice(invoice.id)
    invoices = invoices.map((item) => item.id === selectedInvoice.id ? selectedInvoice : item)
    dataSource = 'API 数据'
    showToast('回款已登记，等待核销')
    render()
  } catch (error) { showToast(`登记回款失败：${error.message}`) }
}

async function reconcileInvoice(button) {
  const id = button.dataset.invoiceId
  try {
    await api.reconcileInvoice(id, '财务人员')
    selectedInvoice = await api.getInvoice(id)
    invoices = invoices.map((item) => item.id === id ? selectedInvoice : item)
    dataSource = 'API 数据'
    showToast('发票已完成核销')
    render()
  } catch (error) { showToast(`核销失败：${error.message}`) }
}

function replaceAppointment(updated) {
  appointments = appointments.map((item) => item.id === updated.id ? normalizeAppointment(updated) : item)
}

async function advanceAppointment(button) {
  const id = button.dataset.appointmentId
  const appointment = appointments.find((item) => item.id === id)
  if (!appointment) return
  const nextStatus = button.dataset.nextStatus
  try {
    const updated = button.dataset.action === 'checkin'
      ? await api.checkinAppointment(id)
      : await api.updateAppointmentStatus(id, nextStatus, '运营人员')
    replaceAppointment(updated)
    dataSource = 'API 数据'
    showToast(`${appointment.patient} 已更新为${updated.status}`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示数据：${error.message}`)
  }
}

async function completeFollowup(button) {
  const id = button.dataset.followupId
  const task = followupTasks.find((item) => item.id === id)
  if (!task) return
  try {
    const updated = await api.completeFollowup(id)
    followupTasks = followupTasks.map((item) => item.id === id ? normalizeFollowup(updated) : item)
    dataSource = 'API 数据'
    showToast(`${task.patient} 的回访已完成`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示任务：${error.message}`)
  }
}

async function createAppointment() {
  try {
    const created = await api.createAppointment({ patient: '移动端演示客户', patientId: 'CUS-MOBILE-DEMO', department: '增值税专票', doctor: '林然 · 财务专员', scheduledAt: new Date().toISOString() })
    appointments = [normalizeAppointment(created), ...appointments]
    dataSource = 'API 数据'
    showToast('发票已创建，可继续在移动端完成确认')
  } catch (error) {
    dataSource = '演示数据'
    showToast(`API 暂不可用，保留演示发票：${error.message}`)
  }
}

function bind() {
  document.querySelectorAll('[data-page]').forEach((element) => element.addEventListener('click', () => {
    page = element.dataset.page
    render()
  }))
  document.querySelectorAll('[data-toast]').forEach((element) => element.addEventListener('click', () => showToast(element.dataset.toast)))
  document.querySelectorAll('[data-refresh]').forEach((element) => element.addEventListener('click', () => refreshFromApi()))
  document.querySelectorAll('[data-action]').forEach((element) => element.addEventListener('click', () => {
    if (element.dataset.action === 'checkin' || element.dataset.action === 'status') return advanceAppointment(element)
    if (element.dataset.action === 'complete-followup') return completeFollowup(element)
    if (element.dataset.action === 'open-invoice') return openInvoice(element)
    if (element.dataset.action === 'invoice-payment') return registerInvoicePayment(element)
    if (element.dataset.action === 'invoice-reconcile') return reconcileInvoice(element)
    if (element.dataset.action === 'create-appointment') return createAppointment()
    return undefined
  }))
}

render()
refreshFromApi({ quiet: true })
