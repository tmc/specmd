package okf_test

import (
	"fmt"
	"path/filepath"

	"github.com/tmc/specmd/okf"
)

func ExampleParseBundle() {
	bundle, err := okf.ParseBundle(filepath.Join("testdata", "okf"))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(bundle.Concepts))
	fmt.Println(bundle.Concepts[0].ID)
	fmt.Println(bundle.Concepts[0].Type)
	// Output:
	// 2
	// datasets/sales
	// BigQuery Dataset
}
