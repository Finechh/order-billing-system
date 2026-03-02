package billinggrpc

import (
	"order-billing-system/billing_service/internal/billing_domain/invoice"
	"order-billing-system/billing_service/internal/proto/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapInvoice(inv *invoice.Invoice) *pb.Invoice {
	var paidAt *timestamppb.Timestamp
	if inv.PaidAt != nil {
		paidAt = timestamppb.New(*inv.PaidAt)
	}

	return &pb.Invoice{
		Id:      inv.ID,
		OrderId: inv.OrderID,
		Amount: &pb.Money{
			Amount:   inv.Amount.Amount,
			Currency: inv.Amount.Currency,
		},
		Status:    mapDomainStatus(inv.Status),
		CreatedAt: timestamppb.New(inv.CreatedAt),
		PaidAt:    paidAt,
	}
}

func mapDomainStatus(s invoice.InvoiceStatus) pb.InvoiceStatus {
	switch s {
	case invoice.InvoiceStatusPending:
		return pb.InvoiceStatus_INVOICE_STATUS_PENDING
	case invoice.InvoiceStatusPaid:
		return pb.InvoiceStatus_INVOICE_STATUS_PAID
	case invoice.InvoiceStatusCancelled:
		return pb.InvoiceStatus_INVOICE_STATUS_CANCELLED
	default:
		return pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}

func mapProtoStatus(s pb.InvoiceStatus) invoice.InvoiceStatus {
	switch s {
	case pb.InvoiceStatus_INVOICE_STATUS_PENDING:
		return invoice.InvoiceStatusPending
	case pb.InvoiceStatus_INVOICE_STATUS_PAID:
		return invoice.InvoiceStatusPaid
	case pb.InvoiceStatus_INVOICE_STATUS_CANCELLED:
		return invoice.InvoiceStatusCancelled
	default:
		return invoice.InvoiceStatusPending
	}
}
