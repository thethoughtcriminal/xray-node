package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func isTerminal() bool {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return true
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func promptReader() *bufio.Reader {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return bufio.NewReader(os.Stdin)
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return bufio.NewReader(os.Stdin)
	}
	return bufio.NewReader(f)
}

func promptString(label, defaultValue string) (string, error) {
	reader := promptReader()
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func promptInt(label string, defaultValue int) (int, error) {
	reader := promptReader()
	fmt.Printf("%s [%d]: ", label, defaultValue)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", label)
	}
	return value, nil
}
