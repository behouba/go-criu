package crit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/checkpoint-restore/go-criu/v6/crit/images"
)

const (
	PageSize = 4096
)

// This can be a substitute name to PsTree since it will be used both
// for listing and getting process tree
type Process struct {
	PId      uint32              `json:"pId"`
	PgId     uint32              `json:"pgId"`
	SId      uint32              `json:"sId"`
	Comm     string              `json:"comm"`
	Process  *images.PstreeEntry `json:"-"`
	Core     *images.CoreEntry   `json:"-"`
	Children []*PsTree           `json:"children,omitempty"`
}

// The "dir" argument present in functions below is the path to 
// the "checkpoint/" director where the container checkpoint is extracted.

// This function will return the process tree from the specified pid.
// Will return the entire process tree if pid is 0
// NOTE: Is this function needed since it will do pretty much the same thing like crit.ExplorePs().
func GetPsTree(dir string, pid int) (tree *Process, err error) {
	panic("not implemented yet")
}

// This function will return a list of process with specified pids.
// Will return the list of all processes if not pid is specified
func GetPsList(dir string, pids []int) (ps []Process, err error) {
	panic("not implemented yet")
}

// This function will return a list of memory map of a process or processes.
// We may add additionnal fields to MemMap struct.
// NOTE: crit.ExploreMems() method do something similar for all processes
func GetMemMaps(dir string, pids []int) (maps []MemMap, err error) {
	panic("not implemented yet")
}

// This function will read the memory address `env_start` and `env_end` of a process,
// then, read this address space from memory pages and return the process arguments.
func GetArguments(dir string, pid int) (args []string, err error) {
	panic("not implemented yet")

}

type EnvVar struct {
	Name  string
	Value string
}

// This function will return the environment variables of a process.
// (to achieve this we can read memory address `mm_arg_start` and `mm_arg_en) .
func GetEnvVars(dir string, pid int) (envVars []EnvVar, err error) {
	panic("not implemented yet")

}

// This function will return a list of file opened file descriptors.
func GetFds(dir string, pids []int) (fds []Fd, err error) {
	panic("not implemented yet")

}

type HistEntry struct{}

// This function will try to retrieve bash history from memory pages.
func GetBashHistory(dir string, pids []int) (entries []HistEntry, err error) {
	panic("not implemented yet")
}

type Socket struct{}

// This function will return socket file associated with the specified processes ids.
// If no pid is provided will return all sockets of the container.
func GetSockets(dir string, pids []int) (sockets []Socket, err error) {
	panic("not implemented yet")
}

type MountPoint struct{}

// This function will return a list of mount points of the container namespace
func GetMountPoints(dir string) (mnts []MountPoint, err error) {
	panic("not implemented yet")
}

// Attempt to retreive memory segment of a process
func GetMemPages(dir string, pid, start, end uint64) (pages []byte, err error) {
	// Get physical memory addresses
	pagemapImg, err := getImg(filepath.Join(dir, fmt.Sprintf("pagemap-%d.img", pid)))
	if err != nil {
		return nil, err
	}

	pagesId := pagemapImg.Entries[0].Message.(*images.PagemapHead).GetPagesId()

	pagemapEntries := make([]*images.PagemapEntry, 0)

	for _, entry := range pagemapImg.Entries {
		pagemapEntries = append(pagemapEntries, entry.Message.(*images.PagemapEntry))
	}

	size := end - start

	if size == 0 {
		return []byte{}, nil
	}

	f, err := os.Open(fmt.Sprintf("pages-%d.img", pagesId))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	startPage := start / PageSize
	endPage := end / PageSize

	buf := make([]byte, 0)

	for pageNumber := startPage; pageNumber <= endPage; pageNumber++ {
		var page []byte = nil

		pageMem, err := getPage(pageNumber, pagemapEntries)
		if err != nil {
			return nil, err
		}

		if f != nil {
			page = make([]byte, PageSize)
			f.Read(page)
			fmt.Println(page)
		}

		if pageMem != nil {
			page = pageMem
		}

		if pageMem == nil {
			page = bytes.Repeat([]byte("\x00"), int(PageSize))
		}

		var nSkip, nRead uint64

		if pageNumber == startPage {
			nSkip = start - pageNumber*PageSize
			if startPage == endPage {
				nRead = size
			} else {
				nRead = PageSize - nSkip
			}
		} else if pageNumber == endPage {
			nSkip = 0
			nRead = end - pageNumber*PageSize
		} else {
			nSkip = 0
			nRead = PageSize
		}

		buf = append(pages, page[nSkip:nSkip+nRead]...)
	}

	return buf, nil

}

func getPage(pgStart uint64, pagemap []*images.PagemapEntry) ([]byte, error) {

	var off uint64 = 0

	for _, m := range pagemap {
		found := false

		for i := 0; i < int(*m.NrPages); i++ {

			if *m.Vaddr+uint64(i)*PageSize == pgStart*PageSize {
				found = true
				break
			}
			off += 1
		}

		if !found {
			continue
		}

		f, err := os.Open("pages-4.img")
		if err != nil {
			return nil, err
		}

		_, err = f.Seek(int64(off*PageSize), 0)
		if err != nil {
			return nil, err
		}

		buff := make([]byte, PageSize)

		_, err = f.Read(buff)
		if err != nil {
			return nil, err
		}

		return buff, nil

	}

	return nil, nil
}
