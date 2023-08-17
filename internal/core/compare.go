package core

import (
	"fmt"
	"os"

	cfg "github.com/1F47E/go-bitreel/internal/config"
)

// encode + decode + compare
func (c *Core) Compare(filename string) (bool, error) {
	defer os.Remove(cfg.PathVideoOut)

	err := c.Encode(filename)
	if err != nil {
		return false, err
	}
	out, err := c.Decode(cfg.PathVideoOut)
	if err != nil {
		return false, err
	}
	defer os.Remove(out)
	// compare files
	same, err := compareFiles(filename, out)
	if err != nil {
		return false, err
	}
	// assert if not same
	if !same {
		return false, fmt.Errorf("Error: files are not the same")
	} else {
		return true, nil
	}
}

// Compare files before and after decoding for test command
func compareFiles(file1, file2 string) (bool, error) {
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
		return false, fmt.Errorf("Files are not the same size")
	}
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			return false, fmt.Errorf("Files are not the same at position %d", i)
		}
	}
	return true, nil
}
