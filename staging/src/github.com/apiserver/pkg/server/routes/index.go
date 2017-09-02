package routes

import (
	"github.com/HuZhou/apiserver/pkg/server/mux"
	"net/http"
	"github.com/HuZhou/apiserver/pkg/endpoints/handlers/responsewriters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
)

// Index provides a webservice for the http root / listing all known paths.
type Index struct{}

// ListedPathProvider is an interface for providing paths that should be reported at /.
type ListedPathProvider interface {
	// ListedPaths is an alphabetically sorted list of paths to be reported at /.
	ListedPaths() []string
}

// Install adds the Index webservice to the given mux.
func (i Index) Install(pathProvider ListedPathProvider, mux *mux.PathRecorderMux) {
	handler := IndexLister{StatusCode: http.StatusOK, PathProvider: pathProvider}

	mux.UnlistedHandle("/", handler)
	mux.UnlistedHandle("/index.html", handler)
}


// IndexLister lists the available indexes with the status code provided
type IndexLister struct {
	StatusCode   int
	PathProvider ListedPathProvider
}

// ServeHTTP serves the available paths.
func (i IndexLister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(i.StatusCode)
	fmt.Println(i.PathProvider.ListedPaths())
	responsewriters.WriteRawJSON(i.StatusCode, metav1.RootPaths{Paths: i.PathProvider.ListedPaths()}, w)
}