package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	out := os.Stdout

	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}

	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f" // nolint: gomnd

	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, files bool) error {
	builder := newTreeBuilder(path, files)

	resultTree, err := builder.buildTree()
	if err != nil {
		return err
	}

	treeBytes := []byte(resultTree.String())

	_, err = out.Write(treeBytes)
	if err != nil {
		return fmt.Errorf("failed to write tree bytes to the output: %w", err)
	}

	return nil
}

type treeBuilder struct {
	addFiles bool
	rootPath string
}

func newTreeBuilder(rootPath string, addFiles bool) *treeBuilder {
	return &treeBuilder{
		addFiles: addFiles,
		rootPath: rootPath,
	}
}

type tree []treeElement

type treeElement struct {
	parentElement *treeElement

	fileInfo os.FileInfo

	isLast bool

	nestedElements tree
}

func (b treeBuilder) buildTree() (tree, error) {
	return b.buildTreeRecursively(b.rootPath, nil)
}

func (b treeBuilder) buildTreeRecursively(fullPath string, parentElement *treeElement) (tree, error) {
	files, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir from fullPath '%s': %w", fullPath, err)
	}

	if !b.addFiles {
		files = removeFiles(files)
	}

	resultTree := make(tree, 0, len(files))

	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for i := range files {
		treeEl := treeElement{
			parentElement: parentElement,
			fileInfo:      files[i],
			isLast:        i == len(files)-1,
		}

		if treeEl.fileInfo.IsDir() {
			newFullPath := filepath.Join(fullPath, treeEl.fileInfo.Name())

			treeEl.nestedElements, err = b.buildTreeRecursively(newFullPath, &treeEl)
			if err != nil {
				return nil, err
			}
		}

		resultTree = append(resultTree, treeEl)
	}

	return resultTree, nil
}

func removeFiles(files []os.FileInfo) []os.FileInfo {
	var result []os.FileInfo

	for i := range files {
		if files[i].IsDir() {
			result = append(result, files[i])
		}
	}

	return result
}

func (t tree) String() string {
	var result string

	for i := range t {
		result += t[i].String()
	}

	return result
}

func (el treeElement) String() string {
	var result string

	nestingLevel := el.getNestingLevel()
	numberOfNotLastParents := el.getNumberOfNotLastParents()

	for i := 0; i < nestingLevel; i++ {
		if i < numberOfNotLastParents {
			result += "│"
		}

		result += "\t"
	}

	if el.isLast {
		result += "└───"
	} else {
		result += "├───"
	}

	result += el.fileInfo.Name()

	if !el.fileInfo.IsDir() {
		result += " "

		var size string

		if el.fileInfo.Size() == 0 {
			size = "(empty)"
		} else {
			size = fmt.Sprintf("(%db)", el.fileInfo.Size())
		}

		result += size
	}

	result += "\n"

	for i := range el.nestedElements {
		result += el.nestedElements[i].String()
	}

	return result
}

func (el treeElement) getNestingLevel() int {
	return el.getNestingLevelRecursively(0)
}

func (el treeElement) getNestingLevelRecursively(level int) int {
	if el.parentElement == nil {
		return level
	}

	return el.parentElement.getNestingLevelRecursively(level + 1)
}

func (el treeElement) getNumberOfNotLastParents() int {
	return el.getNumberOfNotLastParentsRecursively(0)
}

func (el treeElement) getNumberOfNotLastParentsRecursively(num int) int {
	if el.parentElement == nil {
		return num
	}

	if !el.parentElement.isLast {
		num++
	}

	return el.parentElement.getNumberOfNotLastParentsRecursively(num)
}
