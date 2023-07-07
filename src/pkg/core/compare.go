package core

import (
	"fmt"
	"os"
)

// Compare 2 files
func (c *Core) Compare(file1, file2 string) (bool, error) {
	fmt.Println("Comparing files...")
	// read files
	b1, err := os.ReadFile(file1)
	if err != nil {
		return false, err
	}
	b2, err := os.ReadFile(file2)
	if err != nil {
		return false, err
	}
	// compare
	if len(b1) != len(b2) {
		fmt.Println("Files are not the same size")
		return false, nil
	}
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			fmt.Println("Files are not the same at position", i)
			return false, nil
		}
	}
	return true, nil
}
