package anypkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

func readFile(filePath string, onEachLine func(line string)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error opening the file: %v", err))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		onEachLine(line)
	}

	if err := scanner.Err(); err != nil {
		return errors.New(fmt.Sprintf("Error reading the .package file: %v", err))
	}

	return nil
}

func Import() {
	fmt.Println("Executing import ...")
	readFile(".package", func(line string) {
		fmt.Println(line)
	})
}

func Sum() {
	fmt.Println("Executing sum ...")
}

func Clean() {
	fmt.Println("Executing clean ...")
}
