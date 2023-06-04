package crit

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/checkpoint-restore/go-criu/v6/crit/images/mm"
	"github.com/checkpoint-restore/go-criu/v6/crit/images/pagemap"
	"github.com/checkpoint-restore/go-criu/v6/crit/images/vma"
)

const (
	pageSize = 4096

	vmaAreaVvar     = 1 << 12 // VMA_AREA_VVAR
	vmaAreaVsyscall = 1 << 2  // VMA_AREA_VSYSCALL
)

/*
Use cases:
- export memory mmpas content of a process
- list process arguments
- list command-line arguments on the process
- list bash history
- list process envvar
- search
- showing the content of memory (from the pages image) alongside the coresponding addresses (from the pagemap image) might be useful.
*/

type MemoryAnalyzer struct {
	checkpointDir  string
	pid            int
	pagesID        uint32
	pagemapEntries []*pagemap.PagemapEntry
	vmas           []*vma.VmaEntry
}

// NewMemoryAnalyzer creates a new MemoryAnalyzer instance with all the field populated with neccessary data
func NewMemoryAnalyzer(checkpointDir string, pid int) (*MemoryAnalyzer, error) {
	pagemapImg, err := getImg(filepath.Join(checkpointDir, fmt.Sprintf("pagemap-%d.img", pid)), &pagemap.PagemapHead{})
	if err != nil {
		return nil, err
	}

	pagesID := pagemapImg.Entries[0].Message.(*pagemap.PagemapHead).GetPagesId()

	pagemapEntries := make([]*pagemap.PagemapEntry, 0)

	for _, entry := range pagemapImg.Entries[1:] {
		pagemapEntries = append(pagemapEntries, entry.Message.(*pagemap.PagemapEntry))
	}

	mmImg, err := getImg(filepath.Join(checkpointDir, fmt.Sprintf("mm-%d.img", pid)), &mm.MmEntry{})
	if err != nil {
		return nil, err
	}

	return &MemoryAnalyzer{
		checkpointDir:  checkpointDir,
		pid:            pid,
		pagesID:        pagesID,
		pagemapEntries: pagemapEntries,
		vmas:           mmImg.Entries[0].Message.(*mm.MmEntry).GetVmas(),
	}, nil
}

func (ma *MemoryAnalyzer) getVmas() ([]*vma.VmaEntry, error) {
	mmImg, err := getImg(filepath.Join(ma.checkpointDir, fmt.Sprintf("mm-%d.img", ma.pid)), &mm.MmEntry{})
	if err != nil {
		return nil, err
	}
	return mmImg.Entries[0].Message.(*mm.MmEntry).GetVmas(), nil

}

// GetMemPages retrieves memory pages associated with a pid.
func GetMemPages(dir string, pid int) (*bytes.Buffer, error) {
	analyzer, err := NewMemoryAnalyzer(dir, pid)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	for _, vma := range analyzer.vmas {
		size := *vma.End - *vma.Start
		chunk, err := analyzer.GenerateMemoryChunk(vma, size)
		if err != nil {
			return nil, err
		}
		buff.Write(chunk.Bytes())
	}

	return &buff, nil
}

func (ma *MemoryAnalyzer) getVma(addr uint64) (*vma.VmaEntry, error) {

	for i, vma := range ma.vmas {
		if *vma.Start >= addr && addr <= *vma.End {

			chunk, err := ma.GenerateMemoryChunk(vma, *vma.End-*vma.Start)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Memory chunk: ", chunk.String())
			return ma.vmas[i], nil
		}
	}

	return nil, fmt.Errorf("Vma not found for address: %d", addr)
}

// GenerateMemoryChunk generates the memory chunk from a given VMA.
func (ma *MemoryAnalyzer) GenerateMemoryChunk(vma *vma.VmaEntry, size uint64) (*bytes.Buffer, error) {
	if size == 0 {
		return nil, nil
	}

	var buff bytes.Buffer

	// TODO: Is this OK ? since we are in the context of container
	if *vma.Status&vmaAreaVvar != 0 || *vma.Status&vmaAreaVsyscall != 0 {
		buff.Write(getZeroedPage(1))
		return &buff, nil
	}

	pagesFile, err := os.Open(filepath.Join(ma.checkpointDir, fmt.Sprintf("pages-%d.img", ma.pagesID)))
	if err != nil {
		return nil, err
	}

	defer pagesFile.Close()

	start := *vma.Start
	end := *vma.Start + size

	startPage := start / pageSize
	endPage := end / pageSize

	for pageNo := startPage; pageNo <= endPage; pageNo++ {
		var pageData []byte

		pageMem, err := ma.getPage(pageNo)
		if err != nil {
			return nil, err
		}

		if pagesFile != nil {
			pageData = make([]byte, pageSize)
			_, err := pagesFile.Read(pageData)
			if err != nil {
				continue
			}
		}

		if pageMem != nil {
			pageData = pageMem
		}

		if pageMem == nil {
			pageData = getZeroedPage(1)
		}

		var nSkip, nRead uint64

		if pageNo == startPage {
			nSkip = start - pageNo*pageSize
			if startPage == endPage {
				nRead = size
			} else {
				nRead = pageSize - nSkip
			}
		} else if pageNo == endPage {
			nSkip = 0
			nRead = end - pageNo*pageSize
		} else {
			nSkip = 0
			nRead = pageSize
		}

		buff.Write(pageData[nSkip : nSkip+nRead])
	}
	return &buff, nil
}

// getPage try to retrieves the page data for a given page number from the vma.
func (ma *MemoryAnalyzer) getPage(pageNo uint64) ([]byte, error) {
	var off uint64 = 0

	for _, m := range ma.pagemapEntries {
		found := false

		for i := 0; i < int(*m.NrPages); i++ {
			if *m.Vaddr+uint64(i)*pageSize == pageNo*pageSize {
				found = true
				break
			}
			off += 1
		}

		if !found {
			continue
		}

		f, err := os.Open(filepath.Join(ma.checkpointDir, fmt.Sprintf("pages-%d.img", ma.pagesID)))
		if err != nil {
			return nil, err
		}

		defer f.Close()

		_, err = f.Seek(int64(off*pageSize), 0)
		if err != nil {
			return nil, err
		}

		buff := make([]byte, pageSize)

		_, err = f.Read(buff)
		if err != nil {
			return nil, err
		}
		// fmt.Println(buff[:16])
		return buff, nil
	}
	return nil, nil
}

func getZeroedPage(nrPages int) []byte {
	if nrPages < 1 {
		nrPages = 1
	}
	return bytes.Repeat([]byte("\x00"), int(pageSize*nrPages))
}
