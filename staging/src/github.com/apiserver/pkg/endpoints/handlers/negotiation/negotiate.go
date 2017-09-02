package negotiation

import (
	"net/http"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"

	"bitbucket.org/ww/goautoneg"
	"strconv"
)

// MediaTypesForSerializer returns a list of media and stream media types for the server.
func MediaTypesForSerializer(ns runtime.NegotiatedSerializer) (mediaTypes, streamMediaTypes []string) {
	for _, info := range ns.SupportedMediaTypes() {
		mediaTypes = append(mediaTypes, info.MediaType)
		if info.StreamSerializer != nil {
			// stream=watch is the existing mime-type parameter for watch
			streamMediaTypes = append(streamMediaTypes, info.MediaType+";stream=watch")
		}
	}
	return mediaTypes, streamMediaTypes
}

// EndpointRestrictions is an interface that allows content-type negotiation
// to verify server support for specific options
type EndpointRestrictions interface {
	// AllowsConversion should return true if the specified group version kind
	// is an allowed target object.
	AllowsConversion(schema.GroupVersionKind) bool
	// AllowsServerVersion should return true if the specified version is valid
	// for the server group.
	AllowsServerVersion(version string) bool
	// AllowsStreamSchema should return true if the specified stream schema is
	// valid for the server group.
	AllowsStreamSchema(schema string) bool
}

// AcceptedMediaType contains information about a valid media type that the
// server can serialize.
type AcceptedMediaType struct {
	// Type is the first part of the media type ("application")
	Type string
	// SubType is the second part of the media type ("json")
	SubType string
	// Serializer is the serialization info this object accepts
	Serializer runtime.SerializerInfo
}

// MediaTypeOptions describes information for a given media type that may alter
// the server response
type MediaTypeOptions struct {
	// pretty is true if the requested representation should be formatted for human
	// viewing
	Pretty bool

	// stream, if set, indicates that a streaming protocol variant of this encoding
	// is desired. The only currently supported value is watch which returns versioned
	// events. In the future, this may refer to other stream protocols.
	Stream string

	// convert is a request to alter the type of object returned by the server from the
	// normal response
	Convert *schema.GroupVersionKind
	// useServerVersion is an optional version for the server group
	UseServerVersion string

	// export is true if the representation requested should exclude fields the server
	// has set
	Export bool

	// unrecognized is a list of all unrecognized keys
	Unrecognized []string

	// the accepted media type from the client
	Accepted *AcceptedMediaType
}


var DefaultEndpointRestrictions = emptyEndpointRestrictions{}


type emptyEndpointRestrictions struct{}

func (emptyEndpointRestrictions) AllowsConversion(schema.GroupVersionKind) bool { return false }
func (emptyEndpointRestrictions) AllowsServerVersion(string) bool               { return false }
func (emptyEndpointRestrictions) AllowsStreamSchema(s string) bool              { return s == "watch" }
// NegotiateMediaTypeOptions returns the most appropriate content type given the accept header and
// a list of alternatives along with the accepted media type parameters.
func NegotiateMediaTypeOptions(header string, accepted []AcceptedMediaType, endpoint EndpointRestrictions) (MediaTypeOptions, bool) {
	if len(header) == 0 && len(accepted) > 0 {
		return MediaTypeOptions{
			Accepted: &accepted[0],
		}, true
	}

	clauses := goautoneg.ParseAccept(header)
	for _, clause := range clauses {
		for i := range accepted {
			accepts := &accepted[i]
			switch {
			case clause.Type == accepts.Type && clause.SubType == accepts.SubType,
				clause.Type == accepts.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				// TODO: should we prefer the first type with no unrecognized options?  Do we need to ignore unrecognized
				// parameters.
				return acceptMediaTypeOptions(clause.Params, accepts, endpoint)
			}
		}
	}
	return MediaTypeOptions{}, false
}

