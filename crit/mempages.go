package crit

import (
	"bytes"
	"fmt"
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

// GetMemPages retrieves memory pages associated with a pid.
func GetMemPages(dir string, pid int) ([]byte, error) {
	mmImg, err := getImg(filepath.Join(dir, fmt.Sprintf("mm-%d.img", pid)), &mm.MmEntry{})
	if err != nil {
		return nil, err
	}

	vmas := mmImg.Entries[0].Message.(*mm.MmEntry).GetVmas()

	var buff bytes.Buffer
	for _, vma := range vmas {
		size := *vma.End - *vma.Start
		pages, err := generateMemoryChunk(dir, pid, vma, size)
		if err != nil {
			return nil, err
		}
		buff.Write(pages)
	}

	return buff.Bytes(), nil
}

// generateMemoryChunk generates the memory chunk from a given VMA.
func generateMemoryChunk(dir string, pid int, vma *vma.VmaEntry, size uint64) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}

	// TODO: Is this OK ? since we are in the context of container
	if *vma.Status&vmaAreaVvar != 0 {
		return bytes.Repeat([]byte("\x00"), int(pageSize)), nil
	} else if *vma.Status&vmaAreaVsyscall != 0 {
		return bytes.Repeat([]byte("\x00"), int(pageSize)), nil
	}

	pagemapImg, err := getImg(filepath.Join(dir, fmt.Sprintf("pagemap-%d.img", pid)), &pagemap.PagemapHead{})
	if err != nil {
		return nil, err
	}

	pagesID := pagemapImg.Entries[0].Message.(*pagemap.PagemapHead).GetPagesId()

	pagemapEntries := make([]*pagemap.PagemapEntry, 0)

	for _, entry := range pagemapImg.Entries[1:] {
		pagemapEntries = append(pagemapEntries, entry.Message.(*pagemap.PagemapEntry))
	}

	pagesFile, err := os.Open(filepath.Join(dir, fmt.Sprintf("pages-%d.img", pagesID)))
	if err != nil {
		return nil, err
	}

	defer pagesFile.Close()

	start := *vma.Start
	end := *vma.Start + size

	startPage := start / pageSize
	endPage := end / pageSize

	var buff bytes.Buffer

	for pageNo := startPage; pageNo <= endPage; pageNo++ {
		var pageData []byte

		pageMem, err := getPage(dir, int(pagesID), pageNo, pagemapEntries)
		if err != nil {
			return nil, err
		}

		if pagesFile != nil {
			pageData = make([]byte, pageSize)
			_, err := pagesFile.Read(pageData)
			if err != nil {
				pageData = bytes.Repeat([]byte("\x00"), int(pageSize))
			}
		}

		if pageMem != nil {
			pageData = pageMem
		}

		if pageMem == nil {
			pageData = bytes.Repeat([]byte("\x00"), int(pageSize))
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
	return buff.Bytes(), nil
}

// getPage try to retrieves the page data for a given page number.
func getPage(dir string, pagesID int, pageNo uint64, pagemapEntries []*pagemap.PagemapEntry) ([]byte, error) {
	var off uint64 = 0

	for _, m := range pagemapEntries {
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

		f, err := os.Open(filepath.Join(dir, fmt.Sprintf("pages-%d.img", pagesID)))
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
		return buff, nil
	}
	return nil, nil
}
