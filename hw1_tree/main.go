package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type sortableFileInfoArray []os.FileInfo

func (nf sortableFileInfoArray) Len() int      { return len(nf) }
func (nf sortableFileInfoArray) Swap(i, j int) { nf[i], nf[j] = nf[j], nf[i] }
func (nf sortableFileInfoArray) Less(i, j int) bool {
    return nf[i].Name() < nf[j].Name()
}

func selectOnlyDirs(filesOrDirs []os.FileInfo) (result []os.FileInfo) {
	for _, fileOrDir := range filesOrDirs {
		if fileOrDir.IsDir() {
			result = append(result, fileOrDir)
		}
	}
	return result
}

func getFileSizeStr(fileInfo os.FileInfo) string {
	size := fileInfo.Size()
	if size == 0 {
		return "empty"
	}

	return fmt.Sprintf("%db", size)
}

func processDir(out io.Writer, path string, printFiles bool, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	fileInfos, err := file.Readdir(0)
	if err != nil {
		return err
	}

	if !printFiles {
		fileInfos = selectOnlyDirs(fileInfos)
	}

	sort.Sort(sortableFileInfoArray(fileInfos))

	maxI := len(fileInfos) - 1
	for i := 0; i <= maxI; i++ {
		fileInfo := fileInfos[i]
		var (
			localPrefix string
			prefixForSubdirs string
		)
		if i == maxI {
			localPrefix = "└───"
			prefixForSubdirs = "\t"
		} else {
			localPrefix = "├───"
			prefixForSubdirs = "│\t"
		}

		if fileInfo.IsDir() {
			fmt.Fprintf(out, "%s%s%s\n", prefix, localPrefix, fileInfo.Name())
			err = processDir(out, filepath.Join(path, fileInfo.Name()), printFiles, prefix + prefixForSubdirs)
			if err != nil {
				return err
			}
		} else {
			if printFiles {
				fmt.Fprintf(out, "%s%s%s (%s)\n", prefix, localPrefix, fileInfo.Name(), getFileSizeStr(fileInfo))
			}
		}
	}


	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode:= stat.Mode()
	if mode.IsDir() {
		err = processDir(out, path, printFiles, "")
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintln(out, "It is file:", path)
	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