func NegotiateOutputMediaType(req *http.Request, ns runtime.NegotiatedSerializer, restrictions EndpointRestrictions) (MediaTypeOptions, runtime.SerializerInfo, error) {
	mediaType, ok := NegotiateMediaTypeOptions(req.Header.Get("Accept"), AcceptedMediaTypesForEndpoint(ns), restrictions)
	if !ok {
		supported, _ := MediaTypesForSerializer(ns)
		return mediaType, runtime.SerializerInfo{}, NewNotAcceptableError(supported)
	}
	// TODO: move into resthandler
	info := mediaType.Accepted.Serializer
	if (mediaType.Pretty || isPrettyPrint(req)) && info.PrettySerializer != nil {
		info.Serializer = info.PrettySerializer
	}
	return mediaType, info, nil
}

// isPrettyPrint returns true if the "pretty" query parameter is true or if the User-Agent
// matches known "human" clients.
func isPrettyPrint(req *http.Request) bool {
	// DEPRECATED: should be part of the content type
	if req.URL != nil {
		pp := req.URL.Query().Get("pretty")
		if len(pp) > 0 {
			pretty, _ := strconv.ParseBool(pp)
			return pretty
		}
	}
	userAgent := req.UserAgent()
	// This covers basic all browers and cli http tools
	if strings.HasPrefix(userAgent, "curl") || strings.HasPrefix(userAgent, "Wget") || strings.HasPrefix(userAgent, "Mozilla/5.0") {
		return true
	}
	return false
}

func NegotiateOutputSerializer(req *http.Request, ns runtime.NegotiatedSerializer) (runtime.SerializerInfo, error) {
	_, info, err := NegotiateOutputMediaType(req, ns, DefaultEndpointRestrictions)
	return info, err
}


// acceptMediaTypeOptions returns an options object that matches the provided media type params. If
// it returns false, the provided options are not allowed and the media type must be skipped.  These
// parameters are unversioned and may not be changed.
func acceptMediaTypeOptions(params map[string]string, accepts *AcceptedMediaType, endpoint EndpointRestrictions) (MediaTypeOptions, bool) {
	var options MediaTypeOptions

	// extract all known parameters
	for k, v := range params {
		switch k {

		// controls transformation of the object when returned
		case "as":
			if options.Convert == nil {
				options.Convert = &schema.GroupVersionKind{}
			}
			options.Convert.Kind = v
		case "g":
			if options.Convert == nil {
				options.Convert = &schema.GroupVersionKind{}
			}
			options.Convert.Group = v
		case "v":
			if options.Convert == nil {
				options.Convert = &schema.GroupVersionKind{}
			}
			options.Convert.Version = v

			// controls the streaming schema
		case "stream":
			if len(v) > 0 && (accepts.Serializer.StreamSerializer == nil || !endpoint.AllowsStreamSchema(v)) {
				return MediaTypeOptions{}, false
			}
			options.Stream = v

			// controls the version of the server API group used
			// for generic output
		case "sv":
			if len(v) > 0 && !endpoint.AllowsServerVersion(v) {
				return MediaTypeOptions{}, false
			}
			options.UseServerVersion = v

			// if specified, the server should transform the returned
			// output and remove fields that are always server specified,
			// or which fit the default behavior.
		case "export":
			options.Export = v == "1"

			// if specified, the pretty serializer will be used
		case "pretty":
			options.Pretty = v == "1"

		default:
			options.Unrecognized = append(options.Unrecognized, k)
		}
	}

	if options.Convert != nil && !endpoint.AllowsConversion(*options.Convert) {
		return MediaTypeOptions{}, false
	}

	options.Accepted = accepts
	return options, true
}

// AcceptedMediaTypesForEndpoint returns an array of structs that are used to efficiently check which
// allowed media types the server exposes.
func AcceptedMediaTypesForEndpoint(ns runtime.NegotiatedSerializer) []AcceptedMediaType {
	var acceptedMediaTypes []AcceptedMediaType
	for _, info := range ns.SupportedMediaTypes() {
		segments := strings.SplitN(info.MediaType, "/", 2)
		if len(segments) == 1 {
			segments = append(segments, "*")
		}
		t := AcceptedMediaType{
			Type:       segments[0],
			SubType:    segments[1],
			Serializer: info,
		}
		acceptedMediaTypes = append(acceptedMediaTypes, t)
	}
	return acceptedMediaTypes
}