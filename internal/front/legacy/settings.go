package legacy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// What a source remembers about how to build it.
//
// Some of what a build needs is a property of the adventure rather than a
// choice the person compiling it makes. The font is the clearest case: La
// Aventura Original draws the little icons beside its objects with glyphs of
// its own, and built with anybody else's font those come out as "ßÊì". The
// font is not an opinion about the build; it is part of the game.
//
// Carrying it on the command line means remembering it, and forgetting it is
// silent — the adventure compiles, boots, and plays with its icons wrong. So
// the source says it.
//
// It says it in a comment, which is the part worth being pleased about. A
// semicolon starts a comment in every DAAD tool there has ever been, so a
// source with these lines in it is still an ordinary source: the reference
// compiler reads it without noticing, and nothing about the format has
// changed. We are going to have a format of our own one day and this does not
// pre-empt it.
//
//	; qdaad: charset=gfx/chr/udg.chr
//
// Only the header is read — everything before the first directive line — so
// that a setting cannot hide a thousand lines into the tables, and so that
// finding them costs one pass over a few dozen lines.
// The type they go into is the one the preprocessor already fills from the
// source's own directives: a setting is a setting whether the source stated it
// with a # or with a comment.

// settingsPrefix marks a comment this reads. It is deliberately the tool's own
// name: a source is read by more than one program, and a directive that said
// only "charset=" would be staking a claim on a comment syntax that is not
// ours to claim.
const settingsPrefix = "qdaad:"

// ReadSettings takes the build settings out of a source's header comments.
//
// A source with none gives back an empty Settings and no error, which is the
// usual case and not a problem: every source written before this existed is
// one.
func readHeaderSettings(filename string) (Settings, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Settings{}, err
	}

	defer func() { _ = file.Close() }()

	var out Settings

	dir := filepath.Dir(filename)
	scanner := bufio.NewScanner(file)

	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())

		// The header ends at the first thing that is not a comment or blank.
		// A directive, a section, anything: past here the file is the
		// adventure and not what to build it with.
		if text != "" && !strings.HasPrefix(text, ";") {
			break
		}

		body, ok := settingComment(text)
		if !ok {
			continue
		}

		if err := out.set(body, dir); err != nil {
			return Settings{}, fmt.Errorf("%s:%d: %w", filename, line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return Settings{}, err
	}

	return out, nil
}

// settingComment reports whether a line is one of ours and gives back what it
// says.
func settingComment(text string) (string, bool) {
	if !strings.HasPrefix(text, ";") {
		return "", false
	}

	body := strings.TrimSpace(strings.TrimLeft(text, "; "))

	rest, ok := strings.CutPrefix(body, settingsPrefix)
	if !ok {
		return "", false
	}

	return strings.TrimSpace(rest), true
}

// set applies one setting.
//
// A name we do not know is an error rather than something to skip. A setting
// that is quietly ignored is worse than one that is missing: the source says
// what it wants, the build does something else, and the only sign is in the
// result.
func (s *Settings) set(body, dir string) error {
	name, value, ok := strings.Cut(body, "=")
	if !ok {
		return fmt.Errorf("%q is not a setting: they are written name=value", body)
	}

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	if value == "" {
		return fmt.Errorf("%q has no value", name)
	}

	switch name {
	case "charset":
		s.Charset = resolve(value, dir)
	default:
		return fmt.Errorf("%q is not a setting this knows; it knows charset", name)
	}

	return nil
}

// resolve makes a path from the source relative to the source.
func resolve(path, dir string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(dir, path)
}
