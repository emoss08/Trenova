package rptmetavalidator_test

// type MetadataValidatorTestSuite struct {
// 	testutils.BaseSuite
// 	validator *rptmetavalidator.MetadataValidator
// }

// func TestMetadataValidatorSuite(t *testing.T) {
// 	suite.Run(t, new(MetadataValidatorTestSuite))
// }

// func (s *MetadataValidatorTestSuite) SetupTest() {
// 	s.BaseSuite.LoadTestDB()
// 	s.validator = s.Validators.RptMetaValidator
// }

// func (s *MetadataValidatorTestSuite) createValidMetadata() *rptmeta.Metadata {
// 	return &rptmeta.Metadata{
// 		SQL: "SELECT * FROM workers WHERE status = ${workerStatus}",
// 		Variables: []*rptmeta.Variable{
// 			{
// 				Name:          "workerStatus",
// 				Placeholder:   "${workerStatus}",
// 				Type:          "string",
// 				Default:       "Active",
// 				Description:   "Worker ID",
// 				IsRequired:    true,
// 				AllowedValues: []string{"Active", "Inactive", "Suspended"},
// 			},
// 		},
// 		Report: &rptmeta.Report{
// 			Title:       "Test Report",
// 			Description: "This is a test report",
// 			Tags:        []string{"test"},
// 			Version:     1,
// 			Caching: &rptmeta.Caching{
// 				IsCachable:    true,
// 				CacheDuration: 3600,
// 			},
// 			Scheduling: &rptmeta.Scheduling{
// 				IsScheduled: true,
// 				Schedule:    "0 0 * * *",
// 			},
// 		},
// 	}
// }

// func (s *MetadataValidatorTestSuite) TestValidateSQLStructure() {
// 	scenarios := []struct {
// 		name           string
// 		modifyMetadata func(*rptmeta.Metadata)
// 		expectErrors   []struct {
// 			Field   string
// 			Code    errors.ErrorCode
// 			Message string
// 		}
// 	}{
// 		{
// 			name: "sql_query_is_empty",
// 			modifyMetadata: func(m *rptmeta.Metadata) {
// 				m.SQL = ""
// 			},
// 			expectErrors: []struct {
// 				Field   string
// 				Code    errors.ErrorCode
// 				Message string
// 			}{
// 				{Field: "sql", Code: errors.ErrRequired, Message: "SQL is required"},
// 				{Field: "sql", Code: errors.ErrInvalid, Message: "SQL query must contain a SELECT statement"},
// 			},
// 		},
// 		{
// 			name: "sql_query_does_not_contain_select",
// 			modifyMetadata: func(m *rptmeta.Metadata) {
// 				m.SQL = "UPDATE workers SET status = 'Active' WHERE id = ${workerStatus}"
// 			},
// 			expectErrors: []struct {
// 				Field   string
// 				Code    errors.ErrorCode
// 				Message string
// 			}{
// 				{Field: "sql", Code: errors.ErrInvalid, Message: "SQL query must contain a SELECT statement"},
// 			},
// 		},
// 		{
// 			name: "sql_query_has_unbalanced_parentheses",
// 			modifyMetadata: func(m *rptmeta.Metadata) {
// 				m.SQL = "SELECT * FROM workers WHERE status = ${workerStatus} AND (id = ${workerId}"
// 			},
// 			expectErrors: []struct {
// 				Field   string
// 				Code    errors.ErrorCode
// 				Message string
// 			}{
// 				{Field: "sql", Code: errors.ErrInvalid, Message: "SQL query has unbalanced parentheses"},
// 			},
// 		},
// 		{
// 			name: "sql_query_contains_sql_injection",
// 			modifyMetadata: func(m *rptmeta.Metadata) {
// 				m.SQL = "SELECT * FROM workers WHERE status = ${workerStatus}; DROP TABLE workers"
// 			},
// 			expectErrors: []struct {
// 				Field   string
// 				Code    errors.ErrorCode
// 				Message string
// 			}{
// 				{Field: "sql", Code: errors.ErrInvalid, Message: "Potential SQL injection detected"},
// 			},
// 		},
// 	}

// 	for _, tt := range scenarios {
// 		s.Run(tt.name, func() {
// 			metadata := s.createValidMetadata()
// 			if tt.modifyMetadata != nil {
// 				tt.modifyMetadata(metadata)
// 			}

// 			multiErr := s.validator.Validate(metadata)

// 			matcher := testutils.NewErrorMatcher(s.T(), multiErr)
// 			matcher.HasExactErrors(tt.expectErrors)
// 		})
// 	}
// }
