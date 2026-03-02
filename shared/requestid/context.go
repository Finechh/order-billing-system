package requestid

import "context"

type contextKey string

const requestIDKey contextKey = "request_id"

func Set(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func Get(ctx context.Context) string {
	v := ctx.Value(requestIDKey)
	if v == nil {
		return ""
	}
	return v.(string)
}
