package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type prompter struct {
	in    io.Reader
	out   io.Writer
	close func() error
}

func newPrompter() (*prompter, error) {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return &prompter{in: os.Stdin, out: os.Stderr}, nil
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("not running in a terminal; use --port, --sni, or --non-interactive")
	}
	return &prompter{
		in:    tty,
		out:   tty,
		close: tty.Close,
	}, nil
}

func canPrompt() bool {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return true
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = tty.Close()
	return true
}

func (p *prompter) readLine() (string, error) {
	reader, ok := p.in.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(p.in)
		p.in = reader
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *prompter) writePrompt(format string, args ...any) {
	fmt.Fprintf(p.out, format, args...)
	if w, ok := p.out.(interface{ Sync() error }); ok {
		_ = w.Sync()
	}
}

func (p *prompter) promptString(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		p.writePrompt("%s [%s]: ", label, defaultValue)
	} else {
		p.writePrompt("%s: ", label)
	}
	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func (p *prompter) promptInt(label string, defaultValue int) (int, error) {
	p.writePrompt("%s [%d]: ", label, defaultValue)
	line, err := p.readLine()
	if err != nil {
		return 0, err
	}
	if line == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", label)
	}
	return value, nil
}
