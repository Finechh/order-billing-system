package billinggrpc

import (
	"context"
	"errors"

	pb "order-billing-system/billing_service/internal/proto/pb"
	"order-billing-system/billing_service/internal/query"
	"order-billing-system/billing_service/internal/billing_domain/invoice"
	errorsx "order-billing-system/shared/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BillingQueryHandler struct {
	pb.UnimplementedBillingQueryServiceServer
	query *query.InvoiceQueryService
}

func NewBillingQueryHandler(q *query.InvoiceQueryService) *BillingQueryHandler {
	return &BillingQueryHandler{query: q}
}

func (h *BillingQueryHandler) GetInvoiceByOrder(ctx context.Context, req *pb.GetInvoiceByOrderRequest) (*pb.GetInvoiceResponse, error) {
	inv, err := h.query.GetByOrderID(ctx, req.OrderId)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.GetInvoiceResponse{
		Invoice: mapInvoice(inv),
	}, nil
}

func (h *BillingQueryHandler) ListInvoices(ctx context.Context, req *pb.ListInvoicesRequest) (*pb.ListInvoicesResponse, error) {
	var status *invoice.InvoiceStatus
	if req.Status != pb.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED {
		s := mapProtoStatus(req.Status)
		status = &s
	}

	invoices, err := h.query.List(ctx, status, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, mapError(err)
	}

	res := make([]*pb.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		res = append(res, mapInvoice(&inv))
	}

	return &pb.ListInvoicesResponse{
		Invoices: res,
	}, nil
}

func (h *BillingQueryHandler) GetBillingStats(ctx context.Context, _ *pb.GetBillingStatsRequest) (*pb.GetBillingStatsResponse, error) {
	stats, err := h.query.GetStats(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.GetBillingStatsResponse{
		TotalInvoices:     stats.TotalInvoices,
		PaidInvoices:      stats.PaidInvoices,
		PendingInvoices:   stats.PendingInvoices,
		CancelledInvoices: stats.CancelledInvoices,
		TotalRevenue: &pb.Money{
			Amount:   stats.TotalRevenue.Amount,
			Currency: stats.TotalRevenue.Currency,
		},
	}, nil
}

func mapError(err error) error {
	var appErr *errorsx.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case "ERR_NOT_FOUND":
			return status.Error(codes.NotFound, appErr.Message)
		case "ERR_INVALID_INPUT":
			return status.Error(codes.InvalidArgument, appErr.Message)
		case "ERR_INVALID_ORDER_STATE":
			return status.Error(codes.FailedPrecondition, appErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
