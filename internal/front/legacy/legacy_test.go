package legacy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jorgefuertes/QDAAD/internal/front/legacy"
	"github.com/stretchr/testify/require"
)

// The template DAAD READY ships is the acceptance test for the passes that come
// before the parser: it is the source everyone starts from and it uses nearly
// all of the format. It is ISO-8859-1, which we do not read, so it is converted
// first -- which is the workflow the plan describes for any source of theirs.
const template = "../../../work/DRC/BLANK_ES.DSF"

func TestReadsTheDAADReadyTemplate(t *testing.T) {
	if _, err := os.Stat(template); err != nil {
		t.Skipf("the template is not in the tree: %v", err)
	}

	iconv, err := exec.LookPath("iconv")
	if err != nil {
		t.Skip("iconv is not installed")
	}

	utf8 := filepath.Join(t.TempDir(), "BLANK_ES.DSF")

	out, err := exec.Command(iconv, "-f", "ISO-8859-1", "-t", "UTF-8", template).Output()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(utf8, out, 0o600))

	tokens, symbols, err := legacy.Read(utf8)
	require.NoError(t, err)

	require.NotEmpty(t, tokens)
	require.Equal(t, legacy.EOF, tokens[len(tokens)-1].Kind)

	// The flags the standard library defines, as the template writes them.
	require.Equal(t, 28, symbols["fDarkF"])
	require.Equal(t, 38, symbols["fPlayer"])
	require.Equal(t, 87, symbols["TIMEOUT"])
}
