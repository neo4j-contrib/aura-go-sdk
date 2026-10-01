package v2beta1

import (
	"testing"

	"github.com/neo4j-contrib/aura-go-sdk/internal/apisurface"
)

// TestAPISurface guards against accidental changes to this package's public
// API. If this test fails after an intentional change, regenerate the
// golden file with:
//
//	UPDATE_GOLDEN=1 go test ./... -run TestAPISurface
//
// and add a changie fragment (Changed, for breaking changes) alongside the
// code change.
func TestAPISurface(t *testing.T) {
	apisurface.AssertGolden(t, ".", "testdata/api_surface.golden")
}
