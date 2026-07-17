package main

import (
	"context"
	"errors"
	"testing"
)

func TestAppointmentStatusTransitions(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	ctx := context.Background()
	appointment, err := svc.CreateAppointment(ctx, CreateAppointmentInput{Patient: "林晓雨", Department: "全科门诊", Doctor: "林负责人", ScheduledAt: "2026-07-16T09:00:00+08:00"}, "create-1")
	if err != nil {
		t.Fatal(err)
	}
	steps := []string{"已确认", "候诊中", "处理中", "已完成"}
	for _, status := range steps {
		appointment, err = svc.UpdateAppointmentStatus(ctx, appointment.ID, status, "status-"+status)
		if err != nil {
			t.Fatalf("status %s: %v", status, err)
		}
		if appointment.Status != status {
			t.Fatalf("status = %q, want %q", appointment.Status, status)
		}
	}
	if _, err := svc.UpdateAppointmentStatus(ctx, appointment.ID, "处理中", "illegal-1"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
	events, err := store.ListAppointmentEvents(ctx, appointment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("events = %d, want 4", len(events))
	}
}

func TestAppointmentWriteRequiresIdempotencyKey(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	_, err := svc.CreateAppointment(context.Background(), CreateAppointmentInput{Patient: "沈明远"}, "")
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("expected missing idempotency key, got %v", err)
	}
}

func TestAppointmentWriteIsIdempotent(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	input := CreateAppointmentInput{Patient: "赵可心", Department: "皮肤科", Doctor: "沈负责人"}
	a, err := svc.CreateAppointment(context.Background(), input, "same-key")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateAppointment(context.Background(), input, "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("idempotency returned %q then %q", a.ID, b.ID)
	}
}

func TestFollowupCompletesOnce(t *testing.T) {
	store := NewMemoryStore()
	svc := NewCareService(store, NoopIdempotency{})
	followup, err := store.CreateFollowup(context.Background(), Followup{Patient: "林晓雨", Summary: "术后回访"})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := svc.CompleteFollowup(context.Background(), followup.ID, "followup-1")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "已完成" {
		t.Fatalf("status = %q", completed.Status)
	}
	if _, err := svc.CompleteFollowup(context.Background(), followup.ID, "followup-2"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid completion, got %v", err)
	}
}

func TestInvoiceLifecycleRecordsEventsPaymentsAndReconcile(t *testing.T) {
	store := NewMemoryStore()
	svc := NewInvoiceService(store, NoopIdempotency{})
	ctx := context.Background()
	invoice, err := svc.CreateInvoice(ctx, CreateInvoiceInput{CustomerName: "星河科技", AmountCents: 128000, Items: []InvoiceItemInput{{Description: "软件服务", Quantity: 1, UnitPriceCents: 128000}}}, "invoice-create-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{InvoicePendingReview, InvoiceIssued} {
		invoice, err = svc.UpdateInvoiceStatus(ctx, invoice.ID, status, "财务", "invoice-status-"+status)
		if err != nil {
			t.Fatal(err)
		}
	}
	invoice, _, err = svc.AddInvoicePayment(ctx, invoice.ID, AddInvoicePaymentInput{AmountCents: 128000, Method: "银行转账", Reference: "PAY-001"}, "invoice-pay-1")
	if err != nil {
		t.Fatal(err)
	}
	if invoice.PaidCents != 128000 || invoice.Status != InvoicePartiallyPaid {
		t.Fatalf("paid/status = %d/%s", invoice.PaidCents, invoice.Status)
	}
	invoice, _, err = svc.ReconcileInvoice(ctx, invoice.ID, "财务", "invoice-reconcile-1")
	if err != nil {
		t.Fatal(err)
	}
	if invoice.Status != InvoiceReconciled {
		t.Fatalf("status = %s", invoice.Status)
	}
	detail, err := store.GetInvoice(ctx, invoice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Items) != 1 || len(detail.Payments) != 1 || len(detail.Events) < 4 {
		t.Fatalf("detail = %+v", detail)
	}
}

func TestInvoiceWritesRequireIdempotencyKey(t *testing.T) {
	store := NewMemoryStore()
	svc := NewInvoiceService(store, NoopIdempotency{})
	_, err := svc.CreateInvoice(context.Background(), CreateInvoiceInput{CustomerName: "无幂等键", AmountCents: 100}, "")
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("expected missing idempotency key, got %v", err)
	}
}
