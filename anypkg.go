package anypkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func readFile(filePath string, onEachLine func(lineText string, lineNumber int)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error opening the file: %v", err))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	lineNumber := 1
	for scanner.Scan() {
		lineText := scanner.Text()
		onEachLine(lineText, lineNumber)

		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return errors.New(fmt.Sprintf("Error reading the .package file: %v", err))
	}

	return nil
}

func execImport(lineText string, lineNumber int) {
	outdir := "."

	args := strings.Split(lineText, " ")
	i := 0

	for i < len(args) {
		arg := args[i]
		if arg == "->" {
			if i+2 >= len(args) {
				fmt.Fprintln(os.Stderr, "Invalid usage: missing output directory after '->'")
				os.Exit(1)
			}
			outdir = args[i+2]
			if strings.Contains(outdir, "..") || strings.HasPrefix(outdir, "/") {
				fmt.Fprintf(os.Stderr, "Invalid output directory: %s\n", outdir)
				os.Exit(1)
			}
			args = args[:i]
			break
		}
		i++
	}

	url := mkURL(args[0], args[1], args[2])
	fetch("import", url, outdir)
}

func mkURL(parts ...string) string {
	return strings.Join(parts, "/")
}

func fetch(cmd, url, outdir string) {
	fmt.Printf("Fetching %s from %s to %s\n", cmd, url, outdir)
}

func execFile(lineText string, lineNumber int) {

}

func processLine(lineText string, lineNumber int) {
	if strings.HasPrefix(strings.Trim(lineText, " "), "import") {
		execImport(lineText, lineNumber)
	} else if strings.HasPrefix(strings.Trim(lineText, " "), "file") {

	} else if strings.HasPrefix(strings.Trim(lineText, " "), "sum") {

	} else {
		if strings.Trim(lineText, " ") == "" {
			return
		}
		fmt.Fprintf(os.Stderr, "line %d: unknown directive:\n", lineNumber)
		os.Exit(1)
	}
}
func Import() {
	fmt.Println("Executing import ...")
	readFile(".package", func(lineText string, lineNumber int) {
		processLine(lineText, lineNumber)
	})
}

func Sum() {
	fmt.Println("Executing sum ...")
}

func Clean() {
	fmt.Println("Executing clean ...")
}
