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
