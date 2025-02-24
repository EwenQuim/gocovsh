package program_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/orlangure/gocovsh/internal/program"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	version := "1.2.3"
	commit := "abcdef"
	date := time.Now().Format(time.RFC3339)
	buf := bytes.NewBuffer(nil)

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	p := program.New(
		program.WithBuildInfo(version, commit, date),
		program.WithOutput(buf),
		program.WithFlagSet(flagSet, []string{"--version"}),
	)

	require.NoError(t, p.Run())

	expectedVersion := fmt.Sprintf("Version: %s\nCommit: %s\nDate: %s\n", version, commit, date)
	require.Equal(t, expectedVersion, buf.String())
}

func TestLogger(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	f, err := os.CreateTemp("", "test-logger")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(f.Name()))
	})

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	p := program.New(
		program.WithOutput(buf),
		program.WithLogFile(f.Name()),
		program.WithFlagSet(flagSet, nil),
	)

	// in tests, the program fails
	require.Error(t, p.Run())

	logs, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	require.Contains(t, string(logs), "logging to")

	if runtime.GOOS == "windows" {
		// https://github.com/docker/compose/issues/8186#issuecomment-814180124
		t.Log("Workaround for containerd/console bug")
	}
}

func TestInput(t *testing.T) {
	t.Run("read input with pipe mode", func(t *testing.T) {
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
		f := newMockFile("foo\nbar\r\nbaz\n", os.ModeNamedPipe)
		p := program.New(
			program.WithInput(f),
			program.WithFlagSet(flagSet, nil),
		)

		require.Error(t, p.Run())
		require.True(t, f.inputRead)
	})

	t.Run("ignore input with other mode", func(t *testing.T) {
		flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
		f := newMockFile("foo\nbar\r\nbaz\n", os.ModeDir)
		p := program.New(
			program.WithInput(f),
			program.WithFlagSet(flagSet, nil),
		)

		require.Error(t, p.Run())
		require.False(t, f.inputRead)
	})
}

func newMockFile(data string, mode fs.FileMode) *mockFile {
	return &mockFile{
		reader: bytes.NewBufferString(data),
		size:   int64(len(data)),
		mode:   mode,
	}
}

type mockFile struct {
	reader    io.Reader
	mode      fs.FileMode
	size      int64
	inputRead bool
}

func (m *mockFile) Stat() (fs.FileInfo, error) { return &mockFileInfo{size: m.size, mode: m.mode}, nil }
func (m *mockFile) Close() error               { return nil }
func (m *mockFile) Read(buf []byte) (int, error) {
	m.inputRead = true
	return m.reader.Read(buf)
}

type mockFileInfo struct {
	fs.FileInfo
	size int64
	mode fs.FileMode
}

func (fi *mockFileInfo) Mode() os.FileMode { return fi.mode }
func (fi *mockFileInfo) Size() int64       { return fi.size }
