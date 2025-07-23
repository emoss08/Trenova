// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package rptmeta

// func LoadReportsFromDirectory(dir string) ([]*Metadata, error) {
// 	var reports []*Metadata

// 	files, err := os.ReadDir(dir)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, file := range files {
// 		if strings.HasSuffix(file.Name(), ".rptmeta.yaml") {
// 			report, rErr := Parse(filepath.Join(
// 				dir,
// 				file.Name(),
// 			))
// 			if rErr != nil {
// 				return nil, rErr
// 			}
// 			reports = append(reports, report)
// 		}
// 	}

// 	return reports, nil
// }
