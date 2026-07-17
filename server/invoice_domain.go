package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	InvoiceDraft         = "草稿"
	InvoicePendingReview = "待审核"
	InvoiceIssued        = "已开具"
	InvoicePartiallyPaid = "部分回款"
	InvoiceReconciled    = "已核销"
	InvoiceArchived      = "已归档"
)

var invoiceTransitions = map[string]map[string]bool{
	InvoiceDraft:         {InvoicePendingReview: true},
	InvoicePendingReview: {InvoiceIssued: true, InvoiceDraft: true},
	InvoiceIssued:        {InvoicePartiallyPaid: true, InvoiceReconciled: true},
	InvoicePartiallyPaid: {InvoiceReconciled: true},
	InvoiceReconciled:    {InvoiceArchived: true},
	InvoiceArchived:      {},
}

// Invoice is the receivable document tracked through the billing workflow.
type Invoice struct {
	ID           string         `json:"id"`
	CustomerID   string         `json:"customerId,omitempty"`
	CustomerName string         `json:"customerName"`
	TaxNumber    string         `json:"taxNumber,omitempty"`
	Currency     string         `json:"currency"`
	AmountCents  int64          `json:"amountCents"`
	PaidCents    int64          `json:"paidCents"`
	DueDate      string         `json:"dueDate,omitempty"`
	Status       string         `json:"status"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
	Items        []InvoiceItem  `json:"items,omitempty"`
	Payments     []Payment      `json:"payments,omitempty"`
	Events       []InvoiceEvent `json:"events,omitempty"`
}

type InvoiceItem struct {
	ID             string `json:"id"`
	InvoiceID      string `json:"invoiceId"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	AmountCents    int64  `json:"amountCents"`
}

type Payment struct {
	ID          string `json:"id"`
	InvoiceID   string `json:"invoiceId"`
	AmountCents int64  `json:"amountCents"`
	Method      string `json:"method"`
	Reference   string `json:"reference,omitempty"`
	ReceivedAt  string `json:"receivedAt"`
}

