package pkgalign

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath_Contains(t *testing.T) {
	p := NewPath("github.com/mmihic/go-tools/tools/pkgalign")
	assert.True(t, p.Contains(NewPath("github.com/mmihic/go-tools/tools/pkgalign")))
	assert.True(t, p.Contains(NewPath("github.com/mmihic/go-tools/tools/pkgalign/nested")))
	assert.False(t, p.Contains(NewPath("github.com/mmihic/go-tools/tools")))
}

func TestPath_Equal(t *testing.T) {
	p := NewPath("github.com/mmihic/go-tools/tools/pkgalign")
	assert.True(t, p.Equal(NewPath("github.com/mmihic/go-tools/tools/pkgalign")))
	assert.False(t, p.Equal(NewPath("github.com/mmihic/go-tools/tools")))
	assert.False(t, p.Equal(NewPath("github.com/mmihic/go-tools/tools/pkgalign/nested")))
}

func TestPath_Append(t *testing.T) {
	assert.Equal(t,
		NewPath("github.com/mmihic").Append(NewPath("tools/pkgalign")),
		NewPath("github.com/mmihic/go-tools/tools/pkgalign"))
	assert.Equal(t,
		NewPath("").Append(NewPath("github.com/mmihic/go-tools/tools/pkgalign")),
		NewPath("github.com/mmihic/go-tools/tools/pkgalign"))
}
