package responsewriters

import (
	"github.com/HuZhou/apiserver/pkg/endpoints/request"
	"github.com/HuZhou/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	"strconv"
	"github.com/HuZhou/apiserver/pkg/audit"
	"encoding/json"
	"fmt"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// ErrorNegotiated renders an error to the response. Returns the HTTP status code of the error.
// The context is optional and may be nil.
func ErrorNegotiated(ctx request.Context, err error, s runtime.NegotiatedSerializer, gv schema.GroupVersion, w http.ResponseWriter, req *http.Request) int {
	status := ErrorToAPIStatus(err)
	code := int(status.Code)
	// when writing an error, check to see if the status indicates a retry after period
	if status.Details != nil && status.Details.RetryAfterSeconds > 0 {
		delay := strconv.Itoa(int(status.Details.RetryAfterSeconds))
		w.Header().Set("Retry-After", delay)
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return code
	}

	WriteObjectNegotiated(ctx, s, gv, w, req, code, status)
	return code
}

// errorJSONFatal renders an error to the response, and if codec fails will render plaintext.
// Returns the HTTP status code of the error.
func errorJSONFatal(err error, codec runtime.Encoder, w http.ResponseWriter) int {
	utilruntime.HandleError(fmt.Errorf("apiserver was unable to write a JSON response: %v", err))
	status := ErrorToAPIStatus(err)
	code := int(status.Code)
	output, err := runtime.Encode(codec, status)
	if err != nil {
		w.WriteHeader(code)
		fmt.Fprintf(w, "%s: %s", status.Reason, status.Message)
		return code
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(output)
	return code
}

// SerializeObject renders an object in the content type negotiated by the client using the provided encoder.
// The context is optional and can be nil.
func SerializeObject(mediaType string, encoder runtime.Encoder, w http.ResponseWriter, req *http.Request, statusCode int, object runtime.Object) {
	w.Header().Set("Content-Type", mediaType)
	w.WriteHeader(statusCode)

	if err := encoder.Encode(object, w); err != nil {
		errorJSONFatal(err, encoder, w)
	}
}

// WriteObjectNegotiated renders an object in the content type negotiated by the client.
// The context is optional and can be nil.
func WriteObjectNegotiated(ctx request.Context, s runtime.NegotiatedSerializer, gv schema.GroupVersion, w http.ResponseWriter, req *http.Request, statusCode int, object runtime.Object) {
	serializer, err := negotiation.NegotiateOutputSerializer(req, s)
	if err != nil {
		status := ErrorToAPIStatus(err)
		WriteRawJSON(int(status.Code), status, w)
		return
	}

	if ae := request.AuditEventFrom(ctx); ae != nil {
		audit.LogResponseObject(ae, object, gv, s)
	}

	encoder := s.EncoderForVersion(serializer.Serializer, gv)
	SerializeObject(serializer.MediaType, encoder, w, req, statusCode, object)
}

// WriteRawJSON writes a non-API object in JSON.
func WriteRawJSON(statusCode int, object interface{}, w http.ResponseWriter) {
	output, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(output)
}