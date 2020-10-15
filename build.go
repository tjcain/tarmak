// +build tools
package main

import (
	"fmt"

	_ "github.com/golang/mock/mockgen"
	_ "github.com/kevinburke/go-bindata/go-bindata"
	_ "github.com/kubernetes-incubator/reference-docs/gen-apidocs"
	_ "github.com/tcnksm/ghr"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/conversion-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)

func main() {
	fmt.Printf("You just lost the game\n")
}