type InvoiceEvent struct {
	ID         string `json:"id"`
	InvoiceID  string `json:"invoiceId"`
	FromStatus string `json:"fromStatus,omitempty"`
	ToStatus   string `json:"toStatus"`
	Type       string `json:"type"`
	Actor      string `json:"actor"`
	Note       string `json:"note,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type InvoiceItemInput struct {
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
}

type CreateInvoiceInput struct {
	CustomerID   string             `json:"customerId"`
	CustomerName string             `json:"customerName"`
	TaxNumber    string             `json:"taxNumber"`
	Currency     string             `json:"currency"`
	AmountCents  int64              `json:"amountCents"`
	DueDate      string             `json:"dueDate"`
	Items        []InvoiceItemInput `json:"items"`
}

type AddInvoicePaymentInput struct {
	AmountCents int64  `json:"amountCents" binding:"required"`
	Method      string `json:"method" binding:"required"`
	Reference   string `json:"reference"`
	ReceivedAt  string `json:"receivedAt"`
}

type InvoiceStatusInput struct {
	Status string `json:"status" binding:"required"`
	Actor  string `json:"actor"`
}

type InvoiceStore interface {
	ListInvoices(context.Context, int, int, string) ([]Invoice, int, error)
	GetInvoice(context.Context, string) (Invoice, error)
	CreateInvoice(context.Context, Invoice) (Invoice, error)
	UpdateInvoiceStatus(context.Context, string, string, string) (Invoice, InvoiceEvent, error)
	AddInvoicePayment(context.Context, string, Payment) (Invoice, Payment, InvoiceEvent, error)
	ReconcileInvoice(context.Context, string, string) (Invoice, InvoiceEvent, error)
}

// NewInvoiceService builds the billing workflow service on either memory or MySQL storage.
func NewInvoiceService(store InvoiceStore, idem idempotencyStore) *InvoiceService {
	return &InvoiceService{store: store, idem: idem}
}

type InvoiceService struct {
	store InvoiceStore
	idem  idempotencyStore
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, input CreateInvoiceInput, key string) (Invoice, error) {
	if strings.TrimSpace(key) == "" {
		return Invoice{}, ErrMissingIdempotencyKey
	}
	if strings.TrimSpace(input.CustomerName) == "" && strings.TrimSpace(input.CustomerID) == "" {
		return Invoice{}, fmt.Errorf("%w: customer is required", ErrInvalidInput)
	}
	if input.AmountCents <= 0 {
		return Invoice{}, fmt.Errorf("%w: amountCents must be positive", ErrInvalidInput)
	}
	rk := "invoice:create:" + key
	if id, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, err
	} else if ok {
		return s.store.GetInvoice(ctx, id)
	}
	release, err := s.idem.Lock(ctx, "invoice:create-lock", 10*time.Second)
	if err != nil {
		return Invoice{}, err
	}
	defer release()
	if id, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, err
	} else if ok {
		return s.store.GetInvoice(ctx, id)
	}
	currency := input.Currency
	if currency == "" {
		currency = "CNY"
	}
	items := make([]InvoiceItem, 0, len(input.Items))
	for i, item := range input.Items {
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		if strings.TrimSpace(item.Description) == "" {
			return Invoice{}, fmt.Errorf("%w: item description is required", ErrInvalidInput)
		}
		amount := item.UnitPriceCents * int64(item.Quantity)
		items = append(items, InvoiceItem{ID: fmt.Sprintf("item-%d", i+1), Description: item.Description, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, AmountCents: amount})
	}
	invoice, err := s.store.CreateInvoice(ctx, Invoice{CustomerID: input.CustomerID, CustomerName: input.CustomerName, TaxNumber: input.TaxNumber, Currency: currency, AmountCents: input.AmountCents, DueDate: input.DueDate, Status: InvoiceDraft, Items: items})
	if err != nil {
		return Invoice{}, err
	}
	if err := s.idem.Set(ctx, rk, invoice.ID, 24*time.Hour); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (s *InvoiceService) UpdateInvoiceStatus(ctx context.Context, id, status, actor, key string) (Invoice, error) {
	if strings.TrimSpace(key) == "" {
		return Invoice{}, ErrMissingIdempotencyKey
	}
	status = strings.TrimSpace(status)
	if !validInvoiceStatus(status) {
		return Invoice{}, fmt.Errorf("%w: unknown invoice status", ErrInvalidInput)
	}
	rk := "invoice:status:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, err
	} else if ok {
		return s.store.GetInvoice(ctx, existing)
	}
	release, err := s.idem.Lock(ctx, "invoice:status-lock:"+id, 10*time.Second)
	if err != nil {
		return Invoice{}, err
	}
	defer release()
	if existing, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, err
	} else if ok {
		return s.store.GetInvoice(ctx, existing)
	}
	if actor == "" {
		actor = "财务人员"
	}
	invoice, _, err := s.store.UpdateInvoiceStatus(ctx, id, status, actor)
	if err != nil {
		return Invoice{}, err
	}
	if err := s.idem.Set(ctx, rk, invoice.ID, 24*time.Hour); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

func (s *InvoiceService) AddInvoicePayment(ctx context.Context, id string, input AddInvoicePaymentInput, key string) (Invoice, Payment, error) {
	if strings.TrimSpace(key) == "" {
		return Invoice{}, Payment{}, ErrMissingIdempotencyKey
	}
	if input.AmountCents <= 0 || strings.TrimSpace(input.Method) == "" {
		return Invoice{}, Payment{}, fmt.Errorf("%w: payment amount and method are required", ErrInvalidInput)
	}
	rk := "invoice:payment:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, Payment{}, err
	} else if ok {
		inv, e := s.store.GetInvoice(ctx, existing)
		if e != nil {
			return Invoice{}, Payment{}, e
		}
		if len(inv.Payments) == 0 {
			return inv, Payment{}, nil
		}
		return inv, inv.Payments[len(inv.Payments)-1], nil
	}
	release, err := s.idem.Lock(ctx, "invoice:payment-lock:"+id, 10*time.Second)
	if err != nil {
		return Invoice{}, Payment{}, err
	}
	defer release()
	if input.ReceivedAt == "" {
		input.ReceivedAt = nowUTC()
	}
	invoice, payment, _, err := s.store.AddInvoicePayment(ctx, id, Payment{AmountCents: input.AmountCents, Method: input.Method, Reference: input.Reference, ReceivedAt: input.ReceivedAt})
	if err != nil {
		return Invoice{}, Payment{}, err
	}
	if err := s.idem.Set(ctx, rk, invoice.ID, 24*time.Hour); err != nil {
		return Invoice{}, Payment{}, err
	}
	return invoice, payment, nil
}

func (s *InvoiceService) ReconcileInvoice(ctx context.Context, id, actor, key string) (Invoice, InvoiceEvent, error) {
	if strings.TrimSpace(key) == "" {
		return Invoice{}, InvoiceEvent{}, ErrMissingIdempotencyKey
	}
	rk := "invoice:reconcile:" + id + ":" + key
	if existing, ok, err := s.idem.Get(ctx, rk); err != nil {
		return Invoice{}, InvoiceEvent{}, err
	} else if ok {
		invoice, err := s.store.GetInvoice(ctx, existing)
		if err != nil {
			return Invoice{}, InvoiceEvent{}, err
		}
		return invoice, latestInvoiceEvent(invoice), nil
	}
	release, err := s.idem.Lock(ctx, "invoice:reconcile-lock:"+id, 10*time.Second)
	if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	defer release()
	if actor == "" {
		actor = "财务人员"
	}
	invoice, event, err := s.store.ReconcileInvoice(ctx, id, actor)
	if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	if err := s.idem.Set(ctx, rk, invoice.ID, 24*time.Hour); err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	return invoice, event, nil
}

func validInvoiceStatus(status string) bool { _, ok := invoiceTransitions[status]; return ok }

func (s *MemoryStore) ListInvoices(_ context.Context, page, pageSize int, status string) ([]Invoice, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Invoice, 0, len(s.invoices))
	for _, invoice := range s.invoices {
		if status == "" || invoice.Status == status {
			all = append(all, invoice)
		}
	}
	return paginate(all, page, pageSize)
}

func (s *MemoryStore) GetInvoice(_ context.Context, id string) (Invoice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	invoice, ok := s.invoices[id]
	if !ok {
		return Invoice{}, ErrNotFound
	}
	return s.invoiceSnapshot(invoice), nil
}

func (s *MemoryStore) CreateInvoice(_ context.Context, invoice Invoice) (Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if invoice.ID == "" {
		invoice.ID = s.next("INV")
	}
	if invoice.Status == "" {
		invoice.Status = InvoiceDraft
	}
	if invoice.CreatedAt == "" {
		invoice.CreatedAt = nowUTC()
	}
	invoice.UpdatedAt = invoice.CreatedAt
	for i := range invoice.Items {
		invoice.Items[i].ID = fmt.Sprintf("%s-ITEM-%d", invoice.ID, i+1)
		invoice.Items[i].InvoiceID = invoice.ID
	}
	s.invoices[invoice.ID] = invoice
	s.invoiceItems[invoice.ID] = append([]InvoiceItem(nil), invoice.Items...)
	s.invoiceEvents[invoice.ID] = []InvoiceEvent{{ID: s.next("INEV"), InvoiceID: invoice.ID, ToStatus: invoice.Status, Type: "created", Actor: "system", CreatedAt: invoice.CreatedAt}}
	return s.invoiceSnapshot(invoice), nil
}

func (s *MemoryStore) UpdateInvoiceStatus(_ context.Context, id, status, actor string) (Invoice, InvoiceEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	invoice, ok := s.invoices[id]
	if !ok {
		return Invoice{}, InvoiceEvent{}, ErrNotFound
	}
	if !invoiceTransitions[invoice.Status][status] {
		return Invoice{}, InvoiceEvent{}, ErrInvalidTransition
	}
	old := invoice.Status
	invoice.Status = status
	invoice.UpdatedAt = nowUTC()
	s.invoices[id] = invoice
	event := InvoiceEvent{ID: s.next("INEV"), InvoiceID: id, FromStatus: old, ToStatus: status, Type: "status", Actor: actor, CreatedAt: invoice.UpdatedAt}
	s.invoiceEvents[id] = append(s.invoiceEvents[id], event)
	return s.invoiceSnapshot(invoice), event, nil
}

func (s *MemoryStore) AddInvoicePayment(_ context.Context, id string, payment Payment) (Invoice, Payment, InvoiceEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	invoice, ok := s.invoices[id]
	if !ok {
		return Invoice{}, Payment{}, InvoiceEvent{}, ErrNotFound
	}
	if invoice.Status != InvoiceIssued && invoice.Status != InvoicePartiallyPaid {
		return Invoice{}, Payment{}, InvoiceEvent{}, fmt.Errorf("%w: payment is not allowed in %s", ErrInvalidTransition, invoice.Status)
	}
	payment.ID = s.next("PAY")
	payment.InvoiceID = id
	if payment.ReceivedAt == "" {
		payment.ReceivedAt = nowUTC()
	}
	s.invoicePayments[id] = append(s.invoicePayments[id], payment)
	invoice.PaidCents += payment.AmountCents
	old := invoice.Status
	if invoice.PaidCents > 0 {
		invoice.Status = InvoicePartiallyPaid
	}
	invoice.UpdatedAt = payment.ReceivedAt
	s.invoices[id] = invoice
	event := InvoiceEvent{ID: s.next("INEV"), InvoiceID: id, FromStatus: old, ToStatus: invoice.Status, Type: "payment", Actor: "财务人员", Note: payment.Reference, CreatedAt: payment.ReceivedAt}
	s.invoiceEvents[id] = append(s.invoiceEvents[id], event)
	return s.invoiceSnapshot(invoice), payment, event, nil
}

func (s *MemoryStore) ReconcileInvoice(_ context.Context, id, actor string) (Invoice, InvoiceEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	invoice, ok := s.invoices[id]
	if !ok {
		return Invoice{}, InvoiceEvent{}, ErrNotFound
	}
	if invoice.PaidCents < invoice.AmountCents {
		return Invoice{}, InvoiceEvent{}, fmt.Errorf("%w: paid amount is less than invoice amount", ErrInvalidInput)
	}
	if !invoiceTransitions[invoice.Status][InvoiceReconciled] {
		return Invoice{}, InvoiceEvent{}, ErrInvalidTransition
	}
	old := invoice.Status
	invoice.Status = InvoiceReconciled
	invoice.UpdatedAt = nowUTC()
	s.invoices[id] = invoice
	event := InvoiceEvent{ID: s.next("INEV"), InvoiceID: id, FromStatus: old, ToStatus: invoice.Status, Type: "reconcile", Actor: actor, CreatedAt: invoice.UpdatedAt}
	s.invoiceEvents[id] = append(s.invoiceEvents[id], event)
	return s.invoiceSnapshot(invoice), event, nil
}

func (s *MemoryStore) invoiceSnapshot(invoice Invoice) Invoice {
	invoice.Items = append([]InvoiceItem(nil), s.invoiceItems[invoice.ID]...)
	invoice.Payments = append([]Payment(nil), s.invoicePayments[invoice.ID]...)
	invoice.Events = append([]InvoiceEvent(nil), s.invoiceEvents[invoice.ID]...)
	return invoice
}

func latestInvoiceEvent(invoice Invoice) InvoiceEvent {
	if len(invoice.Events) == 0 {
		return InvoiceEvent{}
	}
	return invoice.Events[len(invoice.Events)-1]
}

// SQLStore implements the same invoice workflow for MySQL 8.4 deployments.
func (s *SQLStore) ListInvoices(ctx context.Context, page, pageSize int, status string) ([]Invoice, int, error) {
	page, pageSize = normalizePage(page, pageSize)
	args := []any{}
	where := ""
	if status != "" {
		where = " WHERE status=?"
		args = append(args, status)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invoices"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, "SELECT id,customer_id,customer_name,tax_number,currency,amount_cents,paid_cents,due_date,status,created_at,updated_at FROM invoices"+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Invoice{}
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.CustomerID, &inv.CustomerName, &inv.TaxNumber, &inv.Currency, &inv.AmountCents, &inv.PaidCents, &inv.DueDate, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, inv)
	}
	return out, total, rows.Err()
}

func (s *SQLStore) GetInvoice(ctx context.Context, id string) (Invoice, error) {
	var inv Invoice
	err := s.db.QueryRowContext(ctx, `SELECT id,customer_id,customer_name,tax_number,currency,amount_cents,paid_cents,due_date,status,created_at,updated_at FROM invoices WHERE id=?`, id).Scan(&inv.ID, &inv.CustomerID, &inv.CustomerName, &inv.TaxNumber, &inv.Currency, &inv.AmountCents, &inv.PaidCents, &inv.DueDate, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	items, err := s.db.QueryContext(ctx, `SELECT id,invoice_id,description,quantity,unit_price_cents,amount_cents FROM invoice_items WHERE invoice_id=? ORDER BY id`, id)
	if err != nil {
		return Invoice{}, err
	}
	defer items.Close()
	for items.Next() {
		var item InvoiceItem
		if err := items.Scan(&item.ID, &item.InvoiceID, &item.Description, &item.Quantity, &item.UnitPriceCents, &item.AmountCents); err != nil {
			return Invoice{}, err
		}
		inv.Items = append(inv.Items, item)
	}
	payments, err := s.db.QueryContext(ctx, `SELECT id,invoice_id,amount_cents,method,reference,received_at FROM invoice_payments WHERE invoice_id=? ORDER BY received_at`, id)
	if err != nil {
		return Invoice{}, err
	}
	defer payments.Close()
	for payments.Next() {
		var payment Payment
		if err := payments.Scan(&payment.ID, &payment.InvoiceID, &payment.AmountCents, &payment.Method, &payment.Reference, &payment.ReceivedAt); err != nil {
			return Invoice{}, err
		}
		inv.Payments = append(inv.Payments, payment)
	}
	events, err := s.db.QueryContext(ctx, `SELECT id,invoice_id,from_status,to_status,type,actor,note,created_at FROM invoice_events WHERE invoice_id=? ORDER BY created_at`, id)
	if err != nil {
		return Invoice{}, err
	}
	defer events.Close()
	for events.Next() {
		var event InvoiceEvent
		if err := events.Scan(&event.ID, &event.InvoiceID, &event.FromStatus, &event.ToStatus, &event.Type, &event.Actor, &event.Note, &event.CreatedAt); err != nil {
			return Invoice{}, err
		}
		inv.Events = append(inv.Events, event)
	}
	return inv, nil
}

func (s *SQLStore) CreateInvoice(ctx context.Context, inv Invoice) (Invoice, error) {
	if inv.ID == "" {
		inv.ID = fmt.Sprintf("INV-%d", time.Now().UnixNano())
	}
	if inv.CreatedAt == "" {
		inv.CreatedAt = nowUTC()
	}
	inv.UpdatedAt = inv.CreatedAt
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO invoices (id,customer_id,customer_name,tax_number,currency,amount_cents,paid_cents,due_date,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, inv.ID, inv.CustomerID, inv.CustomerName, inv.TaxNumber, inv.Currency, inv.AmountCents, 0, inv.DueDate, InvoiceDraft, inv.CreatedAt, inv.UpdatedAt)
	if err != nil {
		return Invoice{}, err
	}
	for i := range inv.Items {
		item := &inv.Items[i]
		item.ID = fmt.Sprintf("%s-ITEM-%d", inv.ID, i+1)
		item.InvoiceID = inv.ID
		if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_items (id,invoice_id,description,quantity,unit_price_cents,amount_cents) VALUES (?,?,?,?,?,?)`, item.ID, inv.ID, item.Description, item.Quantity, item.UnitPriceCents, item.AmountCents); err != nil {
			return Invoice{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_events (id,invoice_id,from_status,to_status,type,actor,created_at) VALUES (?,?,?,?,?,?,?)`, fmt.Sprintf("INEV-%d", time.Now().UnixNano()), inv.ID, "", InvoiceDraft, "created", "system", inv.CreatedAt); err != nil {
		return Invoice{}, err
	}
	if err = tx.Commit(); err != nil {
		return Invoice{}, err
	}
	return s.GetInvoice(ctx, inv.ID)
}

func (s *SQLStore) UpdateInvoiceStatus(ctx context.Context, id, status, actor string) (Invoice, InvoiceEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	defer tx.Rollback()
	var inv Invoice
	if err = tx.QueryRowContext(ctx, `SELECT id,customer_id,customer_name,tax_number,currency,amount_cents,paid_cents,due_date,status,created_at,updated_at FROM invoices WHERE id=? FOR UPDATE`, id).Scan(&inv.ID, &inv.CustomerID, &inv.CustomerName, &inv.TaxNumber, &inv.Currency, &inv.AmountCents, &inv.PaidCents, &inv.DueDate, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return Invoice{}, InvoiceEvent{}, ErrNotFound
	} else if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	if !invoiceTransitions[inv.Status][status] {
		return Invoice{}, InvoiceEvent{}, ErrInvalidTransition
	}
	old := inv.Status
	inv.Status = status
	inv.UpdatedAt = nowUTC()
	if _, err = tx.ExecContext(ctx, `UPDATE invoices SET status=?,updated_at=? WHERE id=?`, status, inv.UpdatedAt, id); err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	event := InvoiceEvent{ID: fmt.Sprintf("INEV-%d", time.Now().UnixNano()), InvoiceID: id, FromStatus: old, ToStatus: status, Type: "status", Actor: actor, CreatedAt: inv.UpdatedAt}
	if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_events (id,invoice_id,from_status,to_status,type,actor,created_at) VALUES (?,?,?,?,?,?,?)`, event.ID, id, old, status, event.Type, actor, event.CreatedAt); err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	full, err := s.GetInvoice(ctx, id)
	if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	return full, event, nil
}

