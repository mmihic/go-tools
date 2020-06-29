package pkgalign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriteRules_BestMatch(t *testing.T) {
	rules := RewriteRules{
		{
			From: NewPath("github.com/mmihic/go-tools/pkg/first"),
			To:   NewPath("github.com/mmihic/go-tools/pkg/other"),
		},
		{
			From: NewPath("github.com/mmihic/go-tools/cmd/pkgalign"),
			To:   NewPath("github.com/mmihic/go-tools/cmd/pkgmove"),
		},
		{
			From: NewPath("github.com/mmihic/go-tools/pkg/first/something"),
			To:   NewPath("github.com/mmihic/go-tools/pkg/newpkg"),
		},
	}

	match := rules.BestMatch(NewPath("github.com/mmihic/go-tools/cmd/pkgalign"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/cmd/pkgalign"), match.From)

	match = rules.BestMatch(NewPath("github.com/mmihic/go-tools/pkg/first/something"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/pkg/first/something"), match.From)

	match = rules.BestMatch(NewPath("github.com/mmihic/go-tools/pkg/first"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.BestMatch(NewPath("github.com/mmihic/go-tools/pkg/first/somethingelse"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.BestMatch(NewPath("github.com/mmihic/go-tools/cmd/othertool"))
	require.Nil(t, match)
}

func TestRewriteRules_ExactMatch(t *testing.T) {
	rules := RewriteRules{
		{
			From: NewPath("github.com/mmihic/go-tools/pkg/first"),
			To:   NewPath("github.com/mmihic/go-tools/pkg/other"),
		},
		{
			From: NewPath("github.com/mmihic/go-tools/cmd/pkgalign"),
			To:   NewPath("github.com/mmihic/go-tools/cmd/pkgmove"),
		},
		{
			From: NewPath("github.com/mmihic/go-tools/pkg/first/something"),
			To:   NewPath("github.com/mmihic/go-tools/pkg/newpkg"),
		},
	}

	match := rules.ExactMatch(NewPath("github.com/mmihic/go-tools/cmd/pkgalign"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/cmd/pkgalign"), match.From)

	match = rules.ExactMatch(NewPath("github.com/mmihic/go-tools/pkg/first/something"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/pkg/first/something"), match.From)

	match = rules.ExactMatch(NewPath("github.com/mmihic/go-tools/pkg/first"))
	require.NotNil(t, match)
	require.Equal(t, NewPath("github.com/mmihic/go-tools/pkg/first"), match.From)

	match = rules.ExactMatch(NewPath("github.com/mmihic/go-tools/pkg/first/somethingelse"))
	require.Nil(t, match)

	match = rules.BestMatch(NewPath("github.com/mmihic/go-tools/cmd/othertool"))
	require.Nil(t, match)
}
