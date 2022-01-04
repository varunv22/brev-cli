package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLocalIdentifier(t *testing.T) {
	w := Workspace{DNS: "test-rand-org.brev.sh"}
	assert.Equal(t, "test-rand", w.GetLocalIdentifier())
}
