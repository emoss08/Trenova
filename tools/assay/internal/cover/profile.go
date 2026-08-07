package cover

import (
	"fmt"
	"io"
	"path"
	"sort"

	"golang.org/x/tools/cover"
)

type Block struct {
	StartLine int
	EndLine   int
}

func (b Block) Contains(line int) bool {
	return line >= b.StartLine && line <= b.EndLine
}

type FileBlocks struct {
	ProfileName string
	Blocks      []Block
}

type Resolver func(profileName string) (string, bool)

func ParseProfile(reader io.Reader) ([]FileBlocks, error) {
	profiles, err := cover.ParseProfilesFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("parse coverage profile: %w", err)
	}

	out := make([]FileBlocks, 0, len(profiles))
	for _, profile := range profiles {
		blocks := make([]Block, 0, len(profile.Blocks))
		for _, block := range profile.Blocks {
			if block.Count == 0 {
				continue
			}
			blocks = append(blocks, Block{StartLine: block.StartLine, EndLine: block.EndLine})
		}
		if len(blocks) == 0 {
			continue
		}
		out = append(out, FileBlocks{ProfileName: profile.FileName, Blocks: mergeBlocks(blocks)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ProfileName < out[j].ProfileName })

	return out, nil
}

func mergeBlocks(blocks []Block) []Block {
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].StartLine != blocks[j].StartLine {
			return blocks[i].StartLine < blocks[j].StartLine
		}

		return blocks[i].EndLine < blocks[j].EndLine
	})

	merged := blocks[:1]
	for _, block := range blocks[1:] {
		last := &merged[len(merged)-1]
		if block.StartLine <= last.EndLine+1 {
			if block.EndLine > last.EndLine {
				last.EndLine = block.EndLine
			}

			continue
		}
		merged = append(merged, block)
	}

	return merged
}

func SplitProfileName(profileName string) (importPath, fileName string) {
	dir, file := path.Split(profileName)
	if dir == "" {
		return "", file
	}

	return path.Clean(dir), file
}
