package pkgs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mmihic/go-tools/pkg/path"
)

func TestMoves_BestMatch(t *testing.T) {
	rules := Moves{
		{
			From: path.NewPath("github.com/mmihic/go-tools/pkg/first"),
			To:   path.NewPath("github.com/mmihic/go-tools/pkg/other"),
		},
		{
			From: path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"),
			To:   path.NewPath("github.com/mmihic/go-tools/cmd/pkgmove"),
		},
		{
			From: path.NewPath("github.com/mmihic/go-tools/pkg/first/something"),
			To:   path.NewPath("github.com/mmihic/go-tools/pkg/newpkg"),
		},
	}

	match := rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"), match.From)

	match = rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first/something"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/pkg/first/something"), match.From)

	match = rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first/somethingelse"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/cmd/othertool"))
	require.Nil(t, match)
}

func TestMoves_ExactMatch(t *testing.T) {
	rules := Moves{
		{
			From: path.NewPath("github.com/mmihic/go-tools/pkg/first"),
			To:   path.NewPath("github.com/mmihic/go-tools/pkg/other"),
		},
		{
			From: path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"),
			To:   path.NewPath("github.com/mmihic/go-tools/cmd/pkgmove"),
		},
		{
			From: path.NewPath("github.com/mmihic/go-tools/pkg/first/something"),
			To:   path.NewPath("github.com/mmihic/go-tools/pkg/newpkg"),
		},
	}

	match := rules.ExactMatch(path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/cmd/pkgalign"), match.From)

	match = rules.ExactMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first/something"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/pkg/first/something"), match.From)

	match = rules.ExactMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first"))
	require.NotNil(t, match)
	require.Equal(t, path.NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.ExactMatch(path.NewPath("github.com/mmihic/go-tools/pkg/first/somethingelse"))
	require.Nil(t, match)

	match = rules.BestMatch(path.NewPath("github.com/mmihic/go-tools/cmd/othertool"))
	require.Nil(t, match)
}
