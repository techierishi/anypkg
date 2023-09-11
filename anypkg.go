package anypkg

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	counter int
	tmpdir  string
	url     string
	idir    string
	self    string
	wd      string
)
var fchecks map[string]string

func fcheck(file1, file2 string) bool {
	sum1, err := getFileChecksum(file1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating checksum for %s: %v\n", file1, err)
		return false
	}

	sum2, err := getFileChecksum(file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating checksum for %s: %v\n", file2, err)
		return false
	}

	if _, exists := fchecks[file2]; !exists || sum1 == sum2 {
		fchecks[file2] = sum2
		return true
	}

	return false
}

func getFileChecksum(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func cpc(srcDir, destDir string) {
	fileInfo, err := os.Stat(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing source directory %s: %v\n", srcDir, err)
		return
	}

	if !fileInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Source is not a directory: %s\n", srcDir)
		return
	}

	fileInfos, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source directory %s: %v\n", srcDir, err)
		return
	}

	for _, fileInfo := range fileInfos {
		srcPath := filepath.Join(srcDir, fileInfo.Name())
		destPath := filepath.Join(destDir, fileInfo.Name())

		if fileInfo.IsDir() {
			err := os.MkdirAll(destPath, os.ModePerm)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating destination directory %s: %v\n", destPath, err)
				return
			}
			cpc(srcPath, destPath)
		} else {
			if _, exists := fchecks[destPath]; exists {
				if fcheck(srcPath, destPath) {
					fmt.Fprintf(os.Stderr, "Contents of file differ between imports: %s\n", destPath)
					os.Exit(1)
				}
			}
			fchecks[destPath], _ = getFileChecksum(srcPath)
			copyFile(srcPath, destPath)
		}
	}
}

func curlx(url string) error {
	if strings.HasPrefix(url, "file://") {
		filePath := url[7:]
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		defer os.Chdir(wd)

		err = os.Chdir(filepath.Dir(filePath))
		if err != nil {
			return err
		}

		content, err := ioutil.ReadFile(filepath.Base(filePath))
		if err != nil {
			return err
		}

		fmt.Println(string(content))
	} else {
		response, err := http.Get(url)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request failed with status code: %d", response.StatusCode)
		}

		buffer := make([]byte, 4096)
		for {
			n, err := response.Body.Read(buffer)
			if n > 0 {
				_, writeErr := os.Stdout.Write(buffer[:n])
				if writeErr != nil {
					return writeErr
				}
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
		}
	}

	return nil
}

func rname(filePath string, wd string) (string, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	wdAbs, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFilePath, wdAbs) {
		return "", fmt.Errorf("file outside working directory")
	}

	relativePath := absFilePath[len(wdAbs)+1:]
	return relativePath, nil
}

func grepx(pattern string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		matched, err := regexp.MatchString(pattern, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if matched {
			fmt.Println(line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

func copyFile(src, dest string) {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening source file %s: %v\n", src, err)
		return
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating destination file %s: %v\n", dest, err)
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying content from %s to %s: %v\n", src, dest, err)
	}
}

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
	if len(args) < 3 || len(args) > 5 {
		fmt.Fprintln(os.Stderr, "Invalid usage: bad syntax for import")
	}
	log.Println("args", args)
	for idx, arg := range args {
		if arg == "->" {
			if idx+2 < len(args) {
				fmt.Fprintln(os.Stderr, "Invalid usage: missing output directory after '->'")
				os.Exit(1)
			}
			outdir = args[idx+1]
			if strings.Contains(outdir, "..") || strings.HasPrefix(outdir, "/") {
				fmt.Fprintf(os.Stderr, "Invalid output directory: %s\n", outdir)
				os.Exit(1)
			}
			break
		}
	}
	url, err := makeURL(args[1], args[2])
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fetch(args[0], url, outdir)
}

func makeURL(args ...string) (string, error) {
	argCount := len(args)
	if argCount < 1 {
		return "", fmt.Errorf("missing argument")
	}

	arg1 := args[0]

	switch {
	case strings.HasPrefix(arg1, "..") || strings.HasPrefix(arg1, "./") || strings.HasPrefix(arg1, "/"):
		return "file://" + arg1, nil
	case strings.Contains(arg1, "://"):
		return arg1, nil
	case strings.HasPrefix(arg1, "github.com/"):
		tag := arg1[11:]
		branch := "master"
		if argCount >= 2 {
			branch = args[1]
		}
		file := ".package"
		if argCount >= 3 {
			file = args[2]
		}
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", tag, branch, file), nil
	default:
		return "", fmt.Errorf("invalid location: '%s'", arg1)
	}
}
func fetch(cmd, url, outdir string) error {
	fmt.Printf("Fetching %s from %s to %s\n", cmd, url, outdir)
	hx := hexStr()
	tempDir := os.TempDir()
	tempFolderName := fmt.Sprintf("pkg%s", hx)
	tempFolder := filepath.Join(tempDir, tempFolderName)

	counter++

	pdir := filepath.Join(tempFolder, fmt.Sprintf("%d", counter))
	err := os.MkdirAll(tempFolder, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary folder: %s", err)
	}
	err = os.MkdirAll(pdir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create pdir folder: %s", err)
	}

	err = os.Chdir(pdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to directory %s: %v\n", pdir, err)
		os.Exit(1)
	}

	if filepath.Base(url) == ".package" {
		fmt.Fprintf(os.Stderr, "[get] %s\n", url)
		err = curlx(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching .package: %v\n", err)
		}

		packageContent, err := ioutil.ReadFile(".package")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading .package: %v\n", err)
		}

		lines := strings.Split(string(packageContent), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "file") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					fname := parts[1]
					url := filepath.Join(filepath.Dir(url), fname)
					dirname := filepath.Dir(fname)
					err := os.MkdirAll(dirname, os.ModePerm)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dirname, err)
						os.Exit(1)
					}

					if cmd == "clean" {
						fmt.Printf("%s/%s/%s\n", idir, outdir, fname)
						err := ioutil.WriteFile(fname, []byte(""), os.ModePerm)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error creating file %s: %v\n", fname, err)
							os.Exit(1)
						}
					} else {
						fmt.Fprintf(os.Stderr, "[get] %s\n", url)
						err := curlx(url)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", url, err)
							os.Exit(1)
						}
					}
				}
			}
		}

		fetch(cmd, url, outdir)
		err = os.Remove(".package")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error removing .package: %v\n", err)
		}
	} else if cmd == "clean" {
		fmt.Printf("%s/%s/%s\n", idir, outdir, filepath.Base(url))
		err := ioutil.WriteFile(filepath.Base(url), []byte(""), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file %s: %v\n", filepath.Base(url), err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[get] %s\n", url)
		err := curlx(filepath.Base(url))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", url, err)
			os.Exit(1)
		}
	}

	err = os.Chdir(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error changing back to working directory: %v\n", err)
		os.Exit(1)
	}

	if cmd != "clean" {
		err = os.MkdirAll(outdir, os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", outdir, err)
			os.Exit(1)
		}

		err = exec.Command("cp", "-r", pdir, outdir).Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying directory contents: %v\n", err)
			os.Exit(1)
		}
	}

	return nil
}

func hexStr() string {
	b := make([]byte, 4)
	rand.Read(b)
	hx := hex.EncodeToString(b)
	return hx
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
