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
	in  io.Reader
	out io.Writer
}

func newPrompter() *prompter {
	p := &prompter{out: os.Stderr}
	if canPrompt() {
		p.in = os.Stdin
	}
	return p
}

// canPrompt is true only when stdin is an interactive terminal.
// Do not treat /dev/tty availability as interactive (breaks curl|bash and ssh without -t).
func canPrompt() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func (p *prompter) readLine() (string, error) {
	if p.in == nil {
		return "", fmt.Errorf("not running in a terminal; use --port, --sni, or --non-interactive")
	}
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

func (p *prompter) promptString(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	if w, ok := p.out.(interface{ Sync() error }); ok {
		_ = w.Sync()
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
	fmt.Fprintf(p.out, "%s [%d]: ", label, defaultValue)
	if w, ok := p.out.(interface{ Sync() error }); ok {
		_ = w.Sync()
	}
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
