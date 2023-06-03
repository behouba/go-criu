package crit

import (
	"io/ioutil"
	"regexp"
	"strconv"
	"testing"
)

func TestGetMemPages(t *testing.T) {
	dir := "test-imgs"
	pid := 0

	pagemapFilePattern := `pagemap-(\d+)\.img`

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v\n", err)
	}

	re := regexp.MustCompile(pagemapFilePattern)

	for _, file := range files {
		if re.MatchString(file.Name()) {
			numberStr := re.FindStringSubmatch(file.Name())[1]
			pid, err = strconv.Atoi(numberStr)
			if err != nil {
				t.Fatalf("Failed to convert number: %v\n", err)
			}
			break
		}
	}

	buff, err := GetMemPages(dir, pid)
	if err != nil {
		t.Errorf("GetMemPages returned an error: %v", err)
	}

	if len(buff.String()) == 0 {
		t.Error("Expected non-empty pages slice")
	}
}
