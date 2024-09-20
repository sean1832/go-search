package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// Custom structure to hold flag options
type Options struct {
	isFileOnly      bool
	isDirOnly       bool
	isCaseSensitive bool
	directory       string
	pattern         string
}

// ParseFlags parses the flags and positional arguments in any order
func ParseFlags(args []string) (*Options, error) {
	var opts Options
	var positionalArgs []string
	var program string = args[0]
	args = args[1:]
	for _, arg := range args {
		switch arg {
		case "-f", "--file":
			opts.isFileOnly = true
		case "-d", "--dir":
			opts.isDirOnly = true
		case "-c", "--casesensitive":
			opts.isCaseSensitive = true
		case "-h", "--help":
			displayHelp(program)
			os.Exit(0)
		default:
			// Collect positional arguments (directory and pattern)
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) != 2 {
		return nil, fmt.Errorf("invalid number of positional arguments")
	}

	opts.directory = positionalArgs[0]
	opts.pattern = positionalArgs[1]

	if opts.isFileOnly && opts.isDirOnly {
		return nil, fmt.Errorf("you cannot use both --fileonly and --dironly at the same time")
	}

	return &opts, nil
}

// Search function with additional flags
func Search(rootDir string, pattern string, isFileOnly bool, isDirOnly bool, isCaseSensitive bool) ([]string, error) {
	var matches []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Handle permission errors gracefully
			if pathErr, ok := err.(*os.PathError); ok {
				// Check if the error is an access denied error (on Windows)
				if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
					// Skip the directory we don't have permission to access
					fmt.Printf("Skipping: %s (Access Denied)\n", path)
					return nil
				}
			}
			// Return other types of errors
			fmt.Printf("Skipping: %s (Unhandle Error)\n", err)
			return nil
		}

		// Determine if we should skip based on file or directory flag
		if isFileOnly && d.IsDir() {
			return nil // Skip directories if isFileOnly is true
		}
		if isDirOnly && !d.IsDir() {
			return nil // Skip files if isDirOnly is true
		}

		wg.Add(1)
		go func(path string, d os.DirEntry) {
			defer wg.Done()

			baseName := filepath.Base(path)

			// Handle case sensitivity
			if !isCaseSensitive {
				baseName = strings.ToLower(baseName)
				pattern = strings.ToLower(pattern)
			}

			matched, err := filepath.Match(pattern, baseName)
			if err == nil && matched {
				mu.Lock()
				matches = append(matches, path)
				mu.Unlock()
			}
		}(path, d)

		return nil
	})

	wg.Wait()
	return matches, err
}

// displayHelp prints usage instructions
func displayHelp(program string) {
	fmt.Printf("Usage: %s <directory> <pattern> [OPTIONS]\n", program)
	fmt.Println("Options:")
	fmt.Println("  -f, --file        	 Only return files")
	fmt.Println("  -d, --dir         	 Only return directories")
	fmt.Println("  -c, --casesensitive    Make the search case-sensitive")
	fmt.Println("  -h, --help        	 Display this help message")
}

func main() {
	// Parse the flags and positional arguments manually
	opts, err := ParseFlags(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
		displayHelp(os.Args[0])
		os.Exit(1)
	}

	// Search for files or directories based on flags
	matches, err := Search(opts.directory, opts.pattern, opts.isFileOnly, opts.isDirOnly, opts.isCaseSensitive)
	if err != nil {
		fmt.Println("Error during file search:", err)
		return
	}

	// Output the results
	if len(matches) == 0 {
		fmt.Println("No path matches the pattern")
	} else {
		fmt.Println("Found Paths:")
		for _, match := range matches {
			fmt.Println(match)
		}
	}
}
