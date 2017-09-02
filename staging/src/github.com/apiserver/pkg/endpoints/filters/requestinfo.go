package filters

import (
	"fmt"
	"net/http"
	"github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/HuZhou/apiserver/pkg/endpoints/handlers/responsewriters"
	"errors"
)

// WithRequestInfo attaches a RequestInfo to the context.
func WithRequestInfo(handler http.Handler, resolver *request.RequestInfoFactory, requestContextMapper request.RequestContextMapper) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, ok := requestContextMapper.Get(req)
		if !ok {
			responsewriters.InternalError(w, req, errors.New("no context found for request"))
			return
		}

		info, err := resolver.NewRequestInfo(req)
		if err != nil {
			responsewriters.InternalError(w, req, fmt.Errorf("failed to create RequestInfo: %v", err))
			return
		}

		requestContextMapper.Update(req, request.WithRequestInfo(ctx, info))

		handler.ServeHTTP(w, req)
	})
}