func (s *SQLStore) AddInvoicePayment(ctx context.Context, id string, payment Payment) (Invoice, Payment, InvoiceEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	defer tx.Rollback()
	var inv Invoice
	if err = tx.QueryRowContext(ctx, `SELECT id,amount_cents,paid_cents,status FROM invoices WHERE id=? FOR UPDATE`, id).Scan(&inv.ID, &inv.AmountCents, &inv.PaidCents, &inv.Status); errors.Is(err, sql.ErrNoRows) {
		return Invoice{}, Payment{}, InvoiceEvent{}, ErrNotFound
	} else if err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	if inv.Status != InvoiceIssued && inv.Status != InvoicePartiallyPaid {
		return Invoice{}, Payment{}, InvoiceEvent{}, fmt.Errorf("%w: payment is not allowed", ErrInvalidTransition)
	}
	payment.ID = fmt.Sprintf("PAY-%d", time.Now().UnixNano())
	payment.InvoiceID = id
	if payment.ReceivedAt == "" {
		payment.ReceivedAt = nowUTC()
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_payments (id,invoice_id,amount_cents,method,reference,received_at) VALUES (?,?,?,?,?,?)`, payment.ID, id, payment.AmountCents, payment.Method, payment.Reference, payment.ReceivedAt); err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	inv.PaidCents += payment.AmountCents
	old := inv.Status
	inv.Status = InvoicePartiallyPaid
	if _, err = tx.ExecContext(ctx, `UPDATE invoices SET paid_cents=?,status=?,updated_at=? WHERE id=?`, inv.PaidCents, inv.Status, payment.ReceivedAt, id); err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	event := InvoiceEvent{ID: fmt.Sprintf("INEV-%d", time.Now().UnixNano()), InvoiceID: id, FromStatus: old, ToStatus: inv.Status, Type: "payment", Actor: "财务人员", Note: payment.Reference, CreatedAt: payment.ReceivedAt}
	if _, err = tx.ExecContext(ctx, `INSERT INTO invoice_events (id,invoice_id,from_status,to_status,type,actor,note,created_at) VALUES (?,?,?,?,?,?,?,?)`, event.ID, id, old, inv.Status, event.Type, event.Actor, event.Note, event.CreatedAt); err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Invoice{}, Payment{}, InvoiceEvent{}, err
	}
	full, e := s.GetInvoice(ctx, id)
	return full, payment, event, e
}

func (s *SQLStore) ReconcileInvoice(ctx context.Context, id, actor string) (Invoice, InvoiceEvent, error) {
	inv, err := s.GetInvoice(ctx, id)
	if err != nil {
		return Invoice{}, InvoiceEvent{}, err
	}
	if inv.PaidCents < inv.AmountCents {
		return Invoice{}, InvoiceEvent{}, fmt.Errorf("%w: paid amount is less than invoice amount", ErrInvalidInput)
	}
	if !invoiceTransitions[inv.Status][InvoiceReconciled] {
		return Invoice{}, InvoiceEvent{}, ErrInvalidTransition
	}
	return s.UpdateInvoiceStatus(ctx, id, InvoiceReconciled, actor)
}